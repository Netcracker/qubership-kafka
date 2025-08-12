package workers

import (
	"context"
	"github.com/Netcracker/qubership-kafka/operator/cfg"
	"github.com/Netcracker/qubership-kafka/operator/workers/jobs"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
)

type Pool struct {
	opts       cfg.Cfg
	log        logr.Logger
	workerPool *errgroup.Group
	jbs        []jobs.Job
	ctx        context.Context
}

func NewPool(ctx context.Context, opts cfg.Cfg, logger logr.Logger) *Pool {
	wrk := &Pool{opts: opts,
		log: logger}
	wrk.workerPool, wrk.ctx = errgroup.WithContext(ctx)
	wrk.jbs = []jobs.Job{
		jobs.KafkaJob{},
		jobs.AkhqJob{},
		jobs.KmmJob{},
		jobs.KafkaUserJob{},
	}

	return wrk
}

func (wrk *Pool) Wait() error {
	err := wrk.workerPool.Wait()
	return err
}

func (wrk *Pool) Start() error {
	var err error
	wrk.log.Info("Starting workers")
	err = wrk.runJobs(wrk.opts.ApiGroup, wrk.opts.SecondaryApiGroup)
	if err != nil {
		return err
	}
	return nil
}

func (wrk *Pool) runJobs(apiGroup, secondApiGroup string) error {
	var err error

	for _, j := range wrk.jbs {
		err = wrk.run(j, apiGroup)
		if err != nil {
			return err
		}
		if len(secondApiGroup) > 0 {
			err = wrk.run(j, secondApiGroup)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (wrk *Pool) run(job jobs.Job, apiGroup string) error {
	var err error
	var exe jobs.Exec
	exe, err = job.Build(wrk.ctx, wrk.opts, apiGroup, wrk.log)
	if err != nil {
		return err
	}
	if exe == nil {
		return nil
	}
	wrk.workerPool.Go(exe)
	return nil
}
