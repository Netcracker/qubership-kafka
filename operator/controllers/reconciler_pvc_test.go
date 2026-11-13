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
	"strings"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckPvcResizeJob(t *testing.T) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "pvc-kafka-1",
			Namespace:   "kafka",
			Annotations: map[string]string{pvcChangedByJobFromAnnotation: "1Gi"},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")},
			},
		},
	}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
	r := &Reconciler{Client: c}

	err := r.CheckPvcResizeJob(pvc.Namespace, pvc.Name, "1Gi")
	if err == nil || !strings.Contains(err.Error(), "deploy parameter is 1Gi") {
		t.Fatalf("error %v", err)
	}
	if err := r.CheckPvcResizeJob(pvc.Namespace, pvc.Name, "2Gi"); err != nil {
		t.Fatal(err)
	}
	if err := r.CheckPvcResizeJob(pvc.Namespace, pvc.Name, "2048Mi"); err != nil {
		t.Fatal(err)
	}
	if err := r.CheckPvcResizeJob(pvc.Namespace, "missing", "1Gi"); err != nil {
		t.Fatal(err)
	}
}

func TestUpdatePersistentVolumeClaimRemovesJobAnnotation(t *testing.T) {
	found := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-kafka-1",
			Namespace: "kafka",
			Annotations: map[string]string{
				pvcChangedByJobFromAnnotation: "1Gi",
				"extra":                       "keep-me",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")},
			},
		},
	}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(found.DeepCopy()).Build()
	r := &Reconciler{Client: c}
	desired := found.DeepCopy()
	desired.Annotations = nil

	if err := r.updatePersistentVolumeClaim(found, desired, nil, logr.Discard()); err != nil {
		t.Fatal(err)
	}
	stored := &corev1.PersistentVolumeClaim{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: found.Name, Namespace: found.Namespace}, stored); err != nil {
		t.Fatal(err)
	}
	if _, exists := stored.Annotations[pvcChangedByJobFromAnnotation]; exists {
		t.Fatalf("annotation still present: %v", stored.Annotations)
	}
	if stored.Annotations["extra"] != "keep-me" {
		t.Fatalf("annotations = %v", stored.Annotations)
	}
}
