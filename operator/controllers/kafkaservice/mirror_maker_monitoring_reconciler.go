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

package kafkaservice

import (
	kafkaservice "github.com/Netcracker/qubership-kafka/operator/api/v7"

	"time"

	"github.com/Netcracker/qubership-kafka/operator/controllers/provider"
	"github.com/Netcracker/qubership-kafka/operator/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	mirrorMakerMonitoringConditionReason = "KafkaMirrorMakerMonitoringReadinessStatus"
	mirrorMakerMonitoringHashName        = "spec.mirrorMakerMonitoring"
)

type ReconcileMirrorMakerMonitoring struct {
	cr                            *kafkaservice.KafkaService
	reconciler                    *KafkaServiceReconciler
	logger                        logr.Logger
	drChecked                     bool
	mirrorMakerMonitoringProvider provider.MirrorMakerMonitoringResourceProvider
}

func NewReconcileMirrorMakerMonitoring(r *KafkaServiceReconciler, cr *kafkaservice.KafkaService, logger logr.Logger, drChecked bool) ReconcileMirrorMakerMonitoring {
	return ReconcileMirrorMakerMonitoring{
		cr:                            cr,
		logger:                        logger,
		reconciler:                    r,
		drChecked:                     drChecked,
		mirrorMakerMonitoringProvider: provider.NewMirrorMakerMonitoringResourceProvider(cr, logger),
	}
}

func (r ReconcileMirrorMakerMonitoring) Reconcile() error {
	mirrorMakerMonitoringSecret, err := r.reconciler.WatchSecret(r.cr.Spec.MirrorMakerMonitoring.SecretName, r.cr, r.logger)
	if err != nil {
		return err
	}

	mirrorMakerMonitoringHash, err := util.Hash(r.cr.Spec.MirrorMakerMonitoring)
	if err != nil {
		return err
	}
	disasterRecoveryHash := ""
	if r.cr.Spec.DisasterRecovery != nil {
		disasterRecoveryHash, err = util.Hash(r.cr.Spec.DisasterRecovery)
		if err != nil {
			return nil
		}
	}
	if r.reconciler.ResourceHashes[mirrorMakerMonitoringHashName] != mirrorMakerMonitoringHash ||
		r.drChecked ||
		(mirrorMakerMonitoringSecret.Name != "" && r.reconciler.ResourceVersions[mirrorMakerMonitoringSecret.Name] != mirrorMakerMonitoringSecret.ResourceVersion) {
		mirrorMakerMonitoringProvider := r.mirrorMakerMonitoringProvider

		clientService := mirrorMakerMonitoringProvider.NewMonitoringClientService()
		if err := r.reconciler.SetControllerReference(r.cr, clientService, r.reconciler.Scheme); err != nil {
			return err
		}
		if err := r.reconciler.CreateOrUpdateService(clientService, r.logger); err != nil {
			return err
		}

		serviceAccount := provider.NewServiceAccount(r.mirrorMakerMonitoringProvider.GetServiceAccountName(), r.cr.Namespace, r.cr.Spec.Global.DefaultLabels)
		if err := r.reconciler.CreateOrUpdateServiceAccount(serviceAccount, r.logger); err != nil {
			return err
		}

		deployment := mirrorMakerMonitoringProvider.NewMirrorMakerMonitoringDeployment()
		if err := r.reconciler.SetControllerReference(r.cr, deployment, r.reconciler.Scheme); err != nil {
			return err
		}
		if err := r.reconciler.CreateOrUpdateDeployment(deployment, r.logger); err != nil {
			return err
		}
	} else {
		r.logger.Info("Kafka Mirror Maker monitoring configuration didn't change, skipping reconcile loop")
	}
	r.reconciler.ResourceVersions[mirrorMakerMonitoringSecret.Name] = mirrorMakerMonitoringSecret.ResourceVersion
	r.reconciler.ResourceHashes[mirrorMakerMonitoringHashName] = mirrorMakerMonitoringHash
	r.reconciler.ResourceHashes[disasterRecoveryHashName] = disasterRecoveryHash
	return nil
}

func (r ReconcileMirrorMakerMonitoring) Status() error {
	if err := r.reconciler.updateConditions(NewCondition(statusFalse,
		typeInProgress,
		mirrorMakerMonitoringConditionReason,
		"Kafka Mirror Maker Monitoring health check")); err != nil {
		return err
	}
	r.logger.Info("Start checking the readiness of Kafka Mirror Maker Monitoring pod")
	err := wait.PollImmediate(waitingInterval, time.Duration(r.cr.Spec.Global.PodsReadyTimeout)*time.Second, func() (done bool, err error) {
		labels := r.mirrorMakerMonitoringProvider.GetMirrorMakerMonitoringSelectorLabels()
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		return r.reconciler.updateConditions(NewCondition(statusFalse, typeFailed, mirrorMakerMonitoringConditionReason, "Kafka Mirror Maker Monitoring pod is not ready"))
	}
	return r.reconciler.updateConditions(NewCondition(statusTrue, typeReady, mirrorMakerMonitoringConditionReason, "Kafka Mirror Maker Monitoring pod is ready"))
}
