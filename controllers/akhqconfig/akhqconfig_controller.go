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

package akhqconfig

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	akhqconfigv1 "github.com/Netcracker/qubership-kafka/api/v1"
	akhqproto "github.com/Netcracker/qubership-kafka/controllers/akhqprotobuf"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("controller_akhq_config")

const (
	akhqFinalizer               = "qubership.org/akhq-config-controller"
	internalServerError         = "internal server error, config was not applied"
	protobufConfigurationCMName = "akhq-protobuf-configuration"
	decodingValidationError     = "can not decode descriptor-file-base64 config which is associated with [%s] regular expression"
	duplicateValidationError    = "this regular expression occurs twice - [%s]"
)

type TopicMapping struct {
	TopicRegex           string `yaml:"topic-regex"`
	Name                 string `yaml:"name"`
	DescriptorFile       string `yaml:"descriptor-file"`
	DescriptorFileBase64 string `yaml:"descriptor-file-base64,omitempty"`
	KeyMessageType       string `yaml:"key-message-type,omitempty"`
	ValueMessageType     string `yaml:"value-message-type"`
}

type DeserializationConfig struct {
	Deserialization struct {
		Protobuf struct {
			DescriptorsFolder string         `yaml:"descriptors-folder"`
			TopicsMapping     []TopicMapping `yaml:"topics-mapping"`
		} `yaml:"protobuf"`
	} `yaml:"deserialization"`
}

type AkhqConfigReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Namespace string
}

//+kubebuilder:rbac:groups=qubership.org,resources=akhqconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=qubership.org,resources=akhqconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=qubership.org,resources=akhqconfigs/finalizers,verbs=update

func (r *AkhqConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling AkhqConfig")

	instance := &akhqconfigv1.AkhqConfig{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.DeletionTimestamp.IsZero() {
		isValid, description := r.validate(instance)
		if !isValid {
			reqLogger.Info(fmt.Sprintf("Validation was failed with the following description - %s", description))
			if err = r.updateCrStatus(instance, true, false, description); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}
		if !util.Contains(akhqFinalizer, instance.GetFinalizers()) {
			controllerutil.AddFinalizer(instance, akhqFinalizer)
			if err := r.Client.Update(context.TODO(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}
	} else {
		if util.Contains(akhqFinalizer, instance.GetFinalizers()) {
			if err := r.deleteConfig(instance, reqLogger); err != nil {
				return reconcile.Result{}, err
			}
			controllerutil.RemoveFinalizer(instance, akhqFinalizer)
			if err := r.Client.Update(context.TODO(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}
	if err = r.applyConfig(instance, reqLogger); err != nil {
		reqLogger.Error(err, fmt.Sprintf("Can not apply config for current AkhqConfig CR, name - [%s], namespace - [%s]", instance.Name, instance.Namespace))
		if err := r.updateCrStatus(instance, false, true, internalServerError); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	if err = r.updateCrStatus(instance, false, true, ""); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *AkhqConfigReconciler) validate(instance *akhqconfigv1.AkhqConfig) (bool, string) {
	crConfigs := instance.Spec.Configs
	return r.validateCrConfig(crConfigs)
}

func (r *AkhqConfigReconciler) validateCrConfig(crConfigs []akhqconfigv1.Config) (bool, string) {
	regexSet := make(map[string]bool)
	for _, config := range crConfigs {
		regex := strings.ReplaceAll(config.TopicRegex, ",", "|")
		if _, ok := regexSet[regex]; !ok {
			regexSet[regex] = true
			_, err := base64.StdEncoding.DecodeString(config.DescriptorFileBase64)
			if err != nil {
				return false, fmt.Sprintf(decodingValidationError, config.TopicRegex)
			}
		} else {
			return false, fmt.Sprintf(duplicateValidationError, config.TopicRegex)
		}
	}
	return true, ""
}

func (r *AkhqConfigReconciler) deleteConfig(instance *akhqconfigv1.AkhqConfig, reqLogger logr.Logger) error {

	reqLogger.Info("Start deleting configs")

	//anyway the config map name is hardcoded
	deserializationCM, err := r.getConfigMap(protobufConfigurationCMName, r.Namespace)
	if err != nil {
		return err
	}

	dc, err := r.getDeserializationConfig(deserializationCM)
	if err != nil {
		return err
	}
	crFullName := fmt.Sprintf("%s-%s", instance.Name, instance.Namespace)
	existingConfigs := dc.Deserialization.Protobuf.TopicsMapping

	existingConfigs, mapToDelete := r.deleteDeserializationConfig(existingConfigs, crFullName)
	if err := r.deleteBinaryConfigMap(r.Namespace, mapToDelete, reqLogger); err != nil {
		return err
	}

	dc.Deserialization.Protobuf.TopicsMapping = existingConfigs
	if err = r.updateDeserializationConfigMap(deserializationCM, *dc); err != nil {
		return err
	}

	reqLogger.Info("Deleting has finished successfully")
	return nil
}

func (r *AkhqConfigReconciler) applyConfig(instance *akhqconfigv1.AkhqConfig,
	reqLogger logr.Logger) error {
	reqLogger.Info("Start applying configs")

	crConfigs := instance.Spec.Configs
	crConfigsMap := make(map[string]akhqconfigv1.Config)
	for _, value := range crConfigs {
		convertedString := strings.ReplaceAll(value.TopicRegex, ",", "|")
		value.TopicRegex = convertedString
		crConfigsMap[value.TopicRegex] = value
	}
	deserializationCM, err := akhqproto.CreateProtobufConfigMapIfNotExist(r.Namespace, r.Client, make(map[string]string), nil, nil)
	if err != nil {
		return err
	}

	dc, err := r.getDeserializationConfig(deserializationCM)
	if err != nil {
		return err
	}

	existingConfigs := dc.Deserialization.Protobuf.TopicsMapping
	crFullName := fmt.Sprintf("%s-%s", instance.Name, instance.Namespace)

	existingConfigs, mapToUpdate := r.fullUpdateDeserializationConfig(existingConfigs, crConfigsMap, crFullName)

	if err = r.applyBinaryConfigMap(r.Namespace, mapToUpdate, deserializationCM.Labels, reqLogger); err != nil {
		return err
	}

	dc.Deserialization.Protobuf.TopicsMapping = existingConfigs
	if err = r.updateDeserializationConfigMap(deserializationCM, *dc); err != nil {
		return err
	}
	reqLogger.Info("Applying has finished successfully")

	return nil
}

func (r *AkhqConfigReconciler) updateCrStatus(instance *akhqconfigv1.AkhqConfig, isInvalid, isProcessed bool, problems string) error {
	statusUpdater := NewStatusUpdater(r.Client, instance)
	return statusUpdater.UpdateStatusWithRetry(func(instance *akhqconfigv1.AkhqConfig) {
		instance.Status.IsInvalid = isInvalid
		instance.Status.IsProcessed = isProcessed
		instance.Status.Problems = problems
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *AkhqConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	statusPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&akhqconfigv1.AkhqConfig{}, builder.WithPredicates(statusPredicate)).
		Complete(r)
}
