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
	kafkaservice "github.com/Netcracker/qubership-kafka/api/v7"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type MirrorMakerMonitoringResourceProvider struct {
	cr          *kafkaservice.KafkaService
	logger      logr.Logger
	spec        *kafkaservice.MirrorMakerMonitoring
	serviceName string
}

func (mmmrp MirrorMakerMonitoringResourceProvider) GetServiceName() string {
	return mmmrp.serviceName
}

func (mmmrp MirrorMakerMonitoringResourceProvider) GetServiceAccountName() string {
	return mmmrp.GetServiceName()
}

func NewMirrorMakerMonitoringResourceProvider(cr *kafkaservice.KafkaService, logger logr.Logger) MirrorMakerMonitoringResourceProvider {
	return MirrorMakerMonitoringResourceProvider{
		cr:          cr,
		spec:        cr.Spec.MirrorMakerMonitoring,
		logger:      logger,
		serviceName: fmt.Sprintf("%s-mirror-maker-monitoring", cr.Name),
	}
}

func (mmmrp MirrorMakerMonitoringResourceProvider) getConfigurationName() string {
	return fmt.Sprintf("%s-configuration", mmmrp.serviceName)
}

func (mmmrp MirrorMakerMonitoringResourceProvider) GetMirrorMakerMonitoringLabels() map[string]string {
	labels := make(map[string]string)
	labels["app.kubernetes.io/name"] = mmmrp.serviceName
	labels = util.JoinMaps(util.JoinMaps(labels, mmmrp.GetMirrorMakerMonitoringSelectorLabels()), mmmrp.cr.Spec.Global.DefaultLabels)
	return labels
}

func (mmmrp MirrorMakerMonitoringResourceProvider) GetMirrorMakerMonitoringSelectorLabels() map[string]string {
	return map[string]string{
		"component": "mirror-maker-monitoring",
		"name":      mmmrp.serviceName,
	}
}

func (mmmrp MirrorMakerMonitoringResourceProvider) GetMirrorMakerMonitoringCustomLabels(monitoringLabels map[string]string) map[string]string {
	globalLabels := mmmrp.cr.Spec.Global.CustomLabels
	customLabels := mmmrp.spec.CustomLabels
	return util.JoinMaps(util.JoinMaps(globalLabels, customLabels), monitoringLabels)
}

// NewMirrorMakerMonitoringDeployment returns the deployment for specified Kafka Mirror Maker Monitoring cluster
func (mmmrp MirrorMakerMonitoringResourceProvider) NewMirrorMakerMonitoringDeployment() *appsv1.Deployment {
	deploymentName := mmmrp.serviceName
	monitoringLabels := mmmrp.GetMirrorMakerMonitoringLabels()
	monitoringLabels["app.kubernetes.io/technology"] = "python"
	monitoringLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-%s", deploymentName, mmmrp.cr.Namespace)
	monitoringCustomLabels := mmmrp.GetMirrorMakerMonitoringCustomLabels(monitoringLabels)
	selectorLabels := mmmrp.GetMirrorMakerMonitoringSelectorLabels()
	replicas := int32(1)
	if mmmrp.cr.Spec.DisasterRecovery != nil &&
		(mmmrp.cr.Spec.DisasterRecovery.Mode == "active" || mmmrp.cr.Spec.DisasterRecovery.Mode == "disabled") {
		replicas = int32(0)
	}
	volumes := mmmrp.getMirrorMakerMonitoringVolumes()
	volumeMounts := mmmrp.getMirrorMakerMonitoringVolumeMounts()

	envVars := mmmrp.getMonitoringEnvironmentVariables()

	if mmmrp.spec.MonitoringType == "influxdb" {
		envVars = append(envVars, []corev1.EnvVar{
			{
				Name:  "SM_DB_HOST",
				Value: mmmrp.spec.SmDbHost,
			},
			{
				Name:  "SM_DB_NAME",
				Value: mmmrp.spec.SmDbName,
			},
		}...)
		envVars = append(envVars, mmmrp.getMonitoringCredentialsEnvs()...)
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: mmmrp.cr.Namespace,
			Labels:    monitoringLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: monitoringCustomLabels,
				},
				Spec: corev1.PodSpec{
					Volumes:        volumes,
					InitContainers: mmmrp.getInitContainers(),
					Containers: []corev1.Container{
						{
							Name:            deploymentName,
							Image:           mmmrp.spec.DockerImage,
							Ports:           mmmrp.getContainerPorts(),
							Env:             envVars,
							Resources:       mmmrp.spec.Resources,
							VolumeMounts:    volumeMounts,
							ImagePullPolicy: corev1.PullAlways,
							Command:         mmmrp.getCommand(),
							Args:            mmmrp.getArgs(),
							SecurityContext: getDefaultContainerSecurityContext(),
						},
					},
					SecurityContext:    &mmmrp.spec.SecurityContext,
					ServiceAccountName: mmmrp.GetServiceAccountName(),
					Hostname:           deploymentName,
					Affinity:           &mmmrp.spec.Affinity,
					Tolerations:        mmmrp.spec.Tolerations,
					PriorityClassName:  mmmrp.spec.PriorityClassName,
				},
			},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		},
	}
}

func (mmmrp MirrorMakerMonitoringResourceProvider) getMonitoringEnvironmentVariables() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "OS_PROJECT",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "PROMETHEUS_URLS",
			Value: mmmrp.getPrometheusUrls(),
		},
		{
			Name:  "KMM_EXEC_PLUGIN_TIMEOUT",
			Value: util.DefaultIfEmpty(mmmrp.spec.KmmExecPluginTimeout, "20s"),
		},
		{
			Name:  "KMM_COLLECTION_INTERVAL",
			Value: util.DefaultIfEmpty(mmmrp.spec.KmmCollectionInterval, "5s"),
		},
	}
}

func (mmmrp MirrorMakerMonitoringResourceProvider) NewMonitoringClientService() *corev1.Service {
	monitoringLabels := mmmrp.GetMirrorMakerMonitoringLabels()
	selectorLabels := mmmrp.GetMirrorMakerMonitoringSelectorLabels()
	ports := []corev1.ServicePort{
		{
			Name:     "kmm-monitoring-statsd",
			Protocol: corev1.ProtocolTCP,
			Port:     8125,
		},
		{
			Name:     "kmm-monitoring-tcp",
			Protocol: corev1.ProtocolTCP,
			Port:     8094,
		},
		{
			Name:     "kmm-monitoring-udp",
			Protocol: corev1.ProtocolUDP,
			Port:     8092,
		},
	}
	if mmmrp.spec.MonitoringType == "prometheus" {
		ports = append(ports, corev1.ServicePort{
			Name:     "prometheus-cli",
			Port:     8096,
			Protocol: corev1.ProtocolTCP,
		})
	}
	clientService := newServiceForCR(mmmrp.serviceName, mmmrp.cr.Namespace, monitoringLabels, selectorLabels, ports)
	return clientService
}

func (mmmrp MirrorMakerMonitoringResourceProvider) getContainerPorts() []corev1.ContainerPort {
	ports := []corev1.ContainerPort{
		{
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8125,
		},
		{
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8094,
		},
		{
			Protocol:      corev1.ProtocolUDP,
			ContainerPort: 8092,
		},
	}
	if mmmrp.spec.MonitoringType == "prometheus" {
		ports = append(ports, corev1.ContainerPort{
			Name:          "prometheus-cli",
			ContainerPort: 8096,
			Protocol:      corev1.ProtocolTCP,
		})
	}
	return ports
}

func (mmmrp MirrorMakerMonitoringResourceProvider) getPrometheusUrls() string {
	var urlList = buildPrometheusUrlsByMirrorMakerConfig(mmmrp)
	for i, prometheusUrl := range urlList {
		urlList[i] = fmt.Sprintf("'%s'", prometheusUrl)
	}
	result := strings.Join(urlList, ",")
	result = fmt.Sprintf("[%s]", result)
	return result

}

func buildPrometheusUrlsByMirrorMakerConfig(mmmrp MirrorMakerMonitoringResourceProvider) []string {
	var urlList []string
	ks := mmmrp.cr
	if ks.Spec.MirrorMaker.RegionName == "" {
		for _, cluster := range ks.Spec.MirrorMaker.Clusters {
			deploymentName := fmt.Sprintf("%s-%s", strings.ToLower(cluster.Name), BuildServiceName(ks))
			urlList = append(urlList, fmt.Sprintf("http://%s.%s:8080", deploymentName, ks.Namespace))
		}
	} else {
		for _, cluster := range ks.Spec.MirrorMaker.Clusters {
			currentClusterName := strings.ToLower(cluster.Name)
			if currentClusterName == ks.Spec.MirrorMaker.RegionName {
				deploymentName := fmt.Sprintf("%s-%s", currentClusterName, BuildServiceName(ks))
				urlList = append(urlList, fmt.Sprintf("http://%s.%s:8080", deploymentName, ks.Namespace))
				break
			}
		}
	}
	return urlList
}

// getMirrorMakerMonitoringVolumes configures the list of Kafka Mirror Maker volumes
func (mmmrp MirrorMakerMonitoringResourceProvider) getMirrorMakerMonitoringVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: mmmrp.getConfigurationName(),
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "config",
							Path: "telegraf.conf",
						},
					},
				},
			},
		},
	}
}

// getMirrorMakerMonitoringVolumeMounts configures the list of Kafka Mirror Maker Monitoring volume mounts
func (mmmrp MirrorMakerMonitoringResourceProvider) getMirrorMakerMonitoringVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/etc/telegraf",
		},
	}
	return volumeMounts
}

func (mmmrp MirrorMakerMonitoringResourceProvider) getCommand() []string {
	return nil
}

func (mmmrp MirrorMakerMonitoringResourceProvider) getArgs() []string {
	return nil
}

func (mmmrp MirrorMakerMonitoringResourceProvider) getInitContainers() []corev1.Container {
	return nil
}

func (mmmrp MirrorMakerMonitoringResourceProvider) getMonitoringCredentialsEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:      "SM_DB_USERNAME",
			ValueFrom: getSecretEnvVarSource(mmmrp.spec.SecretName, "sm-db-username"),
		},
		{
			Name:      "SM_DB_PASSWORD",
			ValueFrom: getSecretEnvVarSource(mmmrp.spec.SecretName, "sm-db-password"),
		},
	}
}
