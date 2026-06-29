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

package kafkaservice

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// applyAutoRestartSecretAnnotations sets pod-template annotations from Secret resourceVersion when the
// Secret has kafkaservice.netcracker.com/auto-restart: "true". Returns true if annotations changed
// (a pod-template change triggers a rolling restart for startup-only entrypoints).
func applyAutoRestartSecretAnnotations(deployment *appsv1.Deployment, logger logr.Logger, secrets ...*corev1.Secret) bool {
	modified := false
	for _, secret := range secrets {
		if secret == nil || secret.Name == "" {
			continue
		}
		if secret.Annotations == nil || secret.Annotations[autoRestartAnnotation] != "true" {
			continue
		}
		annotationName := fmt.Sprintf(resourceVersionAnnotationTemplate, secret.Name)
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = map[string]string{}
		}
		if deployment.Spec.Template.Annotations[annotationName] == secret.ResourceVersion {
			continue
		}
		logger.Info(fmt.Sprintf("Add annotation '%s: %s' to deployment '%s'",
			annotationName, secret.ResourceVersion, deployment.Name))
		deployment.Spec.Template.Annotations[annotationName] = secret.ResourceVersion
		modified = true
	}
	return modified
}

// updateDeploymentSecretRestartAnnotations patches an existing Deployment in the cluster.
// This matches the backup-daemon flow: load live object (with resourceVersion), set pod-template
// annotations, then Update — so secret rotation triggers a rollout even when the reconciler
// skips rebuilding the full Deployment spec.
func updateDeploymentSecretRestartAnnotations(
	c client.Client,
	namespace, deploymentName string,
	logger logr.Logger,
	secrets ...*corev1.Secret,
) error {
	deployment := &appsv1.Deployment{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: deploymentName, Namespace: namespace}, deployment)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !applyAutoRestartSecretAnnotations(deployment, logger, secrets...) {
		return nil
	}
	return c.Update(context.TODO(), deployment)
}
