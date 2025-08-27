package workers

import (
	"context"
	"fmt"
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Netcracker/qubership-kafka/operator/cfg"
	"github.com/Netcracker/qubership-kafka/operator/workers/jobs"
	"github.com/go-logr/logr"
)

type Pool struct {
	opts cfg.Cfg
	log  logr.Logger
	ctx  context.Context
	jbs  []jobs.Job
	wg   sync.WaitGroup
}

func NewPool(ctx context.Context, opts cfg.Cfg, logger logr.Logger) *Pool {
	return &Pool{
		opts: opts,
		log:  logger,
		ctx:  ctx,
		jbs: []jobs.Job{
			jobs.KafkaJob{},
			jobs.AkhqJob{},
			jobs.KmmJob{},
			jobs.KafkaUserJob{},
		},
	}
}

func (wrk *Pool) Start() error {
	wrk.log.Info("Starting workers")

	for _, j := range wrk.jbs {
		wrk.launchJob(j, wrk.opts.ApiGroup)
		if sg := wrk.opts.SecondaryApiGroup; sg != "" {
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

		attempt := 0
		for {
			select {
			case <-wrk.ctx.Done():
				log.Info("shutting down received, stopping job")
				return
			default:
			}

			attempt++

			jobCtx, cancel := context.WithCancel(wrk.ctx)
			func() {
				defer cancel()

				var runErr error
				defer func() {
					if r := recover(); r != nil {
						runErr = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
					}
				}()

				exe, err := job.Build(jobCtx, wrk.opts, apiGroup, log)
				if err != nil {
					log.Error(err, "build failed", "attempt", attempt)
					return
				}
				if exe == nil {
					log.Info("build returned nil exec; finishing", "attempt", attempt)
					return
				}

				runErr = exe()

				switch {
				case jobCtx.Err() != nil:
					return
				case runErr == nil:
					log.Info("exec finished unexpectedly without error; restarting", "attempt", attempt)
				default:
					log.Error(runErr, "exec failed; restarting", "attempt", attempt)
				}
			}()

			select {
			case <-wrk.ctx.Done():
				return
			default:
			}

			sleep := backoffWithJitter(attempt, 30*time.Second)
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

func backoffWithJitter(attempt int, capDur time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := time.Second << (attempt - 1)
	if d > capDur {
		d = capDur
	}
	j := time.Duration(rand.Int63n(int64(d) / 3))
	if rand.Intn(2) == 0 {
		return d - j
	}
	return d + j
}
