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
	appsv1 "k8s.io/api/apps/v1"
	"regexp"
	"time"

	kafkaservice "github.com/Netcracker/qubership-kafka/api/v7"
	"github.com/Netcracker/qubership-kafka/controllers"
	akhqproto "github.com/Netcracker/qubership-kafka/controllers/akhqprotobuf"
	"github.com/Netcracker/qubership-kafka/controllers/provider"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	akhqConditionReason                = "AkhqReadinessStatus"
	akhqHashName                       = "spec.akhq"
	protobufConfigurationCMName        = "akhq-protobuf-configuration"
	ldapConfigurationCMName            = "akhq-security-configuration"
	legacyDeserializationConfigMapName = "akhq-deserialization-keys"
	protobufConfig                     = `deserialization:
 protobuf:
   descriptors-folder: "/app/config"
   topics-mapping: []
`
)

var controllerReferenceSet = false
var ldapSecret *corev1.Secret
var ldapConfigMap *corev1.ConfigMap

type ReconcileAkhq struct {
	cr           *kafkaservice.KafkaService
	reconciler   *KafkaServiceReconciler
	akhqProvider provider.AkhqResourceProvider
	logger       logr.Logger
}

func NewReconcileAkhq(r *KafkaServiceReconciler, cr *kafkaservice.KafkaService, logger logr.Logger) ReconcileAkhq {
	return ReconcileAkhq{cr: cr, logger: logger, reconciler: r, akhqProvider: provider.NewAkhqResourceProvider(cr, logger)}
}

func (r ReconcileAkhq) Reconcile() error {
	akhqSecret, err := r.reconciler.WatchSecret("akhq-secret", r.cr, r.logger)
	if err != nil {
		return err
	}

	kafkaServicesSecret, err := r.reconciler.WatchSecret(fmt.Sprintf("%s-services-secret", r.cr.Name), r.cr, r.logger)
	if err != nil {
		return err
	}

	if err := r.deleteLegacyConfigMap(); err != nil {
		return err
	}

	protobufConfigMap, err := r.getProtobufConfigMap()
	if err != nil {
		return err
	}

	if r.cr.Spec.Akhq.Ldap.Enabled {
		ldapSecret, err = r.reconciler.WatchSecret("akhq-ldap-secret", r.cr, r.logger)
		if err != nil {
			return err
		}

		ldapConfigMap, err = r.getLdapConfigMap(ldapConfigurationCMName, r.logger)
		if err != nil {
			r.logger.Error(err, "Error in getting LDAP ConfigMap")
			return err
		}
	}

	deserealizationSourceConfigMaps, err := r.findDeserializationConfigMaps()
	if err != nil {
		r.logger.Error(err, "Cannot create or find deserialization source config maps")
		return err
	}

	akhqSpecHash, err := util.Hash(r.cr.Spec.Akhq)
	if err != nil {
		return err
	}
	if r.reconciler.ResourceHashes[akhqHashName] != akhqSpecHash ||
		r.reconciler.ResourceHashes[globalHashName] != globalSpecHash ||
		(akhqSecret.Name != "" && r.reconciler.ResourceVersions[akhqSecret.Name] != akhqSecret.ResourceVersion) ||
		(ldapSecret != nil && ldapSecret.Name != "" && r.reconciler.ResourceVersions[ldapSecret.Name] != ldapSecret.ResourceVersion) ||
		(kafkaServicesSecret.Name != "" && r.reconciler.ResourceVersions[kafkaServicesSecret.Name] != kafkaServicesSecret.ResourceVersion) ||
		(protobufConfigMap.Name != "" && r.reconciler.ResourceVersions[protobufConfigMap.Name] != protobufConfigMap.ResourceVersion) ||
		(ldapConfigMap != nil && ldapConfigMap.Name != "" && r.reconciler.ResourceVersions[ldapConfigMap.Name] != ldapConfigMap.ResourceVersion) {
		akhqLabels := r.akhqProvider.GetAkhqSelectorLabels()

		trustCertsSecret := r.akhqProvider.NewLdapTrustedSecret("akhq-trusted-certs")
		if err := r.reconciler.CreateOrUpdateSecret(trustCertsSecret, r.logger); err != nil {
			return err
		}

		clientService := r.akhqProvider.NewAkhqClientService()
		if err := controllerutil.SetControllerReference(r.cr, clientService, r.reconciler.Scheme); err != nil {
			return err
		}
		if err := r.reconciler.CreateOrUpdateService(clientService, r.logger); err != nil {
			return err
		}

		serviceAccount := provider.NewServiceAccount(r.akhqProvider.GetServiceAccountName(), r.cr.Namespace, r.cr.Spec.Global.DefaultLabels)
		if err := r.reconciler.CreateOrUpdateServiceAccount(serviceAccount, r.logger); err != nil {
			return err
		}

		r.logger.Info("Checking security configuration. " +
			"Empty security configuration will be created if it does not exist. " +
			"Otherwise, it will not be deleted during installation, to check and change it go to the <akhq-security-configuration> secret")
		securityConfiguration := r.akhqProvider.NewSecurityConfiguration()

		errSec := r.reconciler.RestoreSpecFields(securityConfiguration, r.akhqProvider.ProtectedSecretsFields(), r.logger)
		if errSec != nil {
			return err
		}

		if errSec = r.reconciler.CreateOrUpdateSecret(securityConfiguration, r.logger); errSec != nil {
			return errSec
		}

		deployment := r.akhqProvider.NewAkhqDeployment(protobufConfigMap.ResourceVersion, deserealizationSourceConfigMaps)
		if err := controllerutil.SetControllerReference(r.cr, deployment, r.reconciler.Scheme); err != nil {
			return err
		}
		if kafkaServicesSecret.Annotations != nil && kafkaServicesSecret.Annotations[autoRestartAnnotation] == "true" {
			r.addDeploymentAnnotation(deployment, fmt.Sprintf(resourceVersionAnnotationTemplate, kafkaServicesSecret.Name), kafkaServicesSecret.ResourceVersion)
		}
		if err := r.reconciler.CreateOrUpdateDeployment(deployment, r.logger); err != nil {
			return err
		}

		r.logger.Info("Updating AKHQ status")
		if err := r.updateAkhqStatus(akhqLabels); err != nil {
			return err
		}
	} else {
		r.logger.Info("AKHQ configuration didn't change, skipping reconcile loop")
	}
	r.reconciler.ResourceVersions[akhqSecret.Name] = akhqSecret.ResourceVersion
	r.reconciler.ResourceVersions[kafkaServicesSecret.Name] = kafkaServicesSecret.ResourceVersion
	r.reconciler.ResourceVersions[protobufConfigMap.Name] = protobufConfigMap.ResourceVersion
	r.reconciler.ResourceHashes[akhqHashName] = akhqSpecHash
	if r.cr.Spec.Akhq.Ldap.Enabled {
		r.reconciler.ResourceVersions[ldapSecret.Name] = ldapSecret.ResourceVersion
		r.reconciler.ResourceVersions[ldapConfigMap.Name] = ldapConfigMap.ResourceVersion
	}
	return nil
}

func (r *ReconcileAkhq) getProtobufConfigMap() (*corev1.ConfigMap, error) {
	var protobufConfigMap *corev1.ConfigMap
	var err error
	if !controllerReferenceSet {
		protobufConfigMap, err = akhqproto.CreateProtobufConfigMapIfNotExist(r.cr.Namespace, r.reconciler.Client, r.akhqProvider.GetAkhqLabels(), r.cr, r.reconciler.Scheme)
		if err == nil {
			controllerReferenceSet = true
		}
	} else {
		protobufConfigMap, err = akhqproto.CreateProtobufConfigMapIfNotExist(r.cr.Namespace, r.reconciler.Client, r.akhqProvider.GetAkhqLabels(), nil, nil)
	}
	return protobufConfigMap, err
}

func (r ReconcileAkhq) getLdapConfigMap(configMapName string, logger logr.Logger) (*corev1.ConfigMap, error) {
	ldapConfigMap := r.akhqProvider.NewConfigurationMapForLdap(configMapName)
	err := r.reconciler.CreateOrUpdateConfigMap(ldapConfigMap, logger)
	if err != nil {
		logger.Error(err, "Failed to create or update LDAP ConfigMap", "ConfigMap.Name", configMapName)
		return nil, err
	}
	return ldapConfigMap, nil
}

func (r ReconcileAkhq) Status() error {
	if err := r.reconciler.updateConditions(NewCondition(statusFalse,
		typeInProgress,
		akhqConditionReason,
		"AKHQ health check")); err != nil {
		return err
	}
	r.logger.Info("Start checking the readiness of AKHQ pod")
	err := wait.PollImmediate(waitingInterval, time.Duration(r.cr.Spec.Global.PodsReadyTimeout)*time.Second, func() (done bool, err error) {
		labels := r.akhqProvider.GetAkhqSelectorLabels()
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		return r.reconciler.updateConditions(NewCondition(statusFalse, typeFailed, akhqConditionReason, "AKHQ pod is not ready"))
	}
	return r.reconciler.updateConditions(NewCondition(statusTrue, typeReady, akhqConditionReason, "AKHQ pod is ready"))
}

func (r *ReconcileAkhq) deleteLegacyConfigMap() error {
	r.logger.Info("Deleting legacy deserialization config map...")
	return r.reconciler.DeleteConfigMapByName(legacyDeserializationConfigMapName, r.cr.Namespace, r.logger)
}

func (r *ReconcileAkhq) findDeserializationConfigMaps() ([]*corev1.ConfigMap, error) {
	cms, err := r.reconciler.GetAllConfigMapsFromNamespace(r.cr.Namespace, r.logger)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(`^akhqdcm-(.*)$`)
	if err != nil {
		r.logger.Error(err, "Error while compiling regexp for akhq-deserialization keys searching")
	}

	// the assumption is that most often there will be no more than half of all configuration maps
	deserializationCMs := make([]*corev1.ConfigMap, 0, len(cms.Items)/2)
	for _, item := range cms.Items {
		if re.Match([]byte(item.Name)) {
			deserializationCMs = append(deserializationCMs, &item)
		}
	}

	return deserializationCMs, nil
}

func (r *ReconcileAkhq) updateAkhqStatus(labels map[string]string) error {
	foundPodList, err := r.reconciler.FindPodList(r.cr.Namespace, labels)
	if err != nil {
		return err
	}
	return r.reconciler.StatusUpdater.UpdateStatusWithRetry(func(instance *kafkaservice.KafkaService) {
		instance.Status.AkhqStatus.Nodes = controllers.GetPodNames(foundPodList.Items)
	})
}

func (r ReconcileAkhq) addDeploymentAnnotation(deployment *appsv1.Deployment, annotationName string, annotationValue string) {
	r.logger.Info(fmt.Sprintf("Add annotation '%s: %s' to deployment '%s'",
		annotationName, annotationValue, deployment.Name))
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations[annotationName] = annotationValue
}
