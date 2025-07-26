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

package controllers

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"

	"github.com/Netcracker/qubership-kafka/operator/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	log = logf.Log.WithName("controller_reconciler")
)

type Reconciler struct {
	Client           client.Client
	Scheme           *runtime.Scheme
	ResourceVersions map[string]string
	ResourceHashes   map[string]string
	ApiGroup         string
}

func (r *Reconciler) FindPodList(namespace string, podLabels map[string]string) (*corev1.PodList, error) {
	foundPodList := &corev1.PodList{}
	err := r.Client.List(context.TODO(), foundPodList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(podLabels),
	})
	return foundPodList, err
}

func GetFirstAvailablePod(pods *corev1.PodList) *corev1.Pod {
	for _, pod := range pods.Items {
		conditions := pod.Status.Conditions
		for _, condition := range conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return &pod
			}
		}
	}
	return nil
}

func GetPodNames(pods []corev1.Pod) []string {
	var nodes []string
	for _, pod := range pods {
		nodes = append(nodes, pod.Labels["name"])
	}
	return nodes
}

func GetActualPodNames(pods []corev1.Pod) []string {
	var names []string
	for _, pod := range pods {
		names = append(names, pod.Name)
	}
	return names
}

func ApiGroupMatches(apiVersion string, targetApiGroup string) bool {
	apiGroup := strings.Split(apiVersion, "/")[0]
	return apiGroup == targetApiGroup
}

func (r *Reconciler) GetNodeLabel(nodeName string, labelName string, logger logr.Logger) (string, error) {
	// creates the Kubernetes config
	config := r.GetOperatorClusterConfig()
	// creates the client set
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}
	foundNode, err := clientSet.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Cannot get node [%s] info", nodeName))
		return "", err
	}
	labelValue := foundNode.Labels[labelName]
	if labelValue == "" {
		return "", fmt.Errorf("Node [%s] does not have label [%s]", nodeName, labelName)
	}
	return labelValue, nil
}

func (r *Reconciler) GetOperatorClusterConfig() *rest.Config {
	return kubeconfig.GetConfigOrDie()
}

// CreateOrUpdateServiceAccount creates the serviceAccount if it doesn't exist and updates otherwise
func (r *Reconciler) CreateOrUpdateServiceAccount(serviceAccount *corev1.ServiceAccount, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Checking Existence of [%s] service account", serviceAccount.Name))
	foundService := &corev1.ServiceAccount{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name: serviceAccount.Name, Namespace: serviceAccount.Namespace,
	}, foundService)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new service account",
			"ServiceAccount.Namespace", serviceAccount.Namespace, "ServiceAccount.Name", serviceAccount.Name)
		return r.Client.Create(context.TODO(), serviceAccount)
	} else if err != nil {
		return err
	} else {
		logger.Info("Updating the found service account",
			"ServiceAccount.Namespace", serviceAccount.Namespace, "ServiceAccount.Name", serviceAccount.Name)
		foundService.Labels = util.JoinMaps(foundService.Labels, serviceAccount.Labels)
		return r.Client.Update(context.TODO(), foundService)
	}
}

// CreateOrUpdateService creates the service if it doesn't exist and updates otherwise
func (r *Reconciler) CreateOrUpdateService(service *corev1.Service, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Checking Existence of [%s] service", service.Name))
	foundService := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new service",
			"Service.Namespace", service.Namespace, "Service.Name", service.Name)
		return r.Client.Create(context.TODO(), service)
	} else if err != nil {
		return err
	} else {
		logger.Info("Updating the found service",
			"Service.Namespace", service.Namespace, "Service.Name", service.Name)
		service.ResourceVersion = foundService.ResourceVersion
		if foundService.Spec.Type == corev1.ServiceTypeClusterIP {
			service.Spec.ClusterIP = foundService.Spec.ClusterIP
		}
		return r.Client.Update(context.TODO(), service)
	}
}

func (r *Reconciler) DeleteService(service *corev1.Service, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Checking Existence of [%s] service", service.Name))
	foundService := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Service not exist, nothing to delete",
			"Service.Namespace", service.Namespace, "Service.Name", service.Name)
		return nil
	} else if err != nil {
		return err
	} else {
		logger.Info("Deleting the found service",
			"Service.Namespace", service.Namespace, "Service.Name", service.Name)
		service.ResourceVersion = foundService.ResourceVersion
		return r.Client.Delete(context.TODO(), service)
	}
}

func (r *Reconciler) GetServiceIp(namespace, serviceName string) (string, error) {
	foundService := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: serviceName, Namespace: namespace}, foundService)
	return foundService.Spec.ClusterIP, err
}

func (r *Reconciler) CreatePersistentVolumeClaim(persistentVolumeClaim *corev1.PersistentVolumeClaim, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Checking Existence of [%s] persistent volume claim", persistentVolumeClaim.Name))
	foundPersistentVolumeClaim := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(context.TODO(),
		types.NamespacedName{Name: persistentVolumeClaim.Name, Namespace: persistentVolumeClaim.Namespace},
		foundPersistentVolumeClaim)
	if err == nil {
		logger.Info("Clean 'ownerReferences' for existing persistent volume claim",
			"PersistentVolumeClaim.Namespace", persistentVolumeClaim.Namespace, "PersistentVolumeClaim.Name", persistentVolumeClaim.Name)
		foundPersistentVolumeClaim.ObjectMeta.OwnerReferences = nil
		foundPersistentVolumeClaim.Labels = util.JoinMaps(foundPersistentVolumeClaim.Labels, persistentVolumeClaim.Labels)
		err = r.Client.Update(context.TODO(), foundPersistentVolumeClaim)
		if err != nil {
			// There is no ability to update PVC for some environments.
			log.Error(err, "Error occurred during updating existing PVC, skipping it")
			return nil
		}
	} else if errors.IsNotFound(err) {
		logger.Info("Creating a new persistent volume claim",
			"PersistentVolumeClaim.Namespace", persistentVolumeClaim.Namespace, "PersistentVolumeClaim.Name", persistentVolumeClaim.Name)
		err = r.Client.Create(context.TODO(), persistentVolumeClaim)
	}
	return err
}

func (r *Reconciler) DeletePersistentVolumeClaim(persistentVolumeClaim *corev1.PersistentVolumeClaim, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Checking Existence of [%s] persistent volume claim", persistentVolumeClaim.Name))
	foundPersistentVolumeClaim := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(context.TODO(),
		types.NamespacedName{Name: persistentVolumeClaim.Name, Namespace: persistentVolumeClaim.Namespace},
		foundPersistentVolumeClaim)
	if err == nil {
		err = r.Client.Delete(context.TODO(), foundPersistentVolumeClaim)
		if err != nil {
			log.Error(err, "Error occurred during deleting existing PVC")
			return err
		}
	} else if errors.IsNotFound(err) {
		logger.Info("Persistent volume claim not exist, nothing to delete",
			"PersistentVolumeClaim.Namespace", persistentVolumeClaim.Namespace, "PersistentVolumeClaim.Name", persistentVolumeClaim.Name)
		err = nil
	}
	return err
}

func (r *Reconciler) CreateOrUpdateDeployment(deployment *appsv1.Deployment, logger logr.Logger) error {
	_, err := r.FindDeployment(deployment.Name, deployment.Namespace, logger)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Deployment",
			"Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return r.Client.Create(context.TODO(), deployment)
	} else if err != nil {
		return err
	} else {
		logger.Info("Updating the found Deployment",
			"Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return r.Client.Update(context.TODO(), deployment)
	}
}

func (r *Reconciler) DeleteDeployment(deployment *appsv1.Deployment, logger logr.Logger) error {
	_, err := r.FindDeployment(deployment.Name, deployment.Namespace, logger)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Deployment not found, nothing to delete",
			"Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return nil
	} else if err != nil {
		return err
	} else {
		logger.Info("Deleting the found Deployment",
			"Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return r.Client.Delete(context.TODO(), deployment)
	}
}

func (r *Reconciler) FindDeployment(name string, namespace string, logger logr.Logger) (*appsv1.Deployment, error) {
	logger.Info(fmt.Sprintf("Checking Existence of [%s] deployment", name))
	foundDeployment := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundDeployment)
	return foundDeployment, err
}

func (r *Reconciler) FindDeploymentList(namespace string, deploymentLabels map[string]string) (*appsv1.DeploymentList, error) {
	foundDeploymentList := &appsv1.DeploymentList{}
	err := r.Client.List(context.TODO(), foundDeploymentList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(deploymentLabels),
	})
	return foundDeploymentList, err
}

// GetKafkaLabels configures common labels for Kafka deployments
func GetKafkaLabels(customResourceName string) map[string]string {
	return map[string]string{
		"component":   "kafka",
		"clusterName": customResourceName,
	}
}

// GetMirrorMakerLabels configures common labels for Kafka Mirror Maker deployments
func GetMirrorMakerLabels(customResourceName string) map[string]string {
	return map[string]string{
		"component":   "kafka-mm",
		"clusterName": fmt.Sprintf("%s-mirror-maker", customResourceName),
	}
}

// GetBackupDaemonLabels configures common labels for Backup Daemon deployments
func GetBackupDaemonLabels(customResourceName string) map[string]string {
	return map[string]string{
		"component": "kafka-backup-daemon",
		"name":      fmt.Sprintf("%s-backup-daemon", customResourceName),
	}
}

func (r *Reconciler) FindKafkaDeployments(cr metav1.Object) (*appsv1.DeploymentList, error) {
	kafkaLabels := GetKafkaLabels(cr.GetName())
	return r.FindDeploymentList(cr.GetNamespace(), kafkaLabels)
}

func (r *Reconciler) GetDeploymentParameter(deployment appsv1.Deployment, parameterName string) string {
	envVars := deployment.Spec.Template.Spec.Containers[0].Env
	for _, envVar := range envVars {
		if envVar.Name == parameterName {
			return envVar.Value
		}
	}
	return ""
}

// IsDeploymentReady checks if all deployment replicas are available
func IsDeploymentReady(deployment appsv1.Deployment) bool {
	availableReplicas := util.Min(deployment.Status.ReadyReplicas, deployment.Status.UpdatedReplicas)
	return *deployment.Spec.Replicas == availableReplicas
}

// AreDeploymentsReady finds deployments matching the list of labels and checks their readiness
func (r *Reconciler) AreDeploymentsReady(deploymentLabels map[string]string, namespace string, logger logr.Logger) bool {
	deployments, err := r.FindDeploymentList(namespace, deploymentLabels)
	if err != nil {
		logger.Error(err, "Cannot check deployments status")
		return false
	}

	for _, deployment := range deployments.Items {
		if !IsDeploymentReady(deployment) {
			logger.Info(fmt.Sprintf("%s deployment is not ready yet", deployment.Name))
			return false
		}
	}
	return true
}

func (r *Reconciler) CreateOrUpdateConfigMap(configMap *corev1.ConfigMap, logger logr.Logger) error {
	_, err := r.FindConfigMap(configMap.Name, configMap.Namespace, logger)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new config map",
			"ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		return r.Client.Create(context.TODO(), configMap)
	} else if err != nil {
		return err
	} else {
		logger.Info("Updating the found config map",
			"ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		return r.Client.Update(context.TODO(), configMap)
	}
}

func (r *Reconciler) CreateConfigMapIfNotExist(configMap *corev1.ConfigMap, logger logr.Logger) error {
	_, err := r.FindConfigMap(configMap.Name, configMap.Namespace, logger)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new config map",
			"ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		return r.Client.Create(context.TODO(), configMap)
	}
	return err
}

// FindConfigMap finds configuration map by name
func (r *Reconciler) FindConfigMap(name string, namespace string, logger logr.Logger) (*corev1.ConfigMap, error) {
	logger.Info(fmt.Sprintf("Checking Existence of [%s] config map", name))
	foundConfigMap := &corev1.ConfigMap{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundConfigMap)
	return foundConfigMap, err
}

func (r *Reconciler) GetAllConfigMapsFromNamespace(namespace string, logger logr.Logger) (*corev1.ConfigMapList, error) {
	configMaps := &corev1.ConfigMapList{}

	err := r.Client.List(context.TODO(), configMaps, &client.ListOptions{
		Namespace: namespace,
	})

	if err != nil {
		logger.Error(err, "Error while loading all config maps from namespace:", namespace)
		return nil, err
	}

	return configMaps, err
}

func (r *Reconciler) DeleteConfigMapByName(name string, namespace string, logger logr.Logger) error {
	cm, err := r.FindConfigMap(name, namespace, logger)

	if err != nil && errors.IsNotFound(err) {
		logger.Info("ConfigMap not found, nothing to delete",
			"ConfigMap.Name", name, "Deployment.Namespace", namespace)
		return nil
	}

	if err != nil {
		return err
	}

	logger.Info("Deleting the found ConfigMap", "ConfigMap.Name", name, "Deployment.Namespace", namespace)
	return r.Client.Delete(context.TODO(), cm)
}

// CreateSecret creates secret if it does not exist; returns an error if any operation failed
func (r *Reconciler) CreateSecret(secret *corev1.Secret, logger logr.Logger) error {
	_, err := r.FindSecret(secret.Name, secret.Namespace, logger)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new secret",
			"Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Client.Create(context.TODO(), secret)
	}
	return err
}

func (r *Reconciler) RestoreSpecFields(secret *corev1.Secret, keys []string, logger logr.Logger) error {
	oldSecret, err := r.FindSecret(secret.Name, secret.Namespace, logger)
	if err != nil && errors.IsNotFound(err) {
		return nil
	}

	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	oldSecData := oldSecret.Data
	secData := secret.StringData
	for _, key := range keys {
		secData[key] = string(oldSecData[key])
	}

	secret.StringData = secData
	return nil
}

func (r *Reconciler) CreateOrUpdateSecret(secret *corev1.Secret, logger logr.Logger) error {
	_, err := r.FindSecret(secret.Name, secret.Namespace, logger)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new secret",
			"Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		return r.Client.Create(context.TODO(), secret)
	} else if err != nil {
		return err
	} else {
		logger.Info("Updating the found secret",
			"Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		return r.Client.Update(context.TODO(), secret)
	}
}

// FindSecret finds secret by name
func (r *Reconciler) FindSecret(name string, namespace string, logger logr.Logger) (*corev1.Secret, error) {
	logger.Info(fmt.Sprintf("Checking Existence of [%s] secret", name))
	foundSecret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundSecret)
	return foundSecret, err
}

func (r *Reconciler) WatchSecret(secretName string, obj runtime.Object, logger logr.Logger) (*corev1.Secret, error) {
	cr, err := toUnstructured(obj)
	if err != nil {
		return nil, err
	}
	secret, err := r.FindSecret(secretName, cr.GetNamespace(), logger)
	if err != nil {
		return nil, err
	} else {
		if err := r.SetControllerReference(cr, secret, r.Scheme); err != nil {
			return nil, err
		}
		if err := r.UpdateSecret(secret, logger); err != nil {
			return nil, err
		}
	}
	return secret, nil
}

// SetControllerReference Extension for controllerutil.SetControllerReference to clean up previous owner reference
func (r *Reconciler) SetControllerReference(owner, controlled runtime.Object, scheme *runtime.Scheme) error {
	ownerStructured, err := toUnstructured(owner)
	if err != nil {
		return err
	}
	controlledStructured, err := toUnstructured(controlled)
	if err != nil {
		return err
	}
	// Check if there's an existing owner reference
	if existing := metav1.GetControllerOf(controlledStructured); existing != nil && !ReferSameObject(existing.Name, existing.APIVersion, existing.Kind, ownerStructured.GetName(), ownerStructured.GetAPIVersion(), ownerStructured.GetKind()) {
		controlledStructured.SetOwnerReferences(nil)
	}
	return controllerutil.SetControllerReference(ownerStructured, controlledStructured, scheme)
}

func (r *Reconciler) UpdateSecret(secret *corev1.Secret, logger logr.Logger) error {
	foundSecret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(),
		types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace},
		foundSecret)
	if err != nil && errors.IsNotFound(err) {
		logger.Error(err, fmt.Sprintf("Secret [%s] must exist", secret.Name))
		return err
	} else if err != nil {
		return err
	} else {
		logger.Info("Updating the found secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		return r.Client.Update(context.TODO(), secret)
	}
}

// CleanSecretData
func (r *Reconciler) CleanSecretData(secret *corev1.Secret, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Cleaning data of [%s] secret", secret.Name))
	secret.Data = nil
	return r.Client.Update(context.TODO(), secret)
}

// GetSslCertificates get ssl certificates from secret
func (r *Reconciler) GetSslCertificates(secretName, namespace string, logger logr.Logger) (*SslCertificates, error) {
	foundSecret, err := r.FindSecret(secretName, namespace, logger)
	if err != nil {
		return nil, err
	}
	caCert := foundSecret.Data["ca.crt"]
	tlsCert := foundSecret.Data["tls.crt"]
	tlsKey := foundSecret.Data["tls.key"]
	if len(caCert) == 0 {
		return nil, fmt.Errorf("TLS certificates must be provided by secret with name: %s", secretName)
	}
	return &SslCertificates{CaCert: caCert, TlsCert: tlsCert, TlsKey: tlsKey}, nil
}

func (r *Reconciler) ScaleDeployment(name string, replicas int32, namespace string, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Scaling [%s] deployment to [%d] replicas", name, replicas))
	foundDeployment, err := r.FindDeployment(name, namespace, logger)
	if err == nil {
		foundDeployment.Spec.Replicas = &replicas
		return r.Client.Update(context.TODO(), foundDeployment)
	}
	return err
}

func ReferSameObject(aName string, aGroup string, aKind string, bName string, bGroup string, bKind string) bool {
	aGV, err := schema.ParseGroupVersion(aGroup)
	if err != nil {
		return false
	}

	bGV, err := schema.ParseGroupVersion(bGroup)
	if err != nil {
		return false
	}

	return aGV.Group == bGV.Group && aKind == bKind && aName == bName
}

// ToUnstructured converts runtime.Object into *unstructured.Unstructured.
func toUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u, nil
	}
	if obj == nil {
		return nil, fmt.Errorf("to unstructured: nil object")
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("to unstructured: %w", err)
	}
	return &unstructured.Unstructured{Object: m}, nil
}
