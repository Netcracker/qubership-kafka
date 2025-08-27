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
	"github.com/Netcracker/qubership-kafka/operator/controllers"
	"github.com/Netcracker/qubership-kafka/operator/controllers/kafka"
	"github.com/Netcracker/qubership-kafka/operator/controllers/kafkaservice"
	"github.com/go-logr/logr"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
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
			return nil, err
		}
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		return nil, err
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		return nil, err
	}

	// TEST ONLY ERR
	if d := os.Getenv("DEBUG_FAIL_AFTER"); d != "" {
		if dur, perr := time.ParseDuration(d); perr == nil {
			_ = mgr.Add(manager.RunnableFunc(func(inner context.Context) error {
				select {
				case <-time.After(dur):
					return fmt.Errorf("debug: forced runtime failure (kafka) after %s", dur)
				case <-inner.Done():
					return nil
				}
			}))
		}
	}

	// TEST ONLY PANIC
	if d := os.Getenv("DEBUG_PANIC_AFTER"); d != "" {
		if dur, perr := time.ParseDuration(d); perr == nil {
			_ = mgr.Add(manager.RunnableFunc(func(inner context.Context) error {
				select {
				case <-time.After(dur):
					panic("debug: forced panic in kafka manager runnable")
				case <-inner.Done():
					return nil
				}
			}))
		}
	}

	exec := func() error {
		logger.Info(fmt.Sprintf("starting %s manager", string(opts.Mode)))
		if err = mgr.Start(ctx); err != nil {
			logger.Error(err, "problem running manager")
			return err
		}
		if ctx.Err() == nil {
			return fmt.Errorf("manager stopped unexpectedly without context cancel")
		}
		return nil
	}

	return exec, nil

}
