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

package provider

import (
	"errors"
	"fmt"
	kafkaservice "github.com/Netcracker/qubership-kafka/api/v1"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

const (
	persistentVolumeClaimPattern           = "pvc-%s-%d"
	defaultClockSkew                       = 10
	defaultJwkSourceType                   = "jwks"
	defaultJwksConnectionTimeout           = 1000
	defaultJwksReadTimeout                 = 1000
	defaultJwksSizeLimit                   = 51200
	defaultTerminationGracePeriod          = 180
	defaultVolumeName                      = "emptyDir"
	defaultRollbackTimeout                 = 3600
	defaultHealthCheckTimeout              = 30
	defaultAllBrokersStartTimeoutSeconds   = 600
	defaultTopicReassignmentTimeoutSeconds = 300
	defaultBrokerDeploymentScaleInEnabled  = false
	zooKeeperClusterID                     = "U5tHX5uHQnmsniDS54EF_w"
)

type KafkaResourceProvider struct {
	cr     *kafkaservice.Kafka
	logger logr.Logger
	spec   kafkaservice.KafkaSpec
}

func NewKafkaResourceProvider(cr *kafkaservice.Kafka, logger logr.Logger) KafkaResourceProvider {
	return KafkaResourceProvider{cr: cr, spec: cr.Spec, logger: logger}
}

func (krp KafkaResourceProvider) GetKafkaLabels() map[string]string {
	labels := make(map[string]string)
	labels["name"] = krp.cr.Name
	labels["app.kubernetes.io/name"] = krp.cr.Name
	labels = util.JoinMaps(util.JoinMaps(krp.GetSelectorLabels(), labels), krp.spec.DefaultLabels)
	return labels
}

func (krp KafkaResourceProvider) GetSelectorLabels() map[string]string {
	return map[string]string{
		"component":   "kafka",
		"clusterName": krp.cr.Name,
	}
}

func (krp KafkaResourceProvider) GetKafkaCustomLabels(kafkaLabels map[string]string) map[string]string {
	kafkaCustomLabels := krp.spec.CustomLabels
	return util.JoinMaps(kafkaCustomLabels, kafkaLabels)
}

func (krp KafkaResourceProvider) GetServiceName() string {
	return krp.cr.Name
}

func (krp KafkaResourceProvider) GetServiceAccountName() string {
	return krp.GetServiceName()
}

func (krp KafkaResourceProvider) NewKafkaClientServiceForCR() *corev1.Service {
	kafkaLabels := krp.GetKafkaLabels()
	selectorLabels := krp.GetSelectorLabels()
	ports := []corev1.ServicePort{
		{
			Name:     "kafka-client",
			Port:     9092,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     "external-kafka-client",
			Port:     9094,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     "nonencrypted-kafka-client",
			Port:     9095,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     "jolokia-http",
			Port:     9087,
			Protocol: corev1.ProtocolTCP,
		},
	}
	clientService := newServiceForCR(krp.cr.Name, krp.cr.Namespace, kafkaLabels, selectorLabels, ports)
	return clientService
}

func (krp KafkaResourceProvider) NewKafkaDomainClientServiceForCR() *corev1.Service {
	kafkaLabels := krp.GetKafkaLabels()
	domainServiceName := fmt.Sprintf("%s-broker", krp.cr.Name)
	selectorLabels := krp.GetSelectorLabels()
	ports := []corev1.ServicePort{
		{
			Name:     "kafka-inter-broker",
			Port:     9093,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     "jolokia-http",
			Port:     9087,
			Protocol: corev1.ProtocolTCP,
		},
	}
	if krp.cr.Spec.Kraft.Enabled {
		ports = append(ports, []corev1.ServicePort{
			{
				Name:     "kafka-controller",
				Port:     9096,
				Protocol: corev1.ProtocolTCP,
			},
		}...)
	}
	domainClientService := newDomainServiceForCR(domainServiceName, krp.cr.Namespace, kafkaLabels, selectorLabels, ports)
	return domainClientService
}

func (krp KafkaResourceProvider) NewKafkaBrokerServiceForCR(brokerId int) *corev1.Service {
	serviceName := fmt.Sprintf("%s-%d", krp.cr.Name, brokerId)
	kafkaLabels := krp.GetKafkaLabels()
	kafkaLabels["name"] = serviceName
	selectorLabels := krp.GetSelectorLabels()
	selectorLabels["name"] = serviceName
	ports := []corev1.ServicePort{
		{
			Name:     "kafka-client",
			Port:     9092,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     "external-kafka-client",
			Port:     9094,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     "nonencrypted-kafka-client",
			Port:     9095,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     "jolokia-http",
			Port:     9087,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     "prometheus-http",
			Port:     8080,
			Protocol: corev1.ProtocolTCP,
		},
	}
	kafkaBrokerService := newServiceForBroker(serviceName, krp.cr.Namespace, kafkaLabels, selectorLabels, ports)
	return kafkaBrokerService
}

func (krp KafkaResourceProvider) NewKafkaControllerServiceForCR() *corev1.Service {
	serviceName := fmt.Sprintf("%s-%s", krp.cr.Name, "kraft-controller")
	kafkaLabels := krp.GetKafkaLabels()
	kafkaLabels["name"] = serviceName
	kafkaLabels["component"] = "kafka-controller"
	delete(kafkaLabels, "clusterName")
	selectorLabels := make(map[string]string)
	selectorLabels["component"] = "kafka-controller"
	selectorLabels["name"] = serviceName
	ports := []corev1.ServicePort{
		{
			Name:     "kafka-kraft-controller",
			Port:     9092,
			Protocol: corev1.ProtocolTCP,
		},
	}
	kafkaControllerService := newServiceForBroker(serviceName, krp.cr.Namespace, kafkaLabels, selectorLabels, ports)
	return kafkaControllerService
}

// NewKafkaPersistentVolumeClaimForCR returns a persistent volume claim for specified Kafka server
func (krp KafkaResourceProvider) NewKafkaPersistentVolumeClaimForCR(brokerId int) *corev1.PersistentVolumeClaim {
	var spec corev1.PersistentVolumeClaimSpec
	var volumesCount = len(krp.spec.Storage.Volumes)
	if err := krp.checkStorageClassDefinition(volumesCount); err != nil {
		return nil
	}
	if volumesCount > 0 {
		krp.logger.Info("Persistent volume claims are created by volume names.")
		spec = corev1.PersistentVolumeClaimSpec{
			VolumeName:       krp.spec.Storage.Volumes[brokerId-1],
			StorageClassName: new(string),
		}
		if len(krp.spec.Storage.ClassName) > 0 {
			spec.StorageClassName = krp.getStorageClassForDynamicallyProvidedVolumes(brokerId)
		}
	} else if len(krp.spec.Storage.Labels) > 0 {
		krp.logger.Info("Persistent volume claims are created by labels.")
		keyValue := strings.Split(krp.spec.Storage.Labels[brokerId-1], "=")
		spec = corev1.PersistentVolumeClaimSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					keyValue[0]: keyValue[1],
				},
			},
		}
		if len(krp.spec.Storage.ClassName) > 0 {
			spec.StorageClassName = krp.getStorageClassForDynamicallyProvidedVolumes(brokerId)
		}

	} else if len(krp.spec.Storage.ClassName) > 0 {
		krp.logger.Info("Persistent volume claims are created by class names.")
		spec = corev1.PersistentVolumeClaimSpec{
			StorageClassName: krp.getStorageClassForDynamicallyProvidedVolumes(brokerId),
		}
	} else {
		return nil
	}

	spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	spec.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse(krp.spec.Storage.Size),
		},
	}
	labels := krp.GetKafkaLabels()
	if krp.cr.Spec.Kraft.Enabled {
		labels["kraft"] = "enabled"
	}
	persistentVolumeClaim := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(persistentVolumeClaimPattern, krp.cr.Name, brokerId),
			Namespace: krp.cr.Namespace,
			Labels:    labels,
		},
		Spec: spec,
	}
	return persistentVolumeClaim
}

// NewKafkaControllerPersistentVolumeClaimForCR returns a persistent volume claim for migration Kafka controller
func (krp KafkaResourceProvider) NewKafkaControllerPersistentVolumeClaimForCR() *corev1.PersistentVolumeClaim {
	var spec corev1.PersistentVolumeClaimSpec
	var volumesCount = len(krp.spec.MigrationController.Storage.Volumes)
	if err := krp.checkControllerStorageClassDefinition(); err != nil {
		return nil
	}
	if volumesCount > 0 {
		krp.logger.Info("Persistent volume claims are created by volume names.")
		spec = corev1.PersistentVolumeClaimSpec{
			VolumeName:       krp.spec.MigrationController.Storage.Volumes[0],
			StorageClassName: new(string),
		}
		if len(krp.spec.MigrationController.Storage.ClassName) > 0 {
			spec.StorageClassName = &krp.spec.MigrationController.Storage.ClassName[0]
		}
	} else if len(krp.spec.MigrationController.Storage.Labels) > 0 {
		krp.logger.Info("Persistent volume claims will be created by volume names.")
		keyValue := strings.Split(krp.spec.MigrationController.Storage.Labels[0], "=")
		spec = corev1.PersistentVolumeClaimSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					keyValue[0]: keyValue[1],
				},
			},
		}
		if len(krp.spec.MigrationController.Storage.ClassName) > 0 {
			spec.StorageClassName = &krp.spec.MigrationController.Storage.ClassName[0]
		}

	} else if len(krp.spec.MigrationController.Storage.ClassName) > 0 {
		krp.logger.Info("Persistent volume claims are created by class names.")
		spec = corev1.PersistentVolumeClaimSpec{
			StorageClassName: &krp.spec.MigrationController.Storage.ClassName[0],
		}
	} else {
		return nil
	}

	spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	spec.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse(krp.spec.MigrationController.Storage.Size),
		},
	}
	labels := krp.GetKafkaLabels()
	labels["kraft"] = "enabled"

	persistentVolumeClaim := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pvc-%s-%s", krp.cr.Name, "kraft-controller"),
			Namespace: krp.cr.Namespace,
			Labels:    labels,
		},
		Spec: spec,
	}
	return persistentVolumeClaim
}

// checkStorageClassDefinition checks that number of storage classes is correct
func (krp KafkaResourceProvider) checkStorageClassDefinition(volumesCount int) error {
	var classNamesCount = len(krp.spec.Storage.ClassName)
	if classNamesCount > 1 && classNamesCount != volumesCount {
		return errors.New("number of storage class names should be matched to volumes number")
	}
	return nil
}

// checkControllerStorageClassDefinition checks that number of storage classes is correct for Controller
func (krp KafkaResourceProvider) checkControllerStorageClassDefinition() error {
	var classNamesCount = len(krp.spec.MigrationController.Storage.ClassName)
	if classNamesCount != 0 && classNamesCount != 1 {
		return errors.New("number of storage class names for controller should be 1 or 0")
	}
	return nil
}

// getStorageClassForDynamicallyProvidedVolumes returns storage class for specific Kafka broker with dynamic provisioning
func (krp KafkaResourceProvider) getStorageClassForDynamicallyProvidedVolumes(brokerId int) *string {
	var classNames = krp.spec.Storage.ClassName
	var classNamesCount = len(classNames)
	if classNamesCount == 1 {
		return &classNames[0]
	}
	return &classNames[brokerId-1]
}

// NewEmptySecret creates empty secret for Kafka
func (krp KafkaResourceProvider) NewEmptySecret(secretName string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: krp.cr.Namespace,
			Labels:    krp.GetKafkaLabels(),
		},
	}

	return secret
}

func (krp KafkaResourceProvider) NewKafkaBrokerDeploymentForCR(brokerId int, rack string, kraftEnabled bool, zkClusterID string) *appsv1.Deployment {
	deploymentName := fmt.Sprintf("%s-%d", krp.cr.Name, brokerId)
	domainName := fmt.Sprintf("%s-broker", krp.cr.Name)
	kafkaLabels := krp.GetKafkaLabels()
	kafkaLabels["name"] = deploymentName
	kafkaLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-%s", deploymentName, krp.cr.Namespace)
	selectorLabels := krp.GetSelectorLabels()
	selectorLabels["name"] = deploymentName
	kafkaCustomLabels := krp.GetKafkaCustomLabels(kafkaLabels)
	replicas := int32(1)
	var dataVolumeSource corev1.VolumeSource
	if len(krp.cr.Spec.Storage.Volumes) > 0 || (len(krp.cr.Spec.Storage.ClassName) > 0 && krp.cr.Spec.Storage.ClassName[0] != defaultVolumeName) {
		dataVolumeSource = corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: fmt.Sprintf(persistentVolumeClaimPattern, krp.cr.Name, brokerId),
			},
		}
	} else {
		dataVolumeSource = corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}
	}
	oauth := krp.cr.Spec.Oauth
	terminationGracePeriod := getTerminationGracePeriod(krp.cr.Spec)
	rollbackTimeout := getRollbackTimeout(krp.cr.Spec)

	externalHostName := ""
	externalPort := ""
	if len(krp.cr.Spec.ExternalHostNames) > 0 {
		externalHostName = krp.cr.Spec.ExternalHostNames[brokerId-1]
		externalPort = "9094"
		if len(krp.cr.Spec.ExternalPorts) > 0 {
			externalPort = strconv.Itoa(krp.cr.Spec.ExternalPorts[brokerId-1])
		}
	}

	volumes := []corev1.Volume{
		{Name: "data", VolumeSource: dataVolumeSource},
		{Name: "log", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "trusted-certs", VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-trusted-certs", krp.cr.Name)}}},
		{Name: "public-certs", VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-public-certs", krp.cr.Name)}}},
	}

	volumeMounts := []corev1.VolumeMount{
		{Name: "data", MountPath: "/var/opt/kafka/data"},
		{Name: "log", MountPath: "/opt/kafka/logs"},
		{Name: "trusted-certs", MountPath: "/opt/kafka/trustcerts"},
		{Name: "public-certs", MountPath: "/opt/kafka/public-certs"},
	}

	replicationFactor := 3
	if krp.cr.Spec.Replicas < 3 {
		replicationFactor = krp.cr.Spec.Replicas
	}
	envVars := []corev1.EnvVar{
		{Name: "READINESS_PERIOD", Value: "30"},
		{Name: "BROKER_ID", Value: strconv.Itoa(brokerId)},
		{Name: "REPLICATION_FACTOR", Value: strconv.Itoa(replicationFactor)},
		{Name: "EXTERNAL_HOST_NAME", Value: externalHostName},
		{Name: "EXTERNAL_PORT", Value: externalPort},
		{Name: "CONF_KAFKA_BROKER_RACK", Value: rack},
		{
			Name:  "INTERNAL_HOST_NAME",
			Value: fmt.Sprintf("%s.%s", deploymentName, krp.cr.Namespace),
		},
		{
			Name:  "INTER_BROKER_HOST_NAME",
			Value: fmt.Sprintf("%s.%s.%s", deploymentName, domainName, krp.cr.Namespace),
		},
		{
			Name:  "HEAP_OPTS",
			Value: fmt.Sprintf("-Xms%dm -Xmx%dm", krp.cr.Spec.HeapSize, krp.cr.Spec.HeapSize),
		},
		{Name: "DISABLE_SECURITY", Value: strconv.FormatBool(krp.isSecurityDisabled())},
		{Name: "CLOCK_SKEW", Value: strconv.Itoa(getClockSkew(oauth))},
		{Name: "JWK_SOURCE_TYPE", Value: getJwkSourceType(oauth)},
		{
			Name:  "JWKS_CONNECTION_TIMEOUT",
			Value: strconv.Itoa(getJwksConnectionTimeout(oauth)),
		},
		{Name: "TOKEN_ROLES_PATH", Value: krp.cr.Spec.TokenRolesPath},
		{Name: "JWKS_READ_TIMEOUT", Value: strconv.Itoa(getJwksReadTimeout(oauth))},
		{Name: "JWKS_SIZE_LIMIT", Value: strconv.Itoa(getJwksSizeLimit(oauth))},
		{Name: "ENABLE_AUDIT_LOGS", Value: strconv.FormatBool(krp.isAuditLogsEnabled())},
		{Name: "ENABLE_AUTHORIZATION", Value: strconv.FormatBool(krp.isEnableAuthorization())},
		{Name: "HEALTH_CHECK_TIMEOUT", Value: strconv.Itoa(int(getHealthCheckTimeout(krp.cr.Spec)))},
	}
	if zkClusterID == "" {
		zkClusterID = zooKeeperClusterID
	}
	if kraftEnabled {
		var voters []string
		for i := 1; i <= krp.spec.Replicas; i++ {
			voters = append(voters, fmt.Sprintf("%d@%s-%d.kafka-broker.%s:9096", i, krp.cr.Name, i, krp.cr.Namespace))
		}
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "KRAFT_ENABLED", Value: "true"},
			{Name: "KRAFT_CLUSTER_ID", Value: zkClusterID},
			{Name: "VOTERS", Value: strings.Join(voters, ",")},
			{Name: "PROCESS_ROLES", Value: "broker,controller"},
		}...)
	} else {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "ZOOKEEPER_CONNECT", Value: krp.cr.Spec.ZookeeperConnect},
			{Name: "ZOOKEEPER_SET_ACL", Value: strconv.FormatBool(krp.isZookeeperSetACL())},
		}...)
	}
	envVars = append(envVars, krp.getSecretEnvs(kraftEnabled)...)

	if krp.cr.Spec.Ssl.Enabled && krp.cr.Spec.Ssl.SecretName != "" {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "ENABLE_SSL", Value: "true"},
			{Name: "SSL_CIPHER_SUITES", Value: strings.Join(krp.cr.Spec.Ssl.CipherSuites, ",")},
			{Name: "ENABLE_2WAY_SSL", Value: strconv.FormatBool(krp.cr.Spec.Ssl.EnableTwoWaySsl)},
			{Name: "ALLOW_NONENCRYPTED_ACCESS", Value: strconv.FormatBool(krp.cr.Spec.Ssl.AllowNonencryptedAccess)},
		}...)

		volumes = append(volumes, corev1.Volume{
			Name: "ssl-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: krp.cr.Spec.Ssl.SecretName,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "ssl-certs", MountPath: "/opt/kafka/tls"})
	}

	if krp.cr.Spec.ZookeeperEnableSsl && !kraftEnabled {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "ENABLE_ZOOKEEPER_SSL", Value: "true"},
		}...)
		if krp.cr.Spec.ZookeeperSslSecretName != "" {
			volumes = append(volumes, corev1.Volume{
				Name: "zookeeper-ssl-certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: krp.cr.Spec.ZookeeperSslSecretName,
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "zookeeper-ssl-certs", MountPath: "/opt/kafka/zookeeper-tls"})
		}
	}

	if krp.cr.Spec.CCMetricReporterEnabled {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "METRIC_COLLECTOR_ENABLED", Value: "true"},
		}...)
	}

	brokerDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: krp.cr.Namespace,
			Labels:    kafkaLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Strategy:                appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Replicas:                &replicas,
			ProgressDeadlineSeconds: &rollbackTimeout,
			Selector:                &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: kafkaCustomLabels},
				Spec: corev1.PodSpec{
					Volumes:                       volumes,
					InitContainers:                krp.getInitContainers(),
					Containers:                    krp.createDeploymentContainers(buildEnvs(envVars, krp.spec.EnvironmentVariables, krp.logger), volumeMounts, kraftEnabled, false),
					SecurityContext:               &krp.cr.Spec.SecurityContext,
					ServiceAccountName:            krp.GetServiceAccountName(),
					TerminationGracePeriodSeconds: &terminationGracePeriod,
					Hostname:                      deploymentName,
					Subdomain:                     domainName,
					Affinity:                      krp.getBrokerAffinityForCR(brokerId),
					Tolerations:                   krp.spec.Tolerations,
					PriorityClassName:             krp.spec.PriorityClassName,
				},
			},
		},
	}
	return brokerDeployment
}

func (krp KafkaResourceProvider) NewKafkaKraftControllerDeploymentForCR(zkClusterID string, migrated bool, zookeeperEnabled bool) *appsv1.Deployment {
	deploymentName := fmt.Sprintf("%s-%s", krp.cr.Name, "kraft-controller")
	domainName := fmt.Sprintf("%s-broker", krp.cr.Name)
	kafkaLabels := krp.GetKafkaLabels()
	kafkaLabels["name"] = deploymentName
	kafkaLabels["component"] = "kafka-controller"
	delete(kafkaLabels, "clusterName")
	kafkaLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-%s", deploymentName, krp.cr.Namespace)
	selectorLabels := make(map[string]string)
	selectorLabels["name"] = deploymentName
	selectorLabels["component"] = "kafka-controller"
	kafkaCustomLabels := krp.GetKafkaCustomLabels(kafkaLabels)
	replicas := int32(1)
	claimName := fmt.Sprintf("pvc-%s-%s", krp.cr.Name, "kraft-controller")
	var dataVolumeSource corev1.VolumeSource
	if len(krp.cr.Spec.MigrationController.Storage.Volumes) > 0 || (len(krp.cr.Spec.MigrationController.Storage.ClassName) > 0 && krp.cr.Spec.MigrationController.Storage.ClassName[0] != defaultVolumeName) {
		dataVolumeSource = corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		}
	} else {
		dataVolumeSource = corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}
	}
	oauth := krp.cr.Spec.Oauth
	terminationGracePeriod := getTerminationGracePeriod(krp.cr.Spec)
	rollbackTimeout := getRollbackTimeout(krp.cr.Spec)

	volumes := []corev1.Volume{
		{Name: "data", VolumeSource: dataVolumeSource},
		{Name: "log", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "trusted-certs", VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-trusted-certs", krp.cr.Name)}}},
		{Name: "public-certs", VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-public-certs", krp.cr.Name)}}},
	}

	volumeMounts := []corev1.VolumeMount{
		{Name: "data", MountPath: "/var/opt/kafka/data"},
		{Name: "log", MountPath: "/opt/kafka/logs"},
		{Name: "trusted-certs", MountPath: "/opt/kafka/trustcerts"},
		{Name: "public-certs", MountPath: "/opt/kafka/public-certs"},
	}

	var voters []string
	voters = append(voters, "3000@localhost:9092")
	if migrated {
		for i := 1; i <= krp.spec.Replicas; i++ {
			voters = append(voters, fmt.Sprintf("%d@%s-%d.kafka-broker.%s:9096", i, krp.cr.Name, i, krp.cr.Namespace))
		}
	}

	envVars := []corev1.EnvVar{
		{Name: "KRAFT_ENABLED", Value: "true"},
		{Name: "BROKER_ID", Value: "3000"},
		{Name: "CONTROLLER_LISTENER_NAMES", Value: "CONTROLLER"},
		{Name: "LISTENERS", Value: "CONTROLLER://:9092"},
		{Name: "KRAFT_CLUSTER_ID", Value: zkClusterID},
		{Name: "INTER_BROKER_LISTENER_NAME", Value: "INTERNAL"},
		{Name: "PROCESS_ROLES", Value: "controller"},
		{Name: "VOTERS", Value: strings.Join(voters, ",")},
		{Name: "READINESS_PERIOD", Value: "30"},
		{Name: "REPLICATION_FACTOR", Value: "3"},
		{Name: "EXTERNAL_HOST_NAME", Value: ""},
		{Name: "EXTERNAL_PORT", Value: ""},
		{Name: "CONF_KAFKA_BROKER_RACK", Value: ""},
		{
			Name:  "INTERNAL_HOST_NAME",
			Value: fmt.Sprintf("%s.%s", deploymentName, krp.cr.Namespace),
		},
		{
			Name:  "INTER_BROKER_HOST_NAME",
			Value: fmt.Sprintf("%s.%s.%s", deploymentName, domainName, krp.cr.Namespace),
		},
		{
			Name:  "HEAP_OPTS",
			Value: fmt.Sprintf("-Xms%dm -Xmx%dm", krp.cr.Spec.HeapSize, krp.cr.Spec.HeapSize),
		},
		{Name: "DISABLE_SECURITY", Value: strconv.FormatBool(krp.isSecurityDisabled())},
		{Name: "CLOCK_SKEW", Value: strconv.Itoa(getClockSkew(oauth))},
		{Name: "JWK_SOURCE_TYPE", Value: getJwkSourceType(oauth)},
		{
			Name:  "JWKS_CONNECTION_TIMEOUT",
			Value: strconv.Itoa(getJwksConnectionTimeout(oauth)),
		},
		{Name: "TOKEN_ROLES_PATH", Value: krp.cr.Spec.TokenRolesPath},
		{Name: "JWKS_READ_TIMEOUT", Value: strconv.Itoa(getJwksReadTimeout(oauth))},
		{Name: "JWKS_SIZE_LIMIT", Value: strconv.Itoa(getJwksSizeLimit(oauth))},
		{Name: "ENABLE_AUDIT_LOGS", Value: strconv.FormatBool(krp.isAuditLogsEnabled())},
		{Name: "ENABLE_AUTHORIZATION", Value: strconv.FormatBool(krp.isEnableAuthorization())},
		{Name: "HEALTH_CHECK_TIMEOUT", Value: strconv.Itoa(int(getHealthCheckTimeout(krp.cr.Spec)))},
	}

	if zookeeperEnabled {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "ZOOKEEPER_CONNECT", Value: krp.cr.Spec.ZookeeperConnect},
			{Name: "ZOOKEEPER_SET_ACL", Value: strconv.FormatBool(krp.isZookeeperSetACL())},
		}...)
	}

	if migrated {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "MIGRATED_CONTROLLER", Value: "true"},
		}...)
	} else {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "MIGRATION_CONTROLLER", Value: "true"},
		}...)
	}

	envVars = append(envVars, krp.getSecretEnvs(!zookeeperEnabled)...)

	if krp.cr.Spec.Ssl.Enabled && krp.cr.Spec.Ssl.SecretName != "" {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "ENABLE_SSL", Value: "true"},
			{Name: "SSL_CIPHER_SUITES", Value: strings.Join(krp.cr.Spec.Ssl.CipherSuites, ",")},
			{Name: "ENABLE_2WAY_SSL", Value: strconv.FormatBool(krp.cr.Spec.Ssl.EnableTwoWaySsl)},
			{Name: "ALLOW_NONENCRYPTED_ACCESS", Value: strconv.FormatBool(krp.cr.Spec.Ssl.AllowNonencryptedAccess)},
		}...)

		volumes = append(volumes, corev1.Volume{
			Name: "ssl-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: krp.cr.Spec.Ssl.SecretName,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "ssl-certs", MountPath: "/opt/kafka/tls"})
	}

	if krp.cr.Spec.ZookeeperEnableSsl && zookeeperEnabled {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "ENABLE_ZOOKEEPER_SSL", Value: "true"},
		}...)
		if krp.cr.Spec.ZookeeperSslSecretName != "" {
			volumes = append(volumes, corev1.Volume{
				Name: "zookeeper-ssl-certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: krp.cr.Spec.ZookeeperSslSecretName,
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "zookeeper-ssl-certs", MountPath: "/opt/kafka/zookeeper-tls"})
		}
	}

	if krp.cr.Spec.CCMetricReporterEnabled {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "METRIC_COLLECTOR_ENABLED", Value: "true"},
		}...)
	}

	controllerDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: krp.cr.Namespace,
			Labels:    kafkaLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Strategy:                appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Replicas:                &replicas,
			ProgressDeadlineSeconds: &rollbackTimeout,
			Selector:                &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: kafkaCustomLabels},
				Spec: corev1.PodSpec{
					Volumes:                       volumes,
					InitContainers:                krp.getInitContainers(),
					Containers:                    krp.createDeploymentContainers(buildEnvs(envVars, krp.spec.EnvironmentVariables, krp.logger), volumeMounts, false, true),
					SecurityContext:               &krp.cr.Spec.SecurityContext,
					ServiceAccountName:            krp.GetServiceAccountName(),
					TerminationGracePeriodSeconds: &terminationGracePeriod,
					Hostname:                      deploymentName,
					Subdomain:                     domainName,
					Affinity:                      krp.getControllerAffinityForCR(),
					Tolerations:                   krp.spec.MigrationController.Tolerations,
					PriorityClassName:             krp.spec.MigrationController.PriorityClassName,
				},
			},
		},
	}
	return controllerDeployment
}

func (krp KafkaResourceProvider) GetZooKeeperFullName() string {
	zooKeeperConnect := krp.spec.ZookeeperConnect
	zooKeeperAddress := strings.Split(zooKeeperConnect, ":")[0]
	zooKeeperNameAndSpace := strings.Split(zooKeeperAddress, ".")
	if len(zooKeeperNameAndSpace) == 1 {
		return fmt.Sprintf("%s.%s", zooKeeperAddress, krp.cr.Namespace)
	}
	return zooKeeperAddress
}

func (krp KafkaResourceProvider) isSecurityDisabled() bool {
	if krp.cr.Spec.DisableSecurity != nil {
		return *krp.cr.Spec.DisableSecurity
	}
	return false
}

func (krp KafkaResourceProvider) isAuditLogsEnabled() bool {
	if krp.cr.Spec.EnableAuditLogs != nil {
		return *krp.cr.Spec.EnableAuditLogs
	}
	return false
}

func (krp KafkaResourceProvider) isZookeeperSetACL() bool {
	if krp.cr.Spec.ZookeeperSetACL != nil {
		return *krp.cr.Spec.ZookeeperSetACL
	}
	return false
}

func (krp KafkaResourceProvider) isEnableAuthorization() bool {
	if krp.cr.Spec.EnableAuthorization != nil {
		return *krp.cr.Spec.EnableAuthorization
	}
	return false
}

func getClockSkew(oauth kafkaservice.OAuth) int {
	if oauth.ClockSkew != nil {
		return *oauth.ClockSkew
	}
	return defaultClockSkew
}

func getJwkSourceType(oauth kafkaservice.OAuth) string {
	if oauth.JwkSourceType != nil {
		return *oauth.JwkSourceType
	}
	return defaultJwkSourceType
}

func getJwksConnectionTimeout(oauth kafkaservice.OAuth) int {
	if oauth.JwksConnectionTimeout != nil {
		return *oauth.JwksConnectionTimeout
	}
	return defaultJwksConnectionTimeout
}

func getJwksReadTimeout(oauth kafkaservice.OAuth) int {
	if oauth.JwksReadTimeout != nil {
		return *oauth.JwksReadTimeout
	}
	return defaultJwksReadTimeout
}

func getJwksSizeLimit(oauth kafkaservice.OAuth) int {
	if oauth.JwksSizeLimit != nil {
		return *oauth.JwksSizeLimit
	}
	return defaultJwksSizeLimit
}

func getTerminationGracePeriod(kafka kafkaservice.KafkaSpec) int64 {
	if kafka.TerminationGracePeriod != nil {
		return *kafka.TerminationGracePeriod
	}
	return defaultTerminationGracePeriod
}

func getRollbackTimeout(kafka kafkaservice.KafkaSpec) int32 {
	if kafka.RollbackTimeout != nil {
		return *kafka.RollbackTimeout
	}
	return defaultRollbackTimeout
}

// if cluster is scaled out, partitions reassignment also should be called if ReassignPartitions is nil
// in other case partitions reassignment should be called only on ReassignPartitions=true
func (krp KafkaResourceProvider) IsReassignPartitionsEnabled(clusterScaling bool) bool {
	if krp.cr.Spec.Scaling.ReassignPartitions != nil {
		return *krp.cr.Spec.Scaling.ReassignPartitions
	}
	return clusterScaling
}

func (krp KafkaResourceProvider) GetAllBrokersStartTimeoutSeconds() int {
	if krp.cr.Spec.Scaling.AllBrokersStartTimeoutSeconds != nil {
		return *krp.cr.Spec.Scaling.AllBrokersStartTimeoutSeconds
	}
	return defaultAllBrokersStartTimeoutSeconds
}

func (krp KafkaResourceProvider) IsBrokerScalingInEnabled() bool {
	if krp.cr.Spec.Scaling.BrokerDeploymentScaleInEnabled != nil {
		return *krp.cr.Spec.Scaling.BrokerDeploymentScaleInEnabled
	}
	return defaultBrokerDeploymentScaleInEnabled
}

func (krp KafkaResourceProvider) GetTopicReassignmentTimeoutSeconds() int {
	if krp.cr.Spec.Scaling.TopicReassignmentTimeoutSeconds != nil {
		return *krp.cr.Spec.Scaling.TopicReassignmentTimeoutSeconds
	}
	return defaultTopicReassignmentTimeoutSeconds
}

func getHealthCheckTimeout(kafka kafkaservice.KafkaSpec) int32 {
	if kafka.HealthCheckTimeout != nil {
		return *kafka.HealthCheckTimeout
	}
	return defaultHealthCheckTimeout
}

func (krp KafkaResourceProvider) getBrokerAffinityForCR(brokerId int) *corev1.Affinity {
	affinity := krp.cr.Spec.Affinity.DeepCopy()
	if len(krp.cr.Spec.Storage.Nodes) > 0 {
		affinity.NodeAffinity = &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{krp.cr.Spec.Storage.Nodes[brokerId-1]},
							},
						},
					},
				},
			},
		}
	}
	return affinity
}

func (krp KafkaResourceProvider) getControllerAffinityForCR() *corev1.Affinity {
	affinity := krp.cr.Spec.MigrationController.Affinity.DeepCopy()
	if len(krp.cr.Spec.MigrationController.Storage.Nodes) > 0 {
		affinity.NodeAffinity = &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{krp.cr.Spec.MigrationController.Storage.Nodes[0]},
							},
						},
					},
				},
			},
		}
	}
	return affinity
}

func (krp KafkaResourceProvider) getCommand() []string {
	return nil
}

func (krp KafkaResourceProvider) getArgs() []string {
	return nil
}

func (krp KafkaResourceProvider) getInitContainers() []corev1.Container {
	return nil
}

func (krp KafkaResourceProvider) getExecCommand(originalCommand []string) *corev1.ExecAction {
	return &corev1.ExecAction{Command: originalCommand}
}

func (krp KafkaResourceProvider) getSecretEnvs(kraftEnabled bool) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{Name: "ADMIN_USERNAME", ValueFrom: getSecretEnvVarSource(krp.cr.Spec.SecretName, "admin-username")},
		{Name: "ADMIN_PASSWORD", ValueFrom: getSecretEnvVarSource(krp.cr.Spec.SecretName, "admin-password")},
		{Name: "CLIENT_USERNAME", ValueFrom: getSecretEnvVarSource(krp.cr.Spec.SecretName, "client-username")},
		{Name: "CLIENT_PASSWORD", ValueFrom: getSecretEnvVarSource(krp.cr.Spec.SecretName, "client-password")},

		{Name: "IDP_WHITELIST", ValueFrom: getSecretEnvVarSource(krp.cr.Spec.SecretName, "idp-whitelist")},
	}
	if !kraftEnabled {
		envVars = append(envVars, []corev1.EnvVar{
			{Name: "ZOOKEEPER_CLIENT_USERNAME", ValueFrom: getSecretEnvVarSource(krp.cr.Spec.SecretName, "zookeeper-client-username")},
			{Name: "ZOOKEEPER_CLIENT_PASSWORD", ValueFrom: getSecretEnvVarSource(krp.cr.Spec.SecretName, "zookeeper-client-password")},
		}...)
	}
	return envVars
}

func (krp KafkaResourceProvider) createDeploymentContainers(envs []corev1.EnvVar, volumeMounts []corev1.VolumeMount, kraftEnabled bool, kraftController bool) []corev1.Container {
	ports := []corev1.ContainerPort{
		{ContainerPort: 9092, Protocol: corev1.ProtocolTCP},
		{ContainerPort: 9093, Protocol: corev1.ProtocolTCP},
		{ContainerPort: 9094, Protocol: corev1.ProtocolTCP},
		{ContainerPort: 9095, Protocol: corev1.ProtocolTCP},
		{ContainerPort: 9087, Protocol: corev1.ProtocolTCP},
		{ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
	}
	if kraftEnabled {
		ports = append(ports, []corev1.ContainerPort{
			{ContainerPort: 9096, Protocol: corev1.ProtocolTCP},
		}...)
	}
	var livenessCommand []string
	var readinessCommand []string
	if kraftController {
		livenessCommand = []string{"./bin/kafka-health.sh"}
		readinessCommand = []string{"./bin/kafka-health.sh"}
	} else {
		livenessCommand = []string{"./bin/kafka-health.sh", "live"}
		readinessCommand = []string{"./bin/kafka-health.sh", "ready"}
	}
	deploymentContainers := []corev1.Container{
		{
			Name:    "kafka",
			Image:   krp.cr.Spec.DockerImage,
			Command: krp.getCommand(),
			Args:    krp.getArgs(),
			Ports:   ports,
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: krp.getExecCommand(livenessCommand),
				},
				InitialDelaySeconds: 60,
				TimeoutSeconds:      5,
				PeriodSeconds:       15,
				SuccessThreshold:    1,
				FailureThreshold:    20,
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: krp.getExecCommand(readinessCommand),
				},
				InitialDelaySeconds: 60,
				TimeoutSeconds:      getHealthCheckTimeout(krp.cr.Spec),
				PeriodSeconds:       getHealthCheckTimeout(krp.cr.Spec),
				SuccessThreshold:    1,
				FailureThreshold:    5,
			},
			Env:             envs,
			Resources:       krp.cr.Spec.Resources,
			VolumeMounts:    volumeMounts,
			ImagePullPolicy: corev1.PullAlways,
			SecurityContext: getDefaultContainerSecurityContext(),
		},
	}
	if krp.cr.Spec.DebugContainer {

		resources := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("25Mi"),
				corev1.ResourceCPU:    resource.MustParse("10m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("100Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		}
		if value, found := krp.cr.Spec.Resources.Requests[corev1.ResourceEphemeralStorage]; found {
			resources.Requests[corev1.ResourceEphemeralStorage] = value
		}
		if value, found := krp.cr.Spec.Resources.Limits[corev1.ResourceEphemeralStorage]; found {
			resources.Limits[corev1.ResourceEphemeralStorage] = value
		}
		debugContainer := corev1.Container{
			Name:            "kafka-debug-container",
			Image:           krp.cr.Spec.DockerImage,
			Command:         []string{"/bin/sh"},
			Args:            []string{"-c", "tail -f /dev/null"},
			Env:             envs,
			Resources:       resources,
			VolumeMounts:    volumeMounts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: getDefaultContainerSecurityContext(),
		}
		deploymentContainers = append(deploymentContainers, debugContainer)
	}
	return deploymentContainers
}
