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
	"fmt"
	kafkaservice "github.com/Netcracker/qubership-kafka/operator/api/v7"
	"github.com/Netcracker/qubership-kafka/operator/controllers/kmmconfig"
	"github.com/Netcracker/qubership-kafka/operator/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

const TopicBlackList = "topics.blacklist"

type MirrorMakerResourceProvider struct {
	cr          *kafkaservice.KafkaService
	logger      logr.Logger
	spec        *kafkaservice.MirrorMaker
	serviceName string
}

func NewMirrorMakerResourceProvider(cr *kafkaservice.KafkaService, logger logr.Logger) MirrorMakerResourceProvider {
	return MirrorMakerResourceProvider{
		cr:          cr,
		spec:        cr.Spec.MirrorMaker,
		logger:      logger,
		serviceName: BuildServiceName(cr),
	}
}

func BuildServiceName(cr *kafkaservice.KafkaService) string {
	return fmt.Sprintf("%s-mirror-maker", cr.Name)
}

// NewConfigurationMapForCR returns the configuration map for Kafka Mirror Maker
func (mmrp MirrorMakerResourceProvider) NewConfigurationMapForCR() *corev1.ConfigMap {
	configurationMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mmrp.spec.ConfigurationName,
			Namespace: mmrp.cr.Namespace,
			Labels:    mmrp.GetMirrorMakerLabels(),
		},
		Data: map[string]string{
			"config": mmrp.GetMirrorMakerProperties(),
		},
	}
	return configurationMap
}

// GetMirrorMakerProperties returns the configuration properties for Kafka Mirror Maker as
// # mm2.properties
// clusters = first, second
// replication.factor = 3
// config.storage.replication.factor = 3
// offset.storage.replication.factor = 3
// status.storage.replication.factor = 3
// heartbeats.topic.replication.factor = 3
// checkpoints.topic.replication.factor = 3
// offset-syncs.topic.replication.factor = 3
// refresh.topics.interval.seconds = 5
// refresh.groups.interval.seconds = 5
// sync.topic.acls.enabled = false
//
// # configure [first] cluster
// first.bootstrap.servers = left-kafka:9092
//
// # configure [second] cluster
// second.bootstrap.servers = right-kafka:9092
//
// # configure a specific source->target replication flow
// first->second.enabled = true
// second->first.enabled = true
func (mmrp MirrorMakerResourceProvider) GetMirrorMakerProperties() string {
	clustersConfiguration := []string{"# mm2.properties"}

	var clusterNames []string
	for _, cluster := range mmrp.spec.Clusters {
		clusterName := strings.ToLower(cluster.Name)
		clusterNames = append(clusterNames, clusterName)
	}
	clustersConfiguration = append(clustersConfiguration,
		fmt.Sprintf("clusters = %s", strings.Join(clusterNames, ", ")),
		fmt.Sprintf("replication.factor = %d", mmrp.spec.ReplicationFactor),
		fmt.Sprintf("config.storage.replication.factor = %d", mmrp.spec.ReplicationFactor),
		fmt.Sprintf("offset.storage.replication.factor = %d", mmrp.spec.ReplicationFactor),
		fmt.Sprintf("status.storage.replication.factor = %d", mmrp.spec.ReplicationFactor),
		fmt.Sprintf("heartbeats.topic.replication.factor = %d", mmrp.spec.ReplicationFactor),
		fmt.Sprintf("checkpoints.topic.replication.factor = %d", mmrp.spec.ReplicationFactor),
		fmt.Sprintf("offset-syncs.topic.replication.factor = %d", mmrp.spec.ReplicationFactor),
		fmt.Sprintf("refresh.topics.interval.seconds = %d", getIntValueOrDefault(mmrp.spec.RefreshTopicsIntervalSeconds, 5)),
		fmt.Sprintf("refresh.groups.interval.seconds = %d", getIntValueOrDefault(mmrp.spec.RefreshGroupsIntervalSeconds, 5)),
		fmt.Sprintf("dedicated.mode.enable.internal.rest = %t", getBoolValueOrDefault(mmrp.spec.InternalRestEnabled, true)),
		fmt.Sprintf("tasks.max = %d", getIntValueOrDefault(mmrp.spec.TasksMax, int32(mmrp.spec.Replicas))),
		"sync.topic.acls.enabled = false", "")

	for _, cluster := range mmrp.spec.Clusters {
		clusterName := strings.ToLower(cluster.Name)
		clustersConfiguration = append(clustersConfiguration,
			fmt.Sprintf("# configure [%s] cluster", clusterName),
			fmt.Sprintf("%s.bootstrap.servers = %s", clusterName, cluster.BootstrapServers),
			"")
	}

	var targetClusterNames []string
	if mmrp.spec.RegionName != "" {
		mmrp.logger.Info(fmt.Sprintf("Create a cross-datacenter replication config for region: %s", mmrp.spec.RegionName))
		targetClusterNames = []string{strings.ToLower(mmrp.spec.RegionName)}
		clustersConfiguration = append(clustersConfiguration, fmt.Sprintf("target.dc=%s", mmrp.spec.RegionName))
	} else {
		targetClusterNames = clusterNames
	}
	replicationConfig := mmrp.createReplicationFlowConfig(clusterNames, targetClusterNames, mmrp.isRepeatedReplication())
	clustersConfiguration = append(clustersConfiguration, replicationConfig...)
	return strings.Join(clustersConfiguration, "\n")
}

func (mmrp MirrorMakerResourceProvider) createReplicationFlowConfig(sourceNames []string,
	targetNames []string, repeatedReplication bool) []string {
	var replicationConfig []string
	var topicPrefixes []string

	drEnabled := mmrp.cr.Spec.DisasterRecovery != nil &&
		mmrp.cr.Spec.DisasterRecovery.MirrorMakerReplication.Enabled
	if drEnabled {
		replicationConfig = append(replicationConfig,
			"emit.checkpoints.enabled = true",
			"emit.checkpoints.interval.seconds = 5",
			"sync.group.offsets.enabled = true",
			"sync.group.offsets.interval.seconds = 5")
	}

	identityReplicationEnabled := drEnabled || !mmrp.spec.ReplicationPrefixEnabled
	if identityReplicationEnabled {
		replicationConfig = append(replicationConfig,
			fmt.Sprintf("%s = %s", kmmconfig.ReplicationPolicyClassConfig, kmmconfig.IdentityReplicationPolicy))
	}

	if !mmrp.spec.ConfiguratorEnabled && mmrp.spec.TopicsToReplicate != "" {
		replicationConfig = append(replicationConfig,
			fmt.Sprintf("topics = %s", mmrp.spec.TopicsToReplicate))
	}

	replicationConfig = append(replicationConfig, "", "# configure a specific source->target replication flow")
	for _, sourceClusterName := range sourceNames {
		for _, targetClusterName := range targetNames {
			if sourceClusterName != targetClusterName {
				replicationFlow := kmmconfig.NewReplicationFlow(sourceClusterName, targetClusterName)
				replicationFlowEnabled := drEnabled || mmrp.spec.ReplicationFlowEnabled
				replicationConfig = append(replicationConfig,
					fmt.Sprintf("%s.enabled = %t", replicationFlow, replicationFlowEnabled))
				if replicationFlowEnabled {
					if !mmrp.spec.ConfiguratorEnabled {
						mmrp.logger.Info(fmt.Sprintf("Configuring transformation for %s replication flow", replicationFlow))
						transformationConfigurator := kmmconfig.NewTransformationConfigurator(mmrp.spec.Transformation, "")
						replicationConfig = transformationConfigurator.AddTransformationProperties(
							replicationConfig, sourceClusterName, targetClusterName, identityReplicationEnabled)
					} else {
						mmrp.logger.Info(fmt.Sprintf("Transformation for %s replication flow will not be configured "+
							"because KMM configurator is enabled", replicationFlow))
					}
				}
				if !drEnabled {
					replicationConfig = append(replicationConfig,
						fmt.Sprintf("%s.sync.group.offsets.enabled = false", replicationFlow),
						fmt.Sprintf("%s.sync.group.offsets.interval.seconds = 5", replicationFlow),
					)
				}
			}
		}
		if !repeatedReplication {
			topicPrefixes = append(topicPrefixes, fmt.Sprintf("%s\\.*", sourceClusterName))
		}
	}

	if !repeatedReplication {
		blackList := fmt.Sprintf("%s = %s", TopicBlackList, strings.Join(topicPrefixes, ","))
		replicationConfig = append(replicationConfig, blackList)
	}
	return replicationConfig
}

func (mmrp MirrorMakerResourceProvider) GetServiceName() string {
	return mmrp.serviceName
}

func (mmrp MirrorMakerResourceProvider) GetServiceAccountName() string {
	return mmrp.GetServiceName()
}

// NewMirrorMakerDeploymentForCR returns the deployment for specified Kafka Mirror Maker cluster
func (mmrp MirrorMakerResourceProvider) NewMirrorMakerDeploymentForCR(cluster kafkaservice.Cluster,
	clusters []kafkaservice.Cluster, secretVersion string, configMapVersion string,
	currentClusterName string, deploymentName string) *appsv1.Deployment {

	mirrorMakerLabels := mmrp.GetMirrorMakerLabels()
	mirrorMakerLabels["name"] = deploymentName
	mirrorMakerLabels["app.kubernetes.io/technology"] = "java-others"
	mirrorMakerLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-%s", deploymentName, mmrp.cr.Namespace)
	mirrorMakerCustomLabels := mmrp.GetMirrorMakerCustomLabels(mirrorMakerLabels)
	selectorLabels := mmrp.GetMirrorMakerSelectorLabels()
	selectorLabels["name"] = deploymentName
	replicas := int32(mmrp.spec.Replicas)
	if mmrp.cr.Spec.DisasterRecovery != nil &&
		(mmrp.cr.Spec.DisasterRecovery.Mode == "active" || mmrp.cr.Spec.DisasterRecovery.Mode == "disable") {
		replicas = int32(0)
	}
	prometheusPort := int32(8080)
	volumes := mmrp.getMirrorMakerVolumes()
	volumeMounts := mmrp.getMirrorMakerVolumeMounts()

	envVars := []corev1.EnvVar{
		{
			Name:  "SECRET_VERSION",
			Value: secretVersion,
		},
		{
			Name:  "CONFIG_MAP_VERSION",
			Value: configMapVersion,
		},
		{
			Name:  "HEAP_OPTS",
			Value: fmt.Sprintf("-Xms%dm -Xmx%dm", mmrp.spec.HeapSize, mmrp.spec.HeapSize),
		},
		{
			Name:  "PROMETHEUS_PORT",
			Value: strconv.Itoa(int(prometheusPort)),
		},
		{
			Name:  "CLUSTER",
			Value: currentClusterName,
		},
	}

	var clusterNames []string
	for _, cluster := range clusters {
		lowerClusterName := strings.ToLower(cluster.Name)
		upperClusterName := strings.ToUpper(cluster.Name)
		clusterNames = append(clusterNames, lowerClusterName)
		clusterNameVariables := []corev1.EnvVar{
			{
				Name:  fmt.Sprintf("%s_BOOTSTRAP_SERVERS", upperClusterName),
				Value: cluster.BootstrapServers,
			},
			{
				Name:  fmt.Sprintf("%s_ENABLE_SSL", upperClusterName),
				Value: strconv.FormatBool(cluster.EnableSsl),
			},
			{
				Name:  fmt.Sprintf("%s_SASL_MECHANISM", upperClusterName),
				Value: cluster.SaslMechanism,
			},
		}
		envVars = append(envVars, clusterNameVariables...)
		envVars = append(envVars, mmrp.getKafkaClusterCredentialsEnvs(upperClusterName, lowerClusterName)...)

		if cluster.EnableSsl && cluster.SslSecretName != "" {
			volumes = append(volumes, corev1.Volume{
				Name: fmt.Sprintf("%s-ssl-certs", cluster.Name),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: cluster.SslSecretName,
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: fmt.Sprintf("%s-ssl-certs", cluster.Name), MountPath: fmt.Sprintf("/opt/kafka/tls/%s", cluster.Name)})
		}
	}

	envVars = append(envVars, corev1.EnvVar{
		Name:  "CLUSTERS",
		Value: strings.Join(clusterNames, ", "),
	})

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: mmrp.cr.Namespace,
			Labels:    mirrorMakerLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: mirrorMakerCustomLabels},
				Spec: corev1.PodSpec{
					Volumes:        volumes,
					InitContainers: mmrp.getInitContainers(),
					Containers: []corev1.Container{
						{
							Name:  "kafka-mirror-maker",
							Image: mmrp.spec.DockerImage,
							Ports: []corev1.ContainerPort{
								{ContainerPort: prometheusPort, Protocol: corev1.ProtocolTCP},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{"./bin/health.sh", "live"}},
								},
								InitialDelaySeconds: 50,
								TimeoutSeconds:      5,
								PeriodSeconds:       15,
								SuccessThreshold:    1,
								FailureThreshold:    5,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{"./bin/health.sh", "ready"}},
								},
								InitialDelaySeconds: 60,
								TimeoutSeconds:      15,
								PeriodSeconds:       15,
								SuccessThreshold:    1,
								FailureThreshold:    5,
							},
							Env:             buildEnvs(envVars, mmrp.spec.EnvironmentVariables, mmrp.logger),
							Resources:       mmrp.spec.Resources,
							VolumeMounts:    volumeMounts,
							ImagePullPolicy: corev1.PullAlways,
							Command:         mmrp.getCommand(),
							Args:            mmrp.getArgs(),
							SecurityContext: getDefaultContainerSecurityContext(),
						},
					},
					Hostname:           deploymentName,
					Affinity:           mmrp.getAffinityRules(cluster),
					Tolerations:        mmrp.spec.Tolerations,
					PriorityClassName:  mmrp.spec.PriorityClassName,
					SecurityContext:    &mmrp.spec.SecurityContext,
					ServiceAccountName: mmrp.GetServiceAccountName(),
				},
			},
		},
	}

	return deployment
}

// getMirrorMakerVolumes configures the list of Kafka Mirror Maker volumes
func (mmrp MirrorMakerResourceProvider) getMirrorMakerVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "log",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "data",
		},
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: mmrp.spec.ConfigurationName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "config",
							Path: "kmm.conf",
						},
					},
				},
			},
		},
	}
}

// getMirrorMakerVolumeMounts configures the list of Kafka Mirror Maker volume mounts
func (mmrp MirrorMakerResourceProvider) getMirrorMakerVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "data",
			MountPath: "/var/opt/kafka/data",
		},
		{
			Name:      "log",
			MountPath: "/opt/kafka/logs",
		},
		{
			Name:      "config",
			MountPath: "/opt/kafka/config/kmm",
		},
	}
}

// getAffinityRules configures the Kafka Mirror Maker affinity rules
func (mmrp MirrorMakerResourceProvider) getAffinityRules(cluster kafkaservice.Cluster) *corev1.Affinity {
	affinity := mmrp.spec.Affinity.DeepCopy()
	if cluster.NodeLabel != "" {
		labelPair := strings.Split(cluster.NodeLabel, "=")
		affinity.NodeAffinity = &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      labelPair[0],
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{labelPair[1]},
							},
						},
					},
				},
			},
		}
	}
	return affinity
}

// getIntValueOrDefault returns int32 if value is not nil, default value otherwise
func getIntValueOrDefault(intValue *int32, defaultValue int32) int32 {
	if intValue == nil {
		return defaultValue
	}
	return *intValue
}

// getBoolValueOrDefault returns bool if value is not nil, default value otherwise
func getBoolValueOrDefault(boolValue *bool, defaultValue bool) bool {
	if boolValue == nil {
		return defaultValue
	}
	return *boolValue
}

// GetMirrorMakerLabels configures common labels for Kafka Mirror Maker resources
func (mmrp MirrorMakerResourceProvider) GetMirrorMakerLabels() map[string]string {
	labels := make(map[string]string)
	labels["app.kubernetes.io/name"] = mmrp.serviceName
	labels["name"] = mmrp.serviceName
	labels = util.JoinMaps(util.JoinMaps(labels, mmrp.GetMirrorMakerSelectorLabels()), mmrp.cr.Spec.Global.DefaultLabels)
	return labels
}

func (mmrp MirrorMakerResourceProvider) GetMirrorMakerSelectorLabels() map[string]string {
	return map[string]string{
		"component":   "kafka-mm",
		"clusterName": mmrp.serviceName,
	}
}

func (mmrp MirrorMakerResourceProvider) GetMirrorMakerCustomLabels(mirrorMakerLabels map[string]string) map[string]string {
	globalLabels := mmrp.cr.Spec.Global.CustomLabels
	customLabels := mmrp.spec.CustomLabels
	return util.JoinMaps(util.JoinMaps(globalLabels, customLabels), mirrorMakerLabels)
}

// GetService configures service for kafka mirror maker
func (mmrp MirrorMakerResourceProvider) GetService(deploymentName string) *corev1.Service {
	mirrorMakerLabels := mmrp.GetMirrorMakerLabels()
	mirrorMakerLabels["name"] = deploymentName
	selectorLabels := mmrp.GetMirrorMakerSelectorLabels()
	selectorLabels["name"] = deploymentName
	ports := []corev1.ServicePort{
		{
			Name:     "prometheus-http",
			Port:     int32(8080),
			Protocol: corev1.ProtocolTCP,
		},
	}
	return newServiceForCR(deploymentName, mmrp.cr.Namespace, mirrorMakerLabels, selectorLabels, ports)
}

func (mmrp MirrorMakerResourceProvider) isRepeatedReplication() bool {
	if mmrp.spec.RepeatedReplication != nil {
		return *mmrp.spec.RepeatedReplication
	}
	return false
}

func (mmrp MirrorMakerResourceProvider) getCommand() []string {
	return nil
}

func (mmrp MirrorMakerResourceProvider) getArgs() []string {
	return nil
}

func (mmrp MirrorMakerResourceProvider) getInitContainers() []corev1.Container {
	return nil
}

func (mmrp MirrorMakerResourceProvider) getKafkaClusterCredentialsEnvs(upperClusterName string, lowerClusterName string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:      fmt.Sprintf("%s_KAFKA_USERNAME", upperClusterName),
			ValueFrom: getSecretEnvVarSource(mmrp.spec.SecretName, fmt.Sprintf("%s-kafka-username", lowerClusterName)),
		},
		{
			Name:      fmt.Sprintf("%s_KAFKA_PASSWORD", upperClusterName),
			ValueFrom: getSecretEnvVarSource(mmrp.spec.SecretName, fmt.Sprintf("%s-kafka-password", lowerClusterName)),
		},
	}
}
