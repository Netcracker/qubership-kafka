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
	"strings"
	"time"

	kafkaservice "github.com/Netcracker/qubership-kafka/operator/api/v7"
	"github.com/Netcracker/qubership-kafka/operator/controllers"
	"github.com/Netcracker/qubership-kafka/operator/controllers/provider"
	"github.com/Netcracker/qubership-kafka/operator/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	mirrorMakerConditionReason = "KafkaMirrorMakerReadinessStatus"
	mirrorMakerHashName        = "spec.mirrorMaker"
	disasterRecoveryHashName   = "spec.disasterRecovery"
)

type ReconcileMirrorMaker struct {
	cr                  *kafkaservice.KafkaService
	reconciler          *KafkaServiceReconciler
	logger              logr.Logger
	drChecked           bool
	mirrorMakerProvider provider.MirrorMakerResourceProvider
}

func NewReconcileMirrorMaker(r *KafkaServiceReconciler, cr *kafkaservice.KafkaService, logger logr.Logger, drChecked bool) ReconcileMirrorMaker {
	return ReconcileMirrorMaker{
		cr:                  cr,
		logger:              logger,
		reconciler:          r,
		drChecked:           drChecked,
		mirrorMakerProvider: provider.NewMirrorMakerResourceProvider(cr, logger),
	}
}

func (r ReconcileMirrorMaker) Reconcile() error {
	mirrorMakerProvider := r.mirrorMakerProvider
	mirrorMakerSpec := r.cr.Spec.MirrorMaker
	if len(mirrorMakerSpec.Clusters) > 1 {

		secretKey := fmt.Sprintf("%s.%s", mirrorMakerSpec.SecretName, r.cr.Namespace)
		secret, err := r.reconciler.WatchSecret(mirrorMakerSpec.SecretName, r.cr, r.logger)
		if err != nil {
			return err
		}
		secretVersion := secret.ResourceVersion

		configurationKey := fmt.Sprintf("%s.%s", mirrorMakerSpec.ConfigurationName, r.cr.Namespace)
		configMap, err := r.reconciler.FindConfigMap(mirrorMakerSpec.ConfigurationName, r.cr.Namespace, r.logger)
		if err != nil {
			if errors.IsNotFound(err) {
				configMap = mirrorMakerProvider.NewConfigurationMapForCR()
				if err := r.reconciler.SetControllerReference(r.cr, configMap, r.reconciler.Scheme); err != nil {
					return err
				}
				if err := r.reconciler.CreateOrUpdateConfigMap(configMap, r.logger); err != nil {
					return err
				}
			} else {
				return err
			}
		} else if !mirrorMakerSpec.ConfiguratorEnabled {
			configMap.Data["config"] = mirrorMakerProvider.GetMirrorMakerProperties()
			if err := r.reconciler.CreateOrUpdateConfigMap(configMap, r.logger); err != nil {
				return err
			}
		}
		configurationVersion := configMap.ResourceVersion

		mirrorMakerHash, err := util.Hash(r.cr.Spec.MirrorMaker)
		if err != nil {
			return err
		}
		disasterRecoveryHash := ""
		if r.cr.Spec.DisasterRecovery != nil {
			disasterRecoveryHash, err = util.Hash(r.cr.Spec.DisasterRecovery)
			if err != nil {
				return err
			}
		}
		if r.reconciler.ResourceHashes[mirrorMakerHashName] != mirrorMakerHash ||
			r.drChecked ||
			secret.Name != "" && r.reconciler.ResourceVersions[secretKey] != secretVersion ||
			r.reconciler.ResourceVersions[configurationKey] != configurationVersion {
			serviceAccount := provider.NewServiceAccount(r.mirrorMakerProvider.GetServiceAccountName(), r.cr.Namespace, r.cr.Spec.Global.DefaultLabels)
			if err := r.reconciler.CreateOrUpdateServiceAccount(serviceAccount, r.logger); err != nil {
				return err
			}

			if mirrorMakerSpec.RegionName == "" {
				for _, cluster := range mirrorMakerSpec.Clusters {
					if err := r.createDeployment(cluster, secretVersion, configurationVersion); err != nil {
						return err
					}
				}
			} else {
				for _, cluster := range mirrorMakerSpec.Clusters {
					if cluster.Name == mirrorMakerSpec.RegionName {
						if err := r.createDeployment(cluster, secretVersion, configurationVersion); err != nil {
							return err
						}
						break
					}
				}
			}

			r.logger.Info("Updating Kafka Mirror Maker status")
			if err := r.updateMirrorMakerStatus(r.cr); err != nil {
				return err
			}
		} else {
			r.logger.Info("Kafka mirror maker configuration didn't change, skipping reconcile loop")
		}
		r.reconciler.ResourceVersions[secretKey] = secretVersion
		r.reconciler.ResourceVersions[configurationKey] = configurationVersion
		r.reconciler.ResourceHashes[mirrorMakerHashName] = mirrorMakerHash
		r.reconciler.ResourceHashes[disasterRecoveryHashName] = disasterRecoveryHash
	}
	return nil
}

func (r ReconcileMirrorMaker) Status() error {
	if err := r.reconciler.updateConditions(NewCondition(statusFalse,
		typeInProgress,
		mirrorMakerConditionReason,
		"Kafka Mirror Maker health check")); err != nil {
		return err
	}
	r.logger.Info("Start checking the readiness of Kafka Mirror Maker pods")
	err := wait.PollImmediate(waitingInterval, time.Duration(r.cr.Spec.Global.PodsReadyTimeout)*time.Second, func() (done bool, err error) {
		labels := r.mirrorMakerProvider.GetMirrorMakerSelectorLabels()
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		return r.reconciler.updateConditions(NewCondition(statusFalse, typeFailed, mirrorMakerConditionReason, "Kafka Mirror Maker pods are not ready"))
	}
	return r.reconciler.updateConditions(NewCondition(statusTrue, typeReady, mirrorMakerConditionReason, "Kafka Mirror Maker pods are ready"))
}

func (r ReconcileMirrorMaker) createDeployment(cluster kafkaservice.Cluster, secretVersion string,
	configurationVersion string) error {
	mirrorMakerProvider := r.mirrorMakerProvider
	mirrorMakerSpec := r.cr.Spec.MirrorMaker

	currentClusterName := strings.ToLower(cluster.Name)
	deploymentName := fmt.Sprintf("%s-%s", currentClusterName, mirrorMakerProvider.GetServiceName())
	r.logger.Info("Create deployment for cluster " + currentClusterName)

	mirrorMakerDeployment := mirrorMakerProvider.NewMirrorMakerDeploymentForCR(cluster,
		mirrorMakerSpec.Clusters, secretVersion, configurationVersion, currentClusterName, deploymentName)
	if err := r.reconciler.SetControllerReference(r.cr, mirrorMakerDeployment, r.reconciler.Scheme); err != nil {
		return err
	}
	mirrorMakerService := mirrorMakerProvider.GetService(deploymentName)
	if err := r.reconciler.SetControllerReference(r.cr, mirrorMakerService, r.reconciler.Scheme); err != nil {
		return err
	}
	if err := r.reconciler.CreateOrUpdateService(mirrorMakerService, r.logger); err != nil {
		return err
	}
	if err := r.reconciler.CreateOrUpdateDeployment(mirrorMakerDeployment, r.logger); err != nil {
		return err
	}
	return nil
}

// updateMirrorMakerStatus updates the status of Kafka Mirror Maker
func (r *ReconcileMirrorMaker) updateMirrorMakerStatus(cr *kafkaservice.KafkaService) error {
	labels := r.mirrorMakerProvider.GetMirrorMakerSelectorLabels()
	foundPodList, err := r.reconciler.FindPodList(r.cr.Namespace, labels)
	if err != nil {
		return err
	}
	return r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafkaservice.KafkaService) {
		instance.Status.MirrorMakerStatus.Nodes = controllers.GetPodNames(foundPodList.Items)
	})
}
