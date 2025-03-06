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

package v1

import (
	"github.com/Netcracker/qubership-kafka/api/kmm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KmmTopics struct {
	Topics         string              `json:"topics"`
	SourceDc       string              `json:"sourceDc,omitempty"`
	TargetDc       string              `json:"targetDc,omitempty"`
	Transformation *kmm.Transformation `json:"transformation,omitempty"`
}

// KmmConfigSpec defines the desired state of KmmConfig
type KmmConfigSpec struct {
	KmmTopics *KmmTopics `json:"kmmTopics"`
}

// KmmConfigStatus defines the observed state of KmmConfig
type KmmConfigStatus struct {
	IsProcessed   bool   `json:"isProcessed"`
	ProblemTopics string `json:"problemTopics"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// KmmConfig is the Schema for the kmmconfigs API
type KmmConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KmmConfigSpec   `json:"spec,omitempty"`
	Status KmmConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KmmConfigList contains a list of KmmConfig
type KmmConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KmmConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KmmConfig{}, &KmmConfigList{})
}
