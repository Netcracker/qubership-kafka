package jobs

import (
	"context"
	"fmt"
	"github.com/Netcracker/qubership-kafka/operator/cfg"
	"github.com/Netcracker/qubership-kafka/operator/controllers/kmmconfig"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

const kmmJobName = "kmm"

type KmmJob struct {
}

func (rj KmmJob) Build(ctx context.Context, opts cfg.Cfg, apiGroup string, logger logr.Logger) (Exec, error) {
	var err error

	if opts.Mode == cfg.KafkaMode || !opts.KmmEnabled {
		return nil, nil
	}

	runScheme := scheme
	port := 9543
	if mainApiGroup() != apiGroup {
		runScheme, err = duplicateScheme(apiGroup)
		if err != nil {
			logger.Error(err, "duplicate scheme error", "group", opts.ApiGroup)
			return nil, err
		}
		port += 10
	}

	namespace, err := getWatchNamespace()
	if err != nil {
		logger.Error(err, "Failed to get watch namespace")
		return nil, err
	}

	kmmMgrOpts := ctrl.Options{
		Scheme:                  runScheme,
		MetricsBindAddress:      "0",
		Port:                    port,
		HealthProbeBindAddress:  "0",
		LeaderElection:          opts.EnableLeaderElection,
		LeaderElectionNamespace: opts.OperatorNamespace,
		LeaderElectionID:        fmt.Sprintf("kmmconfig.%s.%s", opts.OperatorNamespace, apiGroup),
	}
	configureManagerNamespaces(&kmmMgrOpts, namespace, opts.OperatorNamespace)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), kmmMgrOpts)
	if err != nil {
		logger.Error(err, fmt.Sprintf("unable to start %s manager", kmmJobName))
		return nil, err
	}

	err = (&kmmconfig.KmmConfigReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		ApiGroup: apiGroup,
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
			logger.Info("KmmConfig manager goroutine has been finished")
		}()

		logger.Info("starting KmmConfig manager")
		if err = mgr.Start(ctx); err != nil {
			logger.Error(err, "problem running KmmConfig manager")
			return err
		}
		return nil
	}

	return exec, nil
}
