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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KafkaUserSpec defines the desired state of KafkaUser
type KafkaUserSpec struct {
	Authentication Authentication `json:"authentication"`
	Authorization  Authorization  `json:"authorization"`
}

type Authentication struct {
	// +kubebuilder:validation:Enum=scram-sha-512
	Type        string  `json:"type"`
	Secret      *Secret `json:"secret,omitempty"`
	WatchSecret *bool   `json:"watchSecret,omitempty"`
}

type Authorization struct {
	// +kubebuilder:validation:Enum=admin;namespace-admin;namespace-producer;namespace-consumer
	Role string `json:"role"`
}

type Secret struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=connection-properties;connection-uri
	Format   string `json:"format"`
	Generate bool   `json:"generate"`
}

// KafkaUserStatus defines the observed state of KafkaUser
type KafkaUserStatus struct {
	AuthenticationStatus AuthenticationStatus `json:"authenticationStatus,omitempty"`
	AuthorizationStatus  AuthorizationStatus  `json:"authorizationStatus,omitempty"`
	// +kubebuilder:validation:Enum=success;failure;processing
	State              string            `json:"state,omitempty"`
	ResourceVersion    string            `json:"resourceVersion,omitempty"`
	ObservedGeneration int64             `json:"observedGeneration,omitempty"`
	Message            string            `json:"message,omitempty"`
	Conditions         []StatusCondition `json:"conditions,omitempty"`
}

type AuthenticationStatus struct {
	// +kubebuilder:validation:Enum=success;failure;disabled
	State           string    `json:"state,omitempty"`
	Password        SecretKey `json:"password,omitempty"`
	Username        SecretKey `json:"username,omitempty"`
	ConnectionUri   SecretKey `json:"connectionUri,omitempty"`
	ResourceVersion string    `json:"resourceVersion,omitempty"`
}

type SecretKey struct {
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
}

type AuthorizationStatus struct {
	// +kubebuilder:validation:Enum=success;failure;disabled
	State string   `json:"state,omitempty"`
	Acls  []string `json:"acls,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// KafkaUser is the Schema for the kafkausers API
type KafkaUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaUserSpec   `json:"spec,omitempty"`
	Status KafkaUserStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KafkaUserList contains a list of KafkaUser
type KafkaUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaUser{}, &KafkaUserList{})
}
