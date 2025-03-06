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

package kmmconfig

import (
	"context"
	"fmt"
	"github.com/Netcracker/qubership-kafka/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strconv"
	"strings"
	"time"

	kmmconfig "github.com/Netcracker/qubership-kafka/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logf.Log.WithName("controller_kmmconfig")
var periodTime, _ = strconv.Atoi(os.Getenv("KMM_CONFIG_RECONCILE_PERIOD_SECONDS"))

type configurationError struct{}

func (c configurationError) Error() string {
	return "configuration error occurred"
}

// KmmConfigReconciler reconciles a KmmConfig object
type KmmConfigReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=qubership.org,resources=kmmconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=qubership.org,resources=kmmconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=qubership.org,resources=kmmconfigs/finalizers,verbs=update

func (r *KmmConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KmmConfig")

	// Fetch the KmmConfig instance
	instance := &kmmconfig.KmmConfig{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	errorsMap := map[string][]string{"sourceDc": {}, "targetDc": {}}

	err = r.UpdateKmmConfigMap(instance, &errorsMap)
	updateStatusError := r.updateCrStatus(instance, err, errorsMap)
	if err != nil || updateStatusError != nil {
		log.Error(err, "Can not update Kafka Mirror Maker Config Map")
		return reconcile.Result{RequeueAfter: time.Second * time.Duration(periodTime)}, nil
	}
	reqLogger.Info("Kafka Mirror Maker Config Map was updated")

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KmmConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
		For(&kmmconfig.KmmConfig{}, builder.WithPredicates(statusPredicate)).
		Complete(r)
}

func (r *KmmConfigReconciler) UpdateKmmConfigMap(instance *kmmconfig.KmmConfig, errorsMap *map[string][]string) error {
	topics := instance.Spec.KmmTopics.Topics
	name := util.GetKmmConfigMapName()
	namespace := os.Getenv("OPERATOR_NAMESPACE") // Do we have another way
	kmmConfigMap := &corev1.ConfigMap{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, kmmConfigMap)
	if err != nil {
		return err
	}

	configMapContent := kmmConfigMap.Data["config"]
	properties := util.GetProperties(configMapContent)
	targetClusterNames, sourceClusterNames, err := recognizeTargetAndSourceDc(instance, properties, errorsMap)
	if err != nil {
		return err
	}
	turnOnReplicationPairs, topicsUpdatePairs, updateContent := getUpdatedContent(targetClusterNames, sourceClusterNames, properties, topics)
	replicationConfig := setUpdatedContent(turnOnReplicationPairs, topicsUpdatePairs, updateContent, configMapContent)

	identityReplicationEnabled := false
	if value, found := properties[ReplicationPolicyClassConfig]; found && value == IdentityReplicationPolicy {
		identityReplicationEnabled = true
	}
	transformationConfigurator := NewTransformationConfigurator(
		instance.Spec.KmmTopics.Transformation, fmt.Sprintf("%s_%s_", instance.Name, instance.Namespace))
	for _, sourceClusterName := range sourceClusterNames {
		for _, targetClusterName := range targetClusterNames {
			if sourceClusterName != targetClusterName {
				replicationConfig = transformationConfigurator.UpdateTransformationProperties(
					replicationConfig, sourceClusterName, targetClusterName, identityReplicationEnabled)
			}
		}
	}

	kmmConfigMap.Data["config"] = strings.Join(replicationConfig, "\n")
	if err = r.Client.Update(context.TODO(), kmmConfigMap); err != nil {
		return err
	}
	return nil
}

func recognizeTargetAndSourceDc(instance *kmmconfig.KmmConfig, properties map[string]interface{}, errorsMap *map[string][]string) ([]string, []string, error) {
	var targetDc []string
	var sourceDc []string
	targetDcCrInit := instance.Spec.KmmTopics.TargetDc
	sourceDcCrInit := instance.Spec.KmmTopics.SourceDc
	var targetDcCM []string
	if targetAsProperty, ok := properties["target.dc"]; ok {
		switch value := targetAsProperty.(type) {
		case string:
			targetDcCM = append(targetDcCM, value)
		case []string:
			targetDcCM = value
		}
	}

	clusters := properties["clusters"].([]string) // One cluster leads to error
	clustersSet := make(map[string]interface{})
	for _, cluster := range clusters {
		clustersSet[cluster] = nil
	}
	if targetDcCrInit != "" {
		targetDcCr := getTrimmedSlice(targetDcCrInit)
		if err := checkTargetDc(targetDcCr, targetDcCM, clustersSet, errorsMap); err != nil {
			return nil, nil, err
		}
		targetDc = targetDcCr
	}
	if targetDc == nil {
		if targetDcCM != nil {
			targetDc = targetDcCM
		} else {
			err := configurationError{}
			log.Error(err, "Can not resolve target data center")
			addMessageToErrorMap(errorsMap, "targetDc", "Can not resolve target data center")
			return nil, nil, err
		}
	}

	if sourceDcCrInit != "" {
		sourceDcCr := getTrimmedSlice(sourceDcCrInit)
		for _, dc := range sourceDcCr {
			if _, ok := clustersSet[dc]; !ok {
				err := configurationError{}
				log.Error(err, fmt.Sprintf("Source dc - %s from Custom Resource is not contained in the KMM2 Config Map", dc))
				addMessageToErrorMap(errorsMap, "sourceDc", fmt.Sprintf("%s - 'clusters' property does not contain current source dc", dc))
				return nil, nil, err
			}
		}
		sourceDc = sourceDcCr
	}
	if sourceDc == nil {
		for _, dc := range clusters {
			if !contains(dc, targetDc) {
				sourceDc = append(sourceDc, dc)
			}
		}
	}
	return targetDc, sourceDc, nil
}

func checkTargetDc(crDc []string, configMapDc []string, clustersSet map[string]interface{}, errorsMap *map[string][]string) error {
	if configMapDc == nil {
		for _, dc := range crDc {
			if _, ok := clustersSet[dc]; !ok {
				err := configurationError{}
				log.Error(err, fmt.Sprintf("Target dc - %s from Custom Resource is not contained in the KMM2 Config Map neither in 'target.dc' property nor in 'clusters' property", dc))
				addMessageToErrorMap(errorsMap, "targetDc", fmt.Sprintf("%s - neither 'target.dc' property nor 'clusters' property contains current target dc", dc))
				return err
			}
		}
		return nil
	}
	for _, cr := range crDc {
		if !contains(cr, configMapDc) {
			err := configurationError{}
			log.Error(err, fmt.Sprintf("Target dc %s from Custom Resource is not contained in the KMM2 Config Map target.dc property", cr))
			addMessageToErrorMap(errorsMap, "targetDc", fmt.Sprintf("%s - target.dc property does not contain target dc", cr))
			return err
		}
	}
	return nil
}

func contains(value string, list []string) bool {
	if len(list) == 0 {
		return false
	}
	for _, tmp := range list {
		if value == tmp {
			return true
		}
	}
	return false
}

func getUpdatedContent(targetDc []string, sourceDc []string, properties map[string]interface{}, topics string) ([]string, map[string]string, string) {
	var turnOnReplicationPairs []string
	updateTopicLines := make(map[string]string)
	var newLines []string
	topicsAsSlice := getTrimmedSlice(topics)
	pairs := getDcPairs(targetDc, sourceDc)
	for _, pair := range pairs {
		enabledPair := fmt.Sprintf("%s.enabled", pair)
		if value, ok := properties[enabledPair]; ok {
			if value.(string) == "false" {
				turnOnReplicationPairs = append(turnOnReplicationPairs, enabledPair)
			}
		} else {
			newLines = append(newLines, fmt.Sprintf("%s = true", enabledPair))
		}
		topicsPair := fmt.Sprintf("%s.topics", pair)
		if existsTopics, ok := properties[topicsPair]; ok {
			existingTopicsSlice := makeSlice(existsTopics)
			updateTopicLines[topicsPair] = mergeTopics(existingTopicsSlice, topicsAsSlice)
		} else {
			newLines = append(newLines, fmt.Sprintf("%s = %s", topicsPair, topics))
		}
	}
	return turnOnReplicationPairs, updateTopicLines, strings.Join(newLines, "\n")
}

func getDcPairs(targetDc []string, sourceDc []string) []string {
	var pairs []string
	for _, source := range sourceDc {
		for _, target := range targetDc {
			if source != target {
				pairs = append(pairs, NewReplicationFlow(source, target))
			}
		}
	}
	return pairs
}

func makeSlice(iStringInterface interface{}) []string {
	switch value := iStringInterface.(type) {
	case string:
		return []string{value}
	case []string:
		return value
	default:
		return []string{}
	}
}

func mergeTopics(oldTopics []string, newTopics []string) string {
	oldTopicsSet := make(map[string]interface{})
	for _, oldTopic := range oldTopics {
		oldTopicsSet[oldTopic] = nil
	}

	for _, newTopic := range newTopics {
		if _, ok := oldTopicsSet[newTopic]; !ok {
			oldTopics = append(oldTopics, newTopic)
		}
	}
	return strings.Join(oldTopics, ",")
}

func setUpdatedContent(turnOnReplicationPairs []string, topicsUpdatePairs map[string]string, updateContent string, configMapContent string) []string {
	cmLines := strings.Split(configMapContent, "\n")
	setEnabledReplication(turnOnReplicationPairs, &cmLines)
	updateTopics(topicsUpdatePairs, &cmLines)
	if updateContent != "" {
		setNewProperties(updateContent, &cmLines)
	}
	return cmLines
}

func setEnabledReplication(turnOnReplicationPairs []string, cmLines *[]string) {
	for _, enabledReplication := range turnOnReplicationPairs {
		for i, line := range *cmLines {
			if strings.HasPrefix(line, enabledReplication) {
				(*cmLines)[i] = fmt.Sprintf("%s = true", enabledReplication)
			}
		}
	}
}

func updateTopics(topicsUpdatePairs map[string]string, cmLines *[]string) {
	for key, value := range topicsUpdatePairs {
		for i, line := range *cmLines {
			if strings.HasPrefix(line, key) {
				(*cmLines)[i] = fmt.Sprintf("%s = %s", key, value)
			}
		}
	}
}

func setNewProperties(updateContent string, cmLines *[]string) {
	blackListExists := false
	blackListPosition := 0
	for i, line := range *cmLines {
		if strings.HasPrefix(line, "topics.blacklist") {
			blackListExists = true
			blackListPosition = i
			break
		}
	}
	updateContentAsSlice := strings.Split(updateContent, "\n")
	if blackListExists {
		propertiesBefore := (*cmLines)[:blackListPosition]
		var propertiesAfter []string
		propertiesAfter = append(propertiesAfter, (*cmLines)[blackListPosition:]...)
		propertiesBefore = append(propertiesBefore, updateContentAsSlice...)
		propertiesBefore = append(propertiesBefore, propertiesAfter...)
		*cmLines = propertiesBefore
	} else {
		*cmLines = append(*cmLines, updateContentAsSlice...)
	}
}

func getTrimmedSlice(source string) []string {
	sources := strings.Split(source, ",")
	for i, src := range sources {
		sources[i] = strings.Trim(src, " ")
	}
	return sources
}

func addMessageToErrorMap(errorsMap *map[string][]string, key string, message string) {
	if valueSlice, ok := (*errorsMap)[key]; ok {
		valueSlice = append(valueSlice, message)
		(*errorsMap)[key] = valueSlice
	}
}

func (r *KmmConfigReconciler) updateCrStatus(instance *kmmconfig.KmmConfig, err error, errorsMap map[string][]string) error {
	instance.Status.IsProcessed = true

	message := generateErrorMessage(err, errorsMap)
	instance.Status.ProblemTopics = message
	updateStatusErr := r.Client.Status().Update(context.TODO(), instance)
	if updateStatusErr != nil {
		log.Error(err, "Error occurred during custom resource status update")
	}
	return updateStatusErr
}

func generateErrorMessage(err error, errorsMap map[string][]string) string {
	message := ""
	if err != nil {
		message = "topics were not added"
	}
	targetErrs := strings.Join(errorsMap["targetDc"], "\n")
	if targetErrs != "" {
		message = fmt.Sprintf("%s; Problem target dc: %s", message, targetErrs)
	}
	sourceErrs := strings.Join(errorsMap["targetDc"], "\n")
	if sourceErrs != "" {
		message = fmt.Sprintf("%s; Problem source dc: %s", message, sourceErrs)
	}
	return message
}
