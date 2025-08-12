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
	"github.com/Netcracker/qubership-kafka/operator/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

type MonitoringResourceProvider struct {
	cr                          *kafkaservice.KafkaService
	logger                      logr.Logger
	spec                        *kafkaservice.Monitoring
	serviceName                 string
	lagExporterResourceProvider LagExporterResourceProvider
}

func NewMonitoringResourceProvider(
	cr *kafkaservice.KafkaService,
	logger logr.Logger) MonitoringResourceProvider {
	return MonitoringResourceProvider{
		cr:                          cr,
		spec:                        cr.Spec.Monitoring,
		logger:                      logger,
		serviceName:                 fmt.Sprintf("%s-monitoring", cr.Name),
		lagExporterResourceProvider: NewLagExporterResourceProvider(cr, logger),
	}
}

func (mrp MonitoringResourceProvider) GetMonitoringLabels() map[string]string {
	labels := make(map[string]string)
	labels["app.kubernetes.io/name"] = mrp.serviceName
	labels = util.JoinMaps(util.JoinMaps(labels, mrp.GetMonitoringSelectorLabels()), mrp.cr.Spec.Global.DefaultLabels)
	return labels
}

func (mrp MonitoringResourceProvider) GetMonitoringSelectorLabels() map[string]string {
	return map[string]string{
		"component": "kafka-monitoring",
		"name":      mrp.serviceName,
	}
}

func (mrp MonitoringResourceProvider) GetMonitoringCustomLabels(monitoringLabels map[string]string) map[string]string {
	globalLabels := mrp.cr.Spec.Global.CustomLabels
	customLabels := mrp.spec.CustomLabels
	return util.JoinMaps(util.JoinMaps(globalLabels, customLabels), monitoringLabels)
}

func (mrp MonitoringResourceProvider) GetServiceName() string {
	return mrp.serviceName
}

func (mrp MonitoringResourceProvider) GetServiceAccountName() string {
	return mrp.GetServiceName()
}

func (mrp MonitoringResourceProvider) NewMonitoringClientService() *corev1.Service {
	monitoringLabels := mrp.GetMonitoringLabels()
	selectorLabels := mrp.GetMonitoringSelectorLabels()
	ports := []corev1.ServicePort{
		{
			Name:     "kafka-monitoring-statsd",
			Protocol: corev1.ProtocolTCP,
			Port:     8125,
		},
		{
			Name:     "kafka-monitoring-tcp",
			Protocol: corev1.ProtocolTCP,
			Port:     8094,
		},
		{
			Name:     "kafka-monitoring-udp",
			Protocol: corev1.ProtocolUDP,
			Port:     8092,
		},
	}
	if mrp.spec.MonitoringType == "prometheus" {
		// Name is reduced because it must be no more than 15 characters
		ports = append(ports, corev1.ServicePort{
			Name:     "prometheus-cli",
			Port:     8096,
			Protocol: corev1.ProtocolTCP,
		})
	}
	if mrp.spec.LagExporter != nil {
		ports = append(ports, corev1.ServicePort{
			Name:     "http",
			Port:     int32(mrp.spec.LagExporter.MetricsPort),
			Protocol: corev1.ProtocolTCP,
		})
	}
	clientService := newServiceForCR(mrp.serviceName, mrp.cr.Namespace, monitoringLabels, selectorLabels, ports)
	return clientService
}

func (mrp MonitoringResourceProvider) NewMonitoringDeployment(cmVersion string) *appsv1.Deployment {
	monitoringLabels := mrp.GetMonitoringLabels()
	monitoringLabels["app.kubernetes.io/technology"] = "python"
	monitoringLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-%s", mrp.serviceName, mrp.cr.Namespace)
	monitoringCustomLabels := mrp.GetMonitoringCustomLabels(monitoringLabels)
	selectorLabels := mrp.GetMonitoringSelectorLabels()
	replicas := int32(1)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mrp.serviceName,
			Namespace: mrp.cr.Namespace,
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
					Volumes:            mrp.getMonitoringVolumes(),
					InitContainers:     mrp.getInitContainers(),
					Containers:         mrp.getMonitoringContainers(cmVersion),
					SecurityContext:    &mrp.spec.SecurityContext,
					ServiceAccountName: mrp.GetServiceAccountName(),
					Hostname:           mrp.serviceName,
					Affinity:           &mrp.spec.Affinity,
					Tolerations:        mrp.spec.Tolerations,
					PriorityClassName:  mrp.spec.PriorityClassName,
				},
			},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		},
	}
}

func (mrp MonitoringResourceProvider) getMonitoringContainers(cmVersion string) []corev1.Container {
	containers := []corev1.Container{
		{
			Name:            mrp.serviceName,
			Image:           mrp.spec.DockerImage,
			Ports:           mrp.getMonitoringContainerPorts(),
			Env:             mrp.getMonitoringEnvs(),
			Resources:       mrp.spec.Resources,
			VolumeMounts:    mrp.getMonitoringVolumeMounts(),
			ImagePullPolicy: corev1.PullAlways,
			Command:         mrp.getCommand(),
			Args:            mrp.getArgs(),
			SecurityContext: getDefaultContainerSecurityContext(),
		},
	}
	if mrp.spec.LagExporter != nil {
		containers = append(containers, mrp.lagExporterResourceProvider.getContainer(cmVersion))
	}
	return containers
}

func (mrp MonitoringResourceProvider) getMonitoringContainerPorts() []corev1.ContainerPort {
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
	ports = append(ports, corev1.ContainerPort{
		Name:          "prometheus-cli",
		ContainerPort: 8096,
		Protocol:      corev1.ProtocolTCP,
	})

	return ports
}

func (mrp MonitoringResourceProvider) getMonitoringEnvs() []corev1.EnvVar {
	envVars := mrp.getMonitoringEnvironmentVariables()
	envVars = append(envVars, mrp.getKafkaCredentialsEnvs()...)

	if mrp.spec.MonitoringType == "influxdb" {
		envVars = append(envVars, []corev1.EnvVar{
			{
				Name:  "SM_DB_HOST",
				Value: mrp.spec.SmDbHost,
			},
			{
				Name:  "SM_DB_NAME",
				Value: mrp.spec.SmDbName,
			},
		}...)
		envVars = append(envVars, mrp.getMonitoringCredentialsEnvs()...)
	} else {
		envVars = append(envVars, []corev1.EnvVar{
			{
				Name:      "PROMETHEUS_USERNAME",
				ValueFrom: getSecretEnvVarSource(mrp.spec.SecretName, "prometheus-username"),
			},
			{
				Name:      "PROMETHEUS_PASSWORD",
				ValueFrom: getSecretEnvVarSource(mrp.spec.SecretName, "prometheus-password"),
			},
		}...)
	}
	return envVars
}

// getMonitoringVolumes configures the list of Kafka Monitoring volumes
func (mrp MonitoringResourceProvider) getMonitoringVolumes() []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-configuration", mrp.serviceName),
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
	if mrp.cr.Spec.Global.KafkaSsl.Enabled && mrp.cr.Spec.Global.KafkaSsl.SecretName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "ssl-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: mrp.cr.Spec.Global.KafkaSsl.SecretName,
				},
			},
		})
	}
	if mrp.spec.LagExporter != nil {
		volumes = append(volumes, mrp.lagExporterResourceProvider.getVolume())
	}
	return volumes
}

// getMonitoringVolumeMounts configures the list of Kafka Monitoring volume mounts
func (mrp MonitoringResourceProvider) getMonitoringVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/etc/telegraf",
		},
	}
	if mrp.cr.Spec.Global.KafkaSsl.Enabled && mrp.cr.Spec.Global.KafkaSsl.SecretName != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "ssl-certs", MountPath: "/tls"})
	}
	return volumeMounts
}

// getMonitoringEnvironmentVariables configures the list of Kafka Monitoring environment variables
func (mrp MonitoringResourceProvider) getMonitoringEnvironmentVariables() []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name: "OS_PROJECT",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "KAFKA_SERVICE_NAME",
			Value: mrp.cr.Name,
		},
		{
			Name:  "KAFKA_EXEC_PLUGIN_TIMEOUT",
			Value: util.DefaultIfEmpty(mrp.spec.KafkaExecPluginTimeout, "10s"),
		},
		{
			Name:  "KAFKA_TOTAL_BROKERS_COUNT",
			Value: strconv.Itoa(mrp.spec.KafkaTotalBrokerCount),
		},
		{
			Name:  "KAFKA_ADDRESSES",
			Value: mrp.spec.BootstrapServers,
		},
		{
			Name:  "KAFKA_SASL_MECHANISM",
			Value: mrp.cr.Spec.Global.KafkaSaslMechanism,
		},
		{
			Name:  "KAFKA_ENABLE_SSL",
			Value: strconv.FormatBool(mrp.cr.Spec.Global.KafkaSsl.Enabled),
		},
		{
			Name:  "DATA_COLLECTION_INTERVAL",
			Value: util.DefaultIfEmpty(mrp.spec.DataCollectionInterval, "10s"),
		},
		{
			Name:  "MIN_VERSION",
			Value: mrp.spec.MinVersion,
		},
		{
			Name:  "MAX_VERSION",
			Value: mrp.spec.MaxVersion,
		},
	}
	if mrp.cr.Spec.Global.Kraft.Enabled {
		envs = append(envs, []corev1.EnvVar{
			{Name: "KRAFT_ENABLED", Value: "true"},
		}...)
	}
	return envs
}

func (mrp MonitoringResourceProvider) getCommand() []string {
	return nil
}

func (mrp MonitoringResourceProvider) getArgs() []string {
	return nil
}

func (mrp MonitoringResourceProvider) getInitContainers() []corev1.Container {

	return nil
}

func (mrp MonitoringResourceProvider) getKafkaCredentialsEnvs() []corev1.EnvVar {
	kafkaSecretName := fmt.Sprintf("%s-services-secret", mrp.cr.Name)
	return []corev1.EnvVar{
		{
			Name:      "KAFKA_USER",
			ValueFrom: getSecretEnvVarSource(kafkaSecretName, "client-username"),
		},
		{
			Name:      "KAFKA_PASSWORD",
			ValueFrom: getSecretEnvVarSource(kafkaSecretName, "client-password"),
		},
	}
}

func (mrp MonitoringResourceProvider) getMonitoringCredentialsEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:      "SM_DB_USERNAME",
			ValueFrom: getSecretEnvVarSource(mrp.spec.SecretName, "sm-db-username"),
		},
		{
			Name:      "SM_DB_PASSWORD",
			ValueFrom: getSecretEnvVarSource(mrp.spec.SecretName, "sm-db-password"),
		},
	}
}
