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

func (rj KmmJob) Enabled(opts cfg.Cfg) (runJob bool, runDuplicate bool) {
	runJob = opts.Mode == cfg.KafkaServiceMode && opts.KmmEnabled
	runDuplicate = true
	return
}
