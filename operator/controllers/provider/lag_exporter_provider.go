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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"strconv"
)

const (
	volumeName               = "config-volume"
	lagExporterNameSuffix    = "lag-exporter"
	lagExporterContainerName = "kafka-lag-exporter"
)

type LagExporterResourceProvider struct {
	cr              *kafkaservice.KafkaService
	logger          logr.Logger
	spec            *kafkaservice.Monitoring
	lagExporterName string
}

func NewLagExporterResourceProvider(cr *kafkaservice.KafkaService, logger logr.Logger) LagExporterResourceProvider {
	return LagExporterResourceProvider{
		cr:              cr,
		spec:            cr.Spec.Monitoring,
		logger:          logger,
		lagExporterName: fmt.Sprintf("%s-%s", cr.Name, lagExporterNameSuffix),
	}
}

func (lep LagExporterResourceProvider) getPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          "http",
			ContainerPort: int32(lep.spec.LagExporter.MetricsPort),
			Protocol:      corev1.ProtocolTCP},
	}
}

func (lep LagExporterResourceProvider) getEnvs(cmVersion string) []corev1.EnvVar {
	kafkaSecretName := fmt.Sprintf("%s-services-secret", lep.cr.Name)
	return []corev1.EnvVar{
		{
			Name:      "KAFKA_USER",
			ValueFrom: getSecretEnvVarSource(kafkaSecretName, "client-username"),
		},
		{
			Name:      "KAFKA_PASSWORD",
			ValueFrom: getSecretEnvVarSource(kafkaSecretName, "client-password"),
		},
		{
			Name:  "KAFKA_SASL_MECHANISM",
			Value: lep.cr.Spec.Global.KafkaSaslMechanism,
		},
		{
			Name:  "KAFKA_ENABLE_SSL",
			Value: strconv.FormatBool(lep.cr.Spec.Global.KafkaSsl.Enabled),
		},
		{
			Name:  "LAG_EXPORTER_CM_VERSION",
			Value: cmVersion,
		},
	}
}

func (lep LagExporterResourceProvider) getVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      volumeName,
			MountPath: "/opt/docker/src/",
		},
	}
	if lep.cr.Spec.Global.KafkaSsl.Enabled && lep.cr.Spec.Global.KafkaSsl.SecretName != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "ssl-certs", MountPath: "/tls"})
	}
	return volumeMounts
}

func (lep LagExporterResourceProvider) getVolume() corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: fmt.Sprintf("%s-configmap", lep.lagExporterName),
			}}}}
}

func (lep LagExporterResourceProvider) getContainer(cmVersion string) corev1.Container {
	return corev1.Container{
		Name:            lagExporterContainerName,
		Image:           lep.spec.LagExporter.DockerImage,
		Ports:           lep.getPorts(),
		Env:             lep.getEnvs(cmVersion),
		Resources:       lep.spec.Resources,
		VolumeMounts:    lep.getVolumeMounts(),
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: getDefaultContainerSecurityContext(),
	}
}
