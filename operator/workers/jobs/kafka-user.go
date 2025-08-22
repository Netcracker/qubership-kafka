package jobs

import (
	"context"
	"fmt"
	"github.com/Netcracker/qubership-kafka/operator/cfg"
	"github.com/Netcracker/qubership-kafka/operator/controllers/kafkauser"
	"github.com/go-logr/logr"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type KafkaUserJob struct {
}

func (rj KafkaUserJob) Build(ctx context.Context, opts cfg.Cfg, apiGroup string, logger logr.Logger) (Exec, error) {
	var err error
	if opts.Mode == cfg.KafkaMode || opts.WatchKafkaUsersCollectNamespace == nil {
		return nil, nil
	}

	namespace := *opts.WatchKafkaUsersCollectNamespace

	runScheme := scheme
	port := 9544
	if mainApiGroup() != apiGroup {
		runScheme, err = duplicateScheme(apiGroup)
		if err != nil {
			logger.Error(err, "duplicate scheme error", "group", apiGroup)
			return nil, err
		}
		port += 10
	}

	kafkaUsersMgrOptions := ctrl.Options{
		Scheme:                  runScheme,
		MetricsBindAddress:      "0",
		Port:                    port,
		HealthProbeBindAddress:  "0",
		LeaderElection:          opts.EnableLeaderElection,
		LeaderElectionNamespace: opts.OperatorNamespace,
		LeaderElectionID:        fmt.Sprintf("kafkausers.%s.%s", opts.OperatorNamespace, apiGroup),
	}
	configureManagerNamespaces(&kafkaUsersMgrOptions, namespace, opts.OperatorNamespace)

	kafkaUserMgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), kafkaUsersMgrOptions)
	if err != nil {
		logger.Error(err, "unable to start Kafka Users manager")
		return nil, err
	}

	reconciliationPeriod := opts.KafkaUserConfiguratorReconcilePeriodSecs

	secretCreatingEnabled := opts.KafkaUserSecretCreatingEnabled

	kafkaSslEnabled := opts.KafkaSslEnabled

	if err = (&kafkauser.KafkaUserReconciler{
		BootstrapServers:      opts.KafkaBootstrapServers,
		Client:                kafkaUserMgr.GetClient(),
		SecretCreatingEnabled: secretCreatingEnabled,
		Namespace:             opts.OperatorNamespace,
		ReconciliationPeriod:  reconciliationPeriod,
		Scheme:                kafkaUserMgr.GetScheme(),
		ResourceHashes:        map[string]string{},
		KafkaSecret:           opts.KafkaSecret,
		KafkaSaslMechanism:    opts.KafkaSaslMechanism,
		KafkaSslEnabled:       kafkaSslEnabled,
		KafkaSslSecret:        opts.KafkaSslSecret,
		ApiGroup:              apiGroup,
	}).SetupWithManager(kafkaUserMgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "KafkaUsers")
		os.Exit(1)
	}

	if err = kafkaUserMgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		return nil, err
	}
	if err = kafkaUserMgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		return nil, err
	}

	exec := func() error {
		defer func() {
			logger.Info("KafkaUser manager goroutine has been finished")
		}()
		logger.Info("starting KafkaUser manager")
		if err = kafkaUserMgr.Start(ctx); err != nil {
			logger.Error(err, "problem running KafkaUser manager")
			return err
		}
		return nil
	}
	return exec, nil
}
