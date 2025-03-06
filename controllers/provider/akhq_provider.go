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
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	kafkaservice "github.com/Netcracker/qubership-kafka/api/v7"
	"github.com/Netcracker/qubership-kafka/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	defaultKafkaPollTimeout          = 10000
	deserializationSourcesVolumeName = "proto-keys"
)

// TODO I propose to make additional structs (a.k.a models) for storing such information like consts, and additional logic.
// Such isolation will be helpfull in case of solving issue related to updating diferent fields on different bussiness requirements
const (
	securityGroupConfigK8SKey = "security_groups_config"
	basicGroupsConfigK8SKey   = "basic_auth_users_config"
	ldapServerConfigK8SKey    = "ldap_server_config"
	ldapUsersConfigK8SKey     = "ldap_users_config"
)

type AkhqResourceProvider struct {
	cr          *kafkaservice.KafkaService
	logger      logr.Logger
	spec        *kafkaservice.Akhq
	serviceName string
}

func NewAkhqResourceProvider(
	cr *kafkaservice.KafkaService,
	logger logr.Logger) AkhqResourceProvider {
	return AkhqResourceProvider{
		cr:          cr,
		spec:        cr.Spec.Akhq,
		logger:      logger,
		serviceName: "akhq"}
}

func (arp AkhqResourceProvider) GetAkhqLabels() map[string]string {
	labels := make(map[string]string)
	labels["app.kubernetes.io/name"] = arp.serviceName
	labels = util.JoinMaps(util.JoinMaps(labels, arp.GetAkhqSelectorLabels()), arp.cr.Spec.Global.DefaultLabels)
	return labels
}

func (arp AkhqResourceProvider) GetAkhqSelectorLabels() map[string]string {
	return map[string]string{
		"component": "akhq",
		"name":      arp.serviceName,
	}
}

func (arp AkhqResourceProvider) GetCustomAkhqLabels(akhqLabels map[string]string) map[string]string {
	globalLabels := arp.cr.Spec.Global.CustomLabels
	customLabels := arp.spec.CustomLabels
	return util.JoinMaps(util.JoinMaps(globalLabels, customLabels), akhqLabels)
}

func (arp AkhqResourceProvider) GetServiceName() string {
	return arp.serviceName
}

func (arp AkhqResourceProvider) GetServiceAccountName() string {
	return arp.GetServiceName()
}

func getKafkaPollTimeout(akhq *kafkaservice.Akhq) int64 {
	if akhq.KafkaPollTimeout != nil {
		return *akhq.KafkaPollTimeout
	}
	return defaultKafkaPollTimeout
}

func (arp AkhqResourceProvider) isAccessLogEnabled() bool {
	if arp.cr.Spec.Akhq.EnableAccessLog != nil {
		return *arp.cr.Spec.Akhq.EnableAccessLog
	}
	return false
}

func (arp AkhqResourceProvider) NewAkhqClientService() *corev1.Service {
	akhqLabels := arp.GetAkhqLabels()
	selectorLabels := arp.GetAkhqSelectorLabels()
	ports := []corev1.ServicePort{
		{
			Name:     "akhq-http",
			Protocol: corev1.ProtocolTCP,
			Port:     8080,
		},
	}
	clientService := newServiceForCR(arp.serviceName, arp.cr.Namespace, akhqLabels, selectorLabels, ports)
	return clientService
}

func (arp AkhqResourceProvider) NewLdapTrustedSecret(secretName string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: arp.cr.Namespace,
			Labels:    arp.GetAkhqLabels(),
		},
	}

	if arp.cr.Spec.Akhq.Ldap.Enabled && arp.cr.Spec.Akhq.Ldap.EnableSsl {
		secret.Data = make(map[string][]byte)
		for key, cert := range arp.cr.Spec.Akhq.Ldap.TrustedCerts {
			decodedCert, err := base64.StdEncoding.DecodeString(cert)
			if err != nil {
				return nil
			}
			secret.Data[key] = []byte(decodedCert)
		}
	}
	return secret
}

// ProtectedSecretsFields return list of the protected from update secrets data fields
func (arp AkhqResourceProvider) ProtectedSecretsFields() []string {
	secKeys := []string{
		securityGroupConfigK8SKey,
		basicGroupsConfigK8SKey,
	}
	return secKeys
}

// NewSecurityConfiguration creates secret for security configuration
func (arp AkhqResourceProvider) NewSecurityConfiguration() *corev1.Secret {
	secretName := "akhq-security-configuration"

	stringData := map[string]string{
		securityGroupConfigK8SKey: "",
		basicGroupsConfigK8SKey:   "",
		ldapServerConfigK8SKey:    "",
		ldapUsersConfigK8SKey:     "",
	}

	if arp.cr != nil && arp.cr.Spec.Akhq != nil && arp.cr.Spec.Akhq.Ldap != nil && arp.cr.Spec.Akhq.Ldap.Enabled {
		stringData[ldapServerConfigK8SKey] = arp.GetLdapServerProperties()
		stringData[ldapUsersConfigK8SKey] = arp.GetLdapUsersProperties()
	}
	akhqSecurityConfigurationSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: arp.cr.Namespace,
			Labels:    arp.GetAkhqLabels(),
		},
		StringData: stringData,
	}
	return akhqSecurityConfigurationSecret
}

// NewConfigMapWithKey creates empty config map with specified key for AKHQ
func (arp AkhqResourceProvider) NewConfigMapWithKey(configMapName string, key string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: arp.cr.Namespace,
			Labels:    arp.GetAkhqLabels(),
		},
		Data: map[string]string{key: ""},
	}

	return configMap
}

func (arp AkhqResourceProvider) NewConfigMapWithKeyValue(configMapName, key, value string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: arp.cr.Namespace,
			Labels:    arp.GetAkhqLabels(),
		},
		Data: map[string]string{key: value},
	}
}

func (arp AkhqResourceProvider) NewBinaryConfigMap(configMapName string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: arp.cr.Namespace,
			Labels:    arp.GetAkhqLabels(),
		},
		BinaryData: make(map[string][]byte),
	}
}

func (arp *AkhqResourceProvider) NewConfigurationMapForLdap(configMapName string) *corev1.ConfigMap {
	configurationMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: arp.cr.Namespace,
			Labels:    arp.GetAkhqLabels(),
		},
		Data: map[string]string{
			ldapServerConfigK8SKey: "\n" + arp.GetLdapServerProperties(),
			ldapUsersConfigK8SKey:  "\n" + arp.GetLdapUsersProperties(),
		},
	}
	return configurationMap
}

func (arp AkhqResourceProvider) NewAkhqDeployment(protobufConfigMapVersion string, deserializationConfigMaps []*corev1.ConfigMap) *appsv1.Deployment {
	deploymentName := arp.serviceName
	akhqLabels := arp.GetAkhqLabels()
	akhqLabels["app.kubernetes.io/technology"] = "java-others"
	akhqLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-%s", deploymentName, arp.cr.Namespace)
	akhqCustomLabels := arp.GetCustomAkhqLabels(akhqLabels)
	selectorLabels := arp.GetAkhqSelectorLabels()
	replicas := int32(1)
	volumes := arp.getAkhqVolumes(deserializationConfigMaps)
	volumeMounts := arp.getAkhqVolumeMounts(deserializationConfigMaps)
	ports := []corev1.ContainerPort{
		{
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8080,
		},
		{
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8081,
		},
	}
	envVars := arp.getAkhqEnvironmentVariables(protobufConfigMapVersion)
	envVars = append(envVars, arp.getSecretEnvs()...)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: arp.cr.Namespace,
			Labels:    akhqLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: akhqCustomLabels,
				},
				Spec: corev1.PodSpec{
					Volumes:        volumes,
					InitContainers: arp.getInitContainers(),
					Containers: []corev1.Container{
						{
							Name:  deploymentName,
							Image: arp.spec.DockerImage,
							Ports: ports,
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{Port: intstr.IntOrString{IntVal: 8080}},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								PeriodSeconds:       15,
								SuccessThreshold:    1,
								FailureThreshold:    5,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/health",
										Port:   intstr.IntOrString{IntVal: 8081},
										Scheme: "HTTP",
									},
								},
								InitialDelaySeconds: 40,
								TimeoutSeconds:      15,
								PeriodSeconds:       15,
								SuccessThreshold:    1,
								FailureThreshold:    5,
							},
							Env:             buildEnvs(envVars, arp.spec.EnvironmentVariables, arp.logger),
							Resources:       arp.spec.Resources,
							VolumeMounts:    volumeMounts,
							ImagePullPolicy: corev1.PullAlways,
							Command:         arp.getCommand(),
							Args:            arp.getArgs(),
							SecurityContext: getDefaultContainerSecurityContext(),
						},
					},
					SecurityContext:    &arp.spec.SecurityContext,
					ServiceAccountName: arp.GetServiceAccountName(),
					Hostname:           deploymentName,
					Affinity:           &arp.spec.Affinity,
					Tolerations:        arp.spec.Tolerations,
					PriorityClassName:  arp.spec.PriorityClassName,
				},
			},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
		},
	}
}

// getAkhqVolumes configures the list of AKHQ volumes
func (arp AkhqResourceProvider) getAkhqVolumes(deserializationConfigMaps []*corev1.ConfigMap) []corev1.Volume {
	mode := new(int32)
	*mode = 420
	volumes := []corev1.Volume{
		{
			Name: "trusted-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "akhq-trusted-certs"},
			},
		},
	}

	if len(deserializationConfigMaps) > 0 {
		volumes = append(volumes, arp.createProjectedVolumeForProtoKeysMaps(deserializationConfigMaps))
	}

	if arp.cr.Spec.Global.KafkaSsl.Enabled && arp.cr.Spec.Global.KafkaSsl.SecretName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "ssl-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: arp.cr.Spec.Global.KafkaSsl.SecretName,
				},
			},
		})
	}

	if arp.cr.Spec.Akhq.Ldap.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: "ldap-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "akhq-security-configuration"},
				},
			},
		})
	}

	return volumes
}

func (arp AkhqResourceProvider) createProjectedVolumeForProtoKeysMaps(deserializationConfigMaps []*corev1.ConfigMap) corev1.Volume {
	projectedVolumeSources := make([]corev1.VolumeProjection, 0, len(deserializationConfigMaps))

	for _, cm := range deserializationConfigMaps {
		volumeProjection := corev1.VolumeProjection{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cm.Name,
				},
			},
		}
		projectedVolumeSources = append(projectedVolumeSources, volumeProjection)
	}

	return corev1.Volume{
		Name: deserializationSourcesVolumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: projectedVolumeSources,
			},
		},
	}
}

// getAkhqVolumeMounts configures the list of AKHQ volume mounts
func (arp AkhqResourceProvider) getAkhqVolumeMounts(deserializationConfigMaps []*corev1.ConfigMap) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "trusted-certs",
			MountPath: "/app/trustcerts",
		},
	}
	if len(deserializationConfigMaps) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: deserializationSourcesVolumeName, MountPath: "/app/config/descs-compressed"})
	}
	if arp.cr.Spec.Global.KafkaSsl.Enabled && arp.cr.Spec.Global.KafkaSsl.SecretName != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "ssl-certs", MountPath: "/tls"})
	}
	if arp.cr.Spec.Akhq.Ldap.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "ldap-config", MountPath: "/app/config/ldap"})
	}
	return volumeMounts
}

// getAkhqEnvironmentVariables configures the list of AKHQ environment variables
func (arp AkhqResourceProvider) getAkhqEnvironmentVariables(protobufConfigMapVersion string) []corev1.EnvVar {
	protobufConfigMapName := fmt.Sprintf("%s-protobuf-configuration", arp.GetServiceName())
	envVars := []corev1.EnvVar{
		{
			Name:  "KAFKA_SERVICE_NAME",
			Value: arp.cr.Name,
		},
		{
			Name:  "KAFKA_POLL_TIMEOUT_MS",
			Value: strconv.FormatInt(getKafkaPollTimeout(arp.cr.Spec.Akhq), 10),
		},
		{
			Name:  "BOOTSTRAP_SERVERS",
			Value: arp.spec.BootstrapServers,
		},
		{
			Name:  "KAFKA_SASL_MECHANISM",
			Value: arp.cr.Spec.Global.KafkaSaslMechanism,
		},
		{
			Name:  "KAFKA_ENABLE_SSL",
			Value: strconv.FormatBool(arp.cr.Spec.Global.KafkaSsl.Enabled),
		},
		{
			Name:  "ENABLE_ACCESS_LOG",
			Value: strconv.FormatBool(arp.isAccessLogEnabled()),
		},
		{
			Name:      "PROTOBUF_CONFIGURATION",
			ValueFrom: getConfigMapEnvVarSource(protobufConfigMapName, "config"),
		},
		{
			Name:  "PROTOBUF_CONFIGURATION_VERSION",
			Value: protobufConfigMapVersion,
		},
	}
	if arp.cr.Spec.Akhq.HeapSize != nil {
		envVar := corev1.EnvVar{
			Name:  "HEAP_OPTS",
			Value: fmt.Sprintf("-Xms%dm -Xmx%dm", *arp.cr.Spec.Akhq.HeapSize, *arp.cr.Spec.Akhq.HeapSize),
		}
		envVars = append(envVars, envVar)
	}
	if arp.spec.SchemaRegistryUrl != "" {
		envVars = append(envVars, []corev1.EnvVar{
			{
				Name:  "SCHEMA_REGISTRY_URL",
				Value: arp.spec.SchemaRegistryUrl,
			},
			{
				Name:  "SCHEMA_REGISTRY_TYPE",
				Value: arp.spec.SchemaRegistryType,
			},
		}...)
	}
	return envVars
}

func (arp AkhqResourceProvider) getSecretEnvs() []corev1.EnvVar {
	kafkaSecretName := fmt.Sprintf("%s-services-secret", arp.cr.Name)
	akhqSecretName := "akhq-secret"
	schemaRegistrySecretName := "schema-registry-secret"
	ldapSecretName := "akhq-ldap-secret"
	akhqSecurityConfigurationName := "akhq-security-configuration"
	envVars := []corev1.EnvVar{
		{
			Name:      "KAFKA_AUTH_USERNAME",
			ValueFrom: getSecretEnvVarSource(kafkaSecretName, "client-username"),
		},
		{
			Name:      "KAFKA_AUTH_PASSWORD",
			ValueFrom: getSecretEnvVarSource(kafkaSecretName, "client-password"),
		},
		{
			Name:      "AKHQ_DEFAULT_USER",
			ValueFrom: getSecretEnvVarSource(akhqSecretName, "akhq_default_user"),
		},
		{
			Name:      "AKHQ_DEFAULT_PASSWORD",
			ValueFrom: getSecretEnvVarSource(akhqSecretName, "akhq_default_password"),
		},
		{
			Name:      "SECURITY_GROUPS_CONFIGURATION",
			ValueFrom: getSecretEnvVarSource(akhqSecurityConfigurationName, securityGroupConfigK8SKey),
		},
		{
			Name:      "BASIC_AUTH_USERS_CONFIGURATION",
			ValueFrom: getSecretEnvVarSource(akhqSecurityConfigurationName, basicGroupsConfigK8SKey),
		},
		{
			Name:      "LDAP_SERVER_CONFIGURATION",
			ValueFrom: getSecretEnvVarSource(akhqSecurityConfigurationName, ldapServerConfigK8SKey),
		},
		{
			Name:      "LDAP_USERS_CONFIGURATION",
			ValueFrom: getSecretEnvVarSource(akhqSecurityConfigurationName, ldapUsersConfigK8SKey),
		},
	}

	if arp.cr.Spec.Akhq.Ldap.Enabled {
		envVars = append(envVars, []corev1.EnvVar{
			{
				Name:      "LDAP_ADMIN_USER",
				ValueFrom: getSecretEnvVarSource(ldapSecretName, "LDAP_ADMIN_USER"),
			},
			{
				Name:      "LDAP_ADMIN_PASSWORD",
				ValueFrom: getSecretEnvVarSource(ldapSecretName, "LDAP_ADMIN_PASSWORD"),
			},
		}...)
	}

	if arp.spec.SchemaRegistryUrl != "" {
		envVars = append(envVars, []corev1.EnvVar{
			{
				Name:      "SCHEMA_REGISTRY_USERNAME",
				ValueFrom: getSecretEnvVarSource(schemaRegistrySecretName, "username"),
			},
			{
				Name:      "SCHEMA_REGISTRY_PASSWORD",
				ValueFrom: getSecretEnvVarSource(schemaRegistrySecretName, "password"),
			},
		}...)
	}
	return envVars
}

func (arp AkhqResourceProvider) getCommand() []string {
	return nil
}

func (arp AkhqResourceProvider) getArgs() []string {
	return nil
}

func (arp AkhqResourceProvider) getInitContainers() []corev1.Container {
	return nil
}

func (arp *AkhqResourceProvider) GetLdapServerProperties() string {
	ldapConfig := []string{}

	if arp.cr.Spec.Akhq.Ldap.Enabled {
		serverScheme := "ldap"
		if arp.cr.Spec.Akhq.Ldap.EnableSsl {
			serverScheme = "ldaps"
		}
		ldapConfig = append(ldapConfig,
			"context:",
			fmt.Sprintf("  server: '%s://%s'", serverScheme, arp.cr.Spec.Akhq.Ldap.Server.Context.Server),
			fmt.Sprintf("  managerDn: '%s'", arp.cr.Spec.Akhq.Ldap.Server.Context.ManagerDn),
			fmt.Sprintf("  managerPassword: '%s'", arp.cr.Spec.Akhq.Ldap.Server.Context.ManagerPass),
			"search:",
			fmt.Sprintf("  base: '%s'", arp.cr.Spec.Akhq.Ldap.Server.Search.Base),
			fmt.Sprintf("  filter: '%s'", arp.cr.Spec.Akhq.Ldap.Server.Search.Filter),
		)
		if arp.cr.Spec.Akhq.Ldap.Server.Groups.Enabled {
			ldapConfig = append(ldapConfig,
				"groups:",
				"  enabled: true",
			)
		}
	}
	return strings.Join(ldapConfig, "\n")
}

func (arp *AkhqResourceProvider) GetLdapUsersProperties() string {
	ldapConfig := []string{}
	if arp.cr.Spec.Akhq.Ldap.Enabled {
		for _, user := range arp.cr.Spec.Akhq.Ldap.UsersConfig.Users {
			if user.Username != "" {
				ldapConfig = append(ldapConfig,
					"ldap:",
					"  users:",
					fmt.Sprintf("    - username: %s", user.Username),
					"      groups:")
				if len(user.Groups) > 0 {
					for _, group := range user.Groups {
						if group != "" {
							ldapConfig = append(ldapConfig, fmt.Sprintf("        - %s", group))
						}
					}
				}
			}
		}
		for _, group := range arp.cr.Spec.Akhq.Ldap.UsersConfig.Groups {
			if group.Name != "" {
				ldapConfig = append(ldapConfig,
					"  groups:",
					fmt.Sprintf("    - name: %s", group.Name),
					"      subgroups:")
				if len(group.Groups) > 0 {
					for _, subgroup := range group.Groups {
						if subgroup != "" {
							ldapConfig = append(ldapConfig, fmt.Sprintf("        - %s", subgroup))
						}
					}
				}
			}
		}
	}
	return strings.Join(ldapConfig, "\n")
}
