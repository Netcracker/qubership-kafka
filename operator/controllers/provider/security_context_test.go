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
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetDefaultContainerSecurityContext(t *testing.T) {
	sc := getDefaultContainerSecurityContext()

	if sc == nil {
		t.Fatal("expected non-nil SecurityContext")
	}
	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
		t.Error("AllowPrivilegeEscalation must be false")
	}
	if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
		t.Error("ReadOnlyRootFilesystem must be true")
	}
	if sc.Capabilities == nil {
		t.Fatal("Capabilities must not be nil")
	}
	if len(sc.Capabilities.Drop) != 1 || sc.Capabilities.Drop[0] != "ALL" {
		t.Errorf("expected Capabilities.Drop=[ALL], got %v", sc.Capabilities.Drop)
	}
}

func TestGetContainerSecurityContextReadOnly(t *testing.T) {
	sc := getContainerSecurityContext(true)

	if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
		t.Error("ReadOnlyRootFilesystem must be true when readOnlyRootFs=true")
	}
}

func TestGetContainerSecurityContextReadWrite(t *testing.T) {
	sc := getContainerSecurityContext(false)

	if sc.ReadOnlyRootFilesystem == nil || *sc.ReadOnlyRootFilesystem {
		t.Error("ReadOnlyRootFilesystem must be false when readOnlyRootFs=false")
	}
	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
		t.Error("AllowPrivilegeEscalation must still be false")
	}
	if len(sc.Capabilities.Drop) != 1 || sc.Capabilities.Drop[0] != "ALL" {
		t.Errorf("expected Capabilities.Drop=[ALL], got %v", sc.Capabilities.Drop)
	}
}

func TestGetTmpVolume(t *testing.T) {
	vol := getTmpVolume()

	if vol.Name != tmpVolumeName {
		t.Errorf("expected volume name %q, got %q", tmpVolumeName, vol.Name)
	}
	if vol.EmptyDir == nil {
		t.Fatal("expected EmptyDir volume source")
	}
	limit := resource.MustParse("100Mi")
	if vol.EmptyDir.SizeLimit == nil || !vol.EmptyDir.SizeLimit.Equal(limit) {
		t.Errorf("expected SizeLimit=100Mi, got %v", vol.EmptyDir.SizeLimit)
	}
}

func TestGetTmpVolumeMount(t *testing.T) {
	vm := getTmpVolumeMount()

	if vm.Name != tmpVolumeName {
		t.Errorf("expected mount name %q, got %q", tmpVolumeName, vm.Name)
	}
	if vm.MountPath != "/tmp" {
		t.Errorf("expected MountPath=/tmp, got %q", vm.MountPath)
	}
}

func TestDefaultSecurityContextMatchesReadOnlyVariant(t *testing.T) {
	def := getDefaultContainerSecurityContext()
	explicit := getContainerSecurityContext(true)

	if *def.ReadOnlyRootFilesystem != *explicit.ReadOnlyRootFilesystem {
		t.Error("default and explicit readOnly=true contexts must match on ReadOnlyRootFilesystem")
	}
	if *def.AllowPrivilegeEscalation != *explicit.AllowPrivilegeEscalation {
		t.Error("default and explicit readOnly=true contexts must match on AllowPrivilegeEscalation")
	}
	if def.Capabilities.Drop[0] != explicit.Capabilities.Drop[0] {
		t.Error("default and explicit readOnly=true contexts must match on Capabilities.Drop")
	}
}

func TestCapabilitiesDropAll(t *testing.T) {
	for _, readOnly := range []bool{true, false} {
		sc := getContainerSecurityContext(readOnly)
		if len(sc.Capabilities.Drop) == 0 {
			t.Errorf("readOnly=%v: Capabilities.Drop must not be empty", readOnly)
		}
		found := false
		for _, cap := range sc.Capabilities.Drop {
			if cap == corev1.Capability("ALL") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("readOnly=%v: Capabilities.Drop must contain ALL, got %v", readOnly, sc.Capabilities.Drop)
		}
	}
}
