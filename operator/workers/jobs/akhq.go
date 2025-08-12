package jobs

import (
	"context"
	"fmt"
	"github.com/Netcracker/qubership-kafka/operator/cfg"
	"github.com/Netcracker/qubership-kafka/operator/controllers/akhqconfig"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

const AkhqJobName = "akhq"

type AkhqJob struct {
}

func (rj AkhqJob) Build(ctx context.Context, opts cfg.Cfg, apiGroup string, logger logr.Logger) (Exec, error) {
	var err error
	if opts.Mode == cfg.KafkaMode || opts.WatchAkhqCollectNamespace == nil {
		return nil, nil
	}
	watchNamespace := *opts.WatchAkhqCollectNamespace
	runScheme := scheme
	port := 9542
	if mainApiGroup() != apiGroup {
		runScheme, err = duplicateScheme(apiGroup)
		if err != nil {
			logger.Error(err, "duplicate scheme error")
			return nil, err
		}
		port += 10
	}

	akhqOpts := ctrl.Options{
		Scheme:                  runScheme,
		MetricsBindAddress:      "0",
		Port:                    port,
		HealthProbeBindAddress:  "0",
		LeaderElection:          opts.EnableLeaderElection,
		LeaderElectionNamespace: opts.OperatorNamespace,
		LeaderElectionID:        fmt.Sprintf("akhqconfig.%s.%s", opts.OperatorNamespace, opts.ApiGroup),
	}

	configureManagerNamespaces(&akhqOpts, watchNamespace, opts.OperatorNamespace)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), akhqOpts)
	if err != nil {
		logger.Error(err, fmt.Sprintf("unable to start %s manager", AkhqJobName))
		return nil, err
	}

	err = (&akhqconfig.AkhqConfigReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Namespace: opts.OperatorNamespace,
		ApiGroup:  apiGroup,
	}).SetupWithManager(mgr)

	if err != nil {
		return nil, err
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		return nil, err
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		return nil, err
	}

	exec := func() error {
		defer func() {
			logger.Info("akhq config manager goroutine has been finished")
		}()

		logger.Info("starting akhq config manager")
		if err = mgr.Start(ctx); err != nil {
			logger.Error(err, "akhq config manager stopped due to error")
			return err
		}
		return nil
	}

	return exec, nil
}
