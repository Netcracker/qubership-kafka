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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func NewServiceAccount(serviceAccountName string, namespace string, labels map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: namespace, Labels: labels},
	}
}

// newServiceForCR returns service with specified parameters
func newServiceForCR(serviceName string, namespace string, labels map[string]string, selectorLabels map[string]string, ports []corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: selectorLabels,
		}}
}

// newServiceForBroker returns service for broker with specified parameters
func newServiceForBroker(serviceName string, namespace string, labels map[string]string, selectorLabels map[string]string, ports []corev1.ServicePort) *corev1.Service {
	service := newServiceForCR(serviceName, namespace, labels, selectorLabels, ports)
	service.Spec.PublishNotReadyAddresses = true
	service.ObjectMeta.Annotations = map[string]string{
		"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
	}
	return service
}

func newDomainServiceForCR(serviceName string, namespace string, labels map[string]string, selectorLabels map[string]string, ports []corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports:                    ports,
			PublishNotReadyAddresses: true,
			Selector:                 selectorLabels,
			ClusterIP:                "None",
		}}
}

func getSecretEnvVarSource(secretName string, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			Key:                  key,
			LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
		},
	}
}

func getConfigMapEnvVarSource(configMapName string, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			Key:                  key,
			LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
		},
	}
}

// buildEnvs builds array of specified environment variables with additional list of environment variables
func buildEnvs(envVars []corev1.EnvVar, additionalEnvs []string, logger logr.Logger) []corev1.EnvVar {
	envsMap := make(map[string]string)

	for _, envVar := range additionalEnvs {
		envPair := strings.SplitN(envVar, "=", 2)
		if len(envPair) == 2 {
			if name := strings.TrimSpace(envPair[0]); len(name) > 0 {
				value := strings.TrimSpace(envPair[1])
				envsMap[name] = value
				continue
			}
		}
		logger.Info(fmt.Sprintf("Environment variable \"%s\" is incorrect", envVar))
	}
	for name, value := range envsMap {
		envVars = append(envVars, corev1.EnvVar{Name: name, Value: value})
	}
	return envVars
}

// getDefaultContainerSecurityContext returns default security context for containers for deployment to restricted environment
func getDefaultContainerSecurityContext() *corev1.SecurityContext {
	falseValue := false
	return &corev1.SecurityContext{AllowPrivilegeEscalation: &falseValue,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}
