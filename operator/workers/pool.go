// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workers

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Netcracker/qubership-kafka/operator/cfg"
	"github.com/Netcracker/qubership-kafka/operator/workers/jobs"
	"github.com/go-logr/logr"
)

type Pool struct {
	opts                   cfg.Cfg
	log                    logr.Logger
	ctx                    context.Context
	jbs                    []jobs.Job
	wg                     sync.WaitGroup
	stop                   context.CancelFunc
	maxConsecutiveRestarts int
	restartResetAfter      time.Duration
	exitFn                 func()
}

func NewPool(ctx context.Context, stop context.CancelFunc, opts cfg.Cfg, logger logr.Logger) *Pool {
	return &Pool{
		opts: opts,
		log:  logger,
		ctx:  ctx,
		stop: stop,
		jbs: []jobs.Job{
			jobs.KafkaJob{},
			jobs.AkhqJob{},
			jobs.KmmJob{},
			jobs.KafkaUserJob{},
		},
		maxConsecutiveRestarts: 5,
		restartResetAfter:      60 * time.Minute,
	}
}

func (wrk *Pool) Start() error {
	wrk.log.Info("Starting workers")

	for _, j := range wrk.jbs {
		runJob, runDuplicate := j.Enabled(wrk.opts)
		if !runJob {
			wrk.log.V(-1).Info(fmt.Sprintf("Job %T is not supported", j))
			continue
		}
		wrk.launchJob(j, wrk.opts.ApiGroup)
		if sg := wrk.opts.SecondaryApiGroup; sg != "" && runDuplicate {
			wrk.launchJob(j, sg)
		}

	}
	return nil
}

func (wrk *Pool) Wait() error {
	wrk.wg.Wait()
	return nil
}

func (wrk *Pool) launchJob(job jobs.Job, apiGroup string) {
	wrk.wg.Add(1)
	go func() {
		defer wrk.wg.Done()

		jobName := fmt.Sprintf("%T[%s]", job, apiGroup)
		log := wrk.log.WithValues("job", jobName)

		var consecFails int
		lastFail := time.Now()
		for {
			select {
			case <-wrk.ctx.Done():
				log.Info("shutting down received, stopping job")
				return
			default:
			}

			jobCtx, cancel := context.WithCancel(wrk.ctx)

			exe, err := job.Build(jobCtx, wrk.opts, apiGroup, log)
			if err != nil {
				log.Error(err, "build failed", "attempt", consecFails)
				cancel()
				return
			}

			func() {
				defer cancel()
				var runErr error
				runErr = exe()
				switch {
				case jobCtx.Err() != nil:
					return
				case runErr == nil:
					log.Error(jobs.UnexpectedError, "job finished unexpectedly without error; restarting", "attempt", consecFails)
				default:
					log.Error(runErr, "job failed; restarting", "attempt", consecFails)
					consecFails++
				}
			}()

			select {
			case <-wrk.ctx.Done():
				return
			default:
			}

			if time.Since(lastFail) > wrk.restartResetAfter {
				consecFails = 1
				lastFail = time.Now()
			}

			if consecFails >= wrk.maxConsecutiveRestarts {
				log.Error(fmt.Errorf("too many consecutive failures"),
					"triggering global shutdown via context cancel",
					"consecutiveFails", consecFails,
					"resetAfter", wrk.restartResetAfter)
				wrk.stop()
				return
			}

			sleep := backoffWithJitter(consecFails, 30*time.Second)
			timer := time.NewTimer(sleep)
			select {
			case <-timer.C:
			case <-wrk.ctx.Done():
				timer.Stop()
				return
			}
		}
	}()
}

func backoffWithJitter(consecFails int, capDur time.Duration) time.Duration {
	if consecFails < 1 {
		consecFails = 1
	}
	d := time.Second << (consecFails - 1)
	if d > capDur {
		d = capDur
	}
	j := time.Duration(rand.Int63n(int64(d) / 3))
	if rand.Intn(2) == 0 {
		return d - j
	}
	return d + j
}
