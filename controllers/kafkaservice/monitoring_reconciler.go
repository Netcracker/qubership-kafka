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
	"fmt"
	"time"

	kafkaservice "github.com/Netcracker/qubership-kafka/api/v7"
	"github.com/Netcracker/qubership-kafka/controllers"
	"github.com/Netcracker/qubership-kafka/controllers/provider"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	monitoringConditionReason      = "KafkaMonitoringReadinessStatus"
	monitoringHashName             = "spec.monitoring"
	lagExporterConfigMapNameSuffix = "lag-exporter-configmap"
	lagExporterConfigMapKey        = "kafka-lag-exporter-cm"
)

type ReconcileMonitoring struct {
	cr                       *kafkaservice.KafkaService
	reconciler               *KafkaServiceReconciler
	logger                   logr.Logger
	monitoringProvider       provider.MonitoringResourceProvider
	lagExporterConfigMapName string
}

func NewReconcileMonitoring(r *KafkaServiceReconciler, cr *kafkaservice.KafkaService, logger logr.Logger) ReconcileMonitoring {
	return ReconcileMonitoring{
		cr:                       cr,
		logger:                   logger,
		reconciler:               r,
		monitoringProvider:       provider.NewMonitoringResourceProvider(cr, logger),
		lagExporterConfigMapName: fmt.Sprintf("%s-%s", cr.Name, lagExporterConfigMapNameSuffix),
	}
}

func (r ReconcileMonitoring) Reconcile() error {
	monitoringSecret, err := r.reconciler.WatchSecret(r.cr.Spec.Monitoring.SecretName, r.cr, r.logger)
	if err != nil {
		return err
	}

	currentCMResourceVersion := ""
	if r.cr.Spec.Monitoring.LagExporter != nil {
		configMap, err := r.reconciler.FindConfigMap(r.lagExporterConfigMapName, r.cr.Namespace, r.logger)
		if err != nil {
			log.Error(err, fmt.Sprintf("Config map - %s is not found and Kafka Lag Exporter can't be configured",
				r.lagExporterConfigMapName))
		}
		if err := controllerutil.SetControllerReference(r.cr, configMap, r.reconciler.Scheme); err != nil {
			return err
		}
		if err := r.reconciler.CreateOrUpdateConfigMap(configMap, r.logger); err != nil {
			return err
		}
		currentCMResourceVersion = configMap.ResourceVersion
	}

	monitoringHash, err := util.Hash(r.cr.Spec.Monitoring)
	if err != nil {
		return err
	}
	lagExporterUpdateEnabled := r.isDeploymentUpdateNeededForKafkaLagExporter(currentCMResourceVersion)
	if r.reconciler.ResourceHashes[monitoringHashName] != monitoringHash ||
		r.reconciler.ResourceHashes[globalHashName] != globalSpecHash ||
		(monitoringSecret.Name != "" && r.reconciler.ResourceVersions[monitoringSecret.Name] != monitoringSecret.ResourceVersion) ||
		lagExporterUpdateEnabled {
		monitoringLabels := r.monitoringProvider.GetMonitoringSelectorLabels()

		clientService := r.monitoringProvider.NewMonitoringClientService()
		if err := controllerutil.SetControllerReference(r.cr, clientService, r.reconciler.Scheme); err != nil {
			return err
		}
		if err := r.reconciler.CreateOrUpdateService(clientService, r.logger); err != nil {
			return err
		}

		serviceAccount := provider.NewServiceAccount(r.monitoringProvider.GetServiceAccountName(), r.cr.Namespace, r.cr.Spec.Global.DefaultLabels)
		if err := r.reconciler.CreateOrUpdateServiceAccount(serviceAccount, r.logger); err != nil {
			return err
		}

		deployment := r.monitoringProvider.NewMonitoringDeployment(currentCMResourceVersion)
		if err := controllerutil.SetControllerReference(r.cr, deployment, r.reconciler.Scheme); err != nil {
			return err
		}
		if err := r.reconciler.CreateOrUpdateDeployment(deployment, r.logger); err != nil {
			return err
		}
		r.logger.Info("Monitoring deployment has been created or updated")

		r.logger.Info("Updating Monitoring status")
		if err := r.updateMonitoringStatus(r.cr, monitoringLabels); err != nil {
			return err
		}
	} else {
		r.logger.Info("Kafka monitoring configuration didn't change, skipping reconcile loop")
	}

	r.reconciler.ResourceVersions[monitoringSecret.Name] = monitoringSecret.ResourceVersion
	r.reconciler.ResourceHashes[monitoringHashName] = monitoringHash
	if r.cr.Spec.Monitoring.LagExporter != nil {
		r.reconciler.ResourceVersions[lagExporterConfigMapKey] = currentCMResourceVersion
	}
	return nil
}

func (r ReconcileMonitoring) Status() error {
	if err := r.reconciler.updateConditions(NewCondition(statusFalse,
		typeInProgress,
		monitoringConditionReason,
		"Kafka Monitoring health check")); err != nil {
		return err
	}
	r.logger.Info("Start checking the readiness of Kafka Monitoring pod")
	err := wait.PollImmediate(waitingInterval, time.Duration(r.cr.Spec.Global.PodsReadyTimeout)*time.Second, func() (done bool, err error) {
		labels := r.monitoringProvider.GetMonitoringSelectorLabels()
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		return r.reconciler.updateConditions(NewCondition(statusFalse, typeFailed, monitoringConditionReason, "Kafka Monitoring pod is not ready"))
	}
	return r.reconciler.updateConditions(NewCondition(statusTrue, typeReady, monitoringConditionReason, "Kafka Monitoring pod is ready"))
}

func (r *ReconcileMonitoring) updateMonitoringStatus(cr *kafkaservice.KafkaService, labels map[string]string) error {
	foundPodList, err := r.reconciler.FindPodList(r.cr.Namespace, labels)
	if err != nil {
		return err
	}
	return r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafkaservice.KafkaService) {
		instance.Status.MonitoringStatus.Nodes = controllers.GetPodNames(foundPodList.Items)
	})
}

func (r ReconcileMonitoring) isDeploymentUpdateNeededForKafkaLagExporter(currentCMResourceVersion string) bool {
	if r.cr.Spec.Monitoring.LagExporter == nil {
		return false
	}
	if r.reconciler.ResourceVersions[lagExporterConfigMapKey] == "" || currentCMResourceVersion == "" {
		return false
	}
	return currentCMResourceVersion != r.reconciler.ResourceVersions[lagExporterConfigMapKey]
}
