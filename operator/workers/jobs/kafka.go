package jobs

import (
	"context"
	"fmt"
	"github.com/Netcracker/qubership-kafka/operator/cfg"
	"github.com/Netcracker/qubership-kafka/operator/controllers"
	"github.com/Netcracker/qubership-kafka/operator/controllers/kafka"
	"github.com/Netcracker/qubership-kafka/operator/controllers/kafkaservice"
	"github.com/go-logr/logr"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

const kafkaJobName = "kafka-service"

type KafkaJob struct {
}

func (rj KafkaJob) Build(ctx context.Context, opts cfg.Cfg, apiGroup string, logger logr.Logger) (Exec, error) {
	var err error
	if len(opts.Mode) == 0 {
		return nil, nil
	}

	runScheme := scheme
	port := 9443
	metricsAddr := opts.MetricsAddr
	probeAddr := opts.ProbeAddr
	if mainApiGroup() != apiGroup {
		return nil, nil
	}

	kafkaOpts := ctrl.Options{
		Scheme:                  runScheme,
		Namespace:               opts.OperatorNamespace,
		MetricsBindAddress:      metricsAddr,
		Port:                    port,
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          opts.EnableLeaderElection,
		LeaderElectionNamespace: opts.OperatorNamespace,
		LeaderElectionID:        fmt.Sprintf("%s.%s.%s", string(opts.Mode), opts.OperatorNamespace, apiGroup),
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), kafkaOpts)
	if err != nil {
		logger.Error(err, "unable to start manager", "job", kafkaJobName)
		return nil, err
	}

	if opts.Mode == cfg.KafkaMode {
		if err = (&kafka.KafkaReconciler{
			Reconciler: controllers.Reconciler{
				Client:           mgr.GetClient(),
				Scheme:           mgr.GetScheme(),
				ResourceVersions: map[string]string{},
				ResourceHashes:   map[string]string{},
				ApiGroup:         apiGroup,
			},
		}).SetupWithManager(mgr); err != nil {
			logger.Error(err, "unable to create controller", "controller", "Kafka")
			return nil, err
		}
	} else {
		if err = (&kafkaservice.KafkaServiceReconciler{
			Reconciler: controllers.Reconciler{
				Client:           mgr.GetClient(),
				Scheme:           mgr.GetScheme(),
				ResourceVersions: map[string]string{},
				ResourceHashes:   map[string]string{},
				ApiGroup:         apiGroup,
			},
		}).SetupWithManager(mgr); err != nil {
			logger.Error(err, "unable to create controller", "controller", "KafkaService")
			os.Exit(1)
		}
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	exec := func() error {

		logger.Info(fmt.Sprintf("starting %s manager", string(opts.Mode)))
		if err = mgr.Start(ctx); err != nil {
			logger.Error(err, "problem running manager")
			return err
		}
		return nil
	}

	return exec, nil

}
