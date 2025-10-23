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

package jobs

import (
	"context"
	"fmt"
	"github.com/Netcracker/qubership-kafka/operator/cfg"
	"github.com/Netcracker/qubership-kafka/operator/controllers/kafkauser"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type KafkaUserJob struct {
}

func (rj KafkaUserJob) Build(ctx context.Context, opts cfg.Cfg, apiGroup string, logger logr.Logger) (Exec, error) {
	var err error

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
		return nil, err
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

func (rj KafkaUserJob) Enabled(opts cfg.Cfg) (runJob bool, runDuplicate bool) {
	runJob = opts.Mode == cfg.KafkaServiceMode && opts.WatchKafkaUsersCollectNamespace != nil
	runDuplicate = true
	return
}
