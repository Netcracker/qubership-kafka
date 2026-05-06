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
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// getDefaultContainerSecurityContext returns the default container
// security context for deployment to a restricted environment. Workloads
// whose entrypoint still mutates files under the image root filesystem
// must call getContainerSecurityContext(false) and migrate to the
// default once their writes are redirected to a writable mount.
func getDefaultContainerSecurityContext() *corev1.SecurityContext {
	return getContainerSecurityContext(true)
}

// getContainerSecurityContext returns a restricted container security
// context. The readOnlyRootFs argument controls whether the root
// filesystem is mounted read-only.
func getContainerSecurityContext(readOnlyRootFs bool) *corev1.SecurityContext {
	falseValue := false
	readOnlyRoot := readOnlyRootFs
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: &falseValue,
		ReadOnlyRootFilesystem:   &readOnlyRoot,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

// tmpVolumeName is the name of the emptyDir volume mounted at /tmp on
// every operator-built pod. Required when readOnlyRootFilesystem is
// enabled so that JVM, Python and shell scratch writes succeed.
const tmpVolumeName = "tmp"

// getTmpVolume returns a small emptyDir volume to be mounted at /tmp.
func getTmpVolume() corev1.Volume {
	sizeLimit := resource.MustParse("100Mi")
	return corev1.Volume{
		Name: tmpVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				SizeLimit: &sizeLimit,
			},
		},
	}
}

// getTmpVolumeMount returns the matching VolumeMount for getTmpVolume.
func getTmpVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      tmpVolumeName,
		MountPath: "/tmp",
	}
}
