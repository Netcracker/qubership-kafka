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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KafkaSpec defines the desired state of Kafka
type KafkaSpec struct {
	WaitForPodsReady        bool                    `json:"waitForPodsReady"`
	PodsReadyTimeout        int                     `json:"podReadinessTimeout"`
	Affinity                v1.Affinity             `json:"affinity,omitempty"`
	Tolerations             []v1.Toleration         `json:"tolerations,omitempty"`
	PriorityClassName       string                  `json:"priorityClassName,omitempty"`
	DisableSecurity         *bool                   `json:"disableSecurity,omitempty"`
	DockerImage             string                  `json:"dockerImage"`
	HeapSize                int                     `json:"heapSize"`
	Oauth                   OAuth                   `json:"oauth,omitempty"`
	Ssl                     Ssl                     `json:"ssl,omitempty"`
	Replicas                int                     `json:"replicas"`
	Scaling                 Scaling                 `json:"scaling,omitempty"`
	Resources               v1.ResourceRequirements `json:"resources,omitempty"`
	SecretName              string                  `json:"secretName"`
	SecurityContext         v1.PodSecurityContext   `json:"securityContext,omitempty"`
	Storage                 Storage                 `json:"storage"`
	GetRacksFromNodeLabels  *bool                   `json:"getRacksFromNodeLabels,omitempty"`
	NodeLabelNameForRack    string                  `json:"nodeLabelNameForRack,omitempty"`
	Racks                   []string                `json:"racks,omitempty"`
	TerminationGracePeriod  *int64                  `json:"terminationGracePeriod"`
	ZookeeperConnect        string                  `json:"zookeeperConnect,omitempty"`
	ZookeeperEnableSsl      bool                    `json:"zookeeperEnableSsl,omitempty"`
	ZookeeperSslSecretName  string                  `json:"zookeeperSslSecretName,omitempty"`
	ZookeeperSetACL         *bool                   `json:"zookeeperSetACL,omitempty"`
	ExternalHostNames       []string                `json:"externalHostNames,omitempty"`
	ExternalPorts           []int                   `json:"externalPorts,omitempty"`
	EnvironmentVariables    []string                `json:"environmentVariables,omitempty"`
	RollbackTimeout         *int32                  `json:"rollbackTimeout,omitempty"`
	HealthCheckTimeout      *int32                  `json:"healthCheckTimeout,omitempty"`
	EnableAuditLogs         *bool                   `json:"enableAuditLogs,omitempty"`
	TokenRolesPath          string                  `json:"tokenRolesPath,omitempty"`
	EnableAuthorization     *bool                   `json:"enableAuthorization,omitempty"`
	DiscoveryEnabled        bool                    `json:"consulDiscovery,omitempty"`
	RegisteredServiceName   string                  `json:"registeredServiceName,omitempty"`
	KafkaDiscoveryMeta      map[string]string       `json:"kafkaDiscoveryMeta,omitempty"`
	KafkaDiscoveryTags      []string                `json:"kafkaDiscoveryTags,omitempty"`
	ConsulAclEnabled        bool                    `json:"consulAclEnabled,omitempty"`
	ConsulAuthMethod        string                  `json:"consulAuthMethod,omitempty"`
	RollingUpdate           bool                    `json:"rollingUpdate,omitempty"`
	CustomLabels            map[string]string       `json:"customLabels,omitempty"`
	DefaultLabels           map[string]string       `json:"defaultLabels,omitempty"`
	DebugContainer          bool                    `json:"debugContainer,omitempty"`
	CCMetricReporterEnabled bool                    `json:"ccMetricReporterEnabled,omitempty"`
	Kraft                   Kraft                   `json:"kraft,omitempty"`
	MigrationController     MigrationController     `json:"migrationController,omitempty"`
}

// Kraft defines Kafka parameters for Kraft
type Kraft struct {
	Enabled          bool `json:"enabled,omitempty"`
	Migration        bool `json:"migration,omitempty"`
	MigrationTimeout int  `json:"migrationTimeout,omitempty"`
}

// MigrationController defines Kafka parameters for Kraft
type MigrationController struct {
	Affinity          v1.Affinity             `json:"affinity,omitempty"`
	Tolerations       []v1.Toleration         `json:"tolerations,omitempty"`
	PriorityClassName string                  `json:"priorityClassName,omitempty"`
	HeapSize          int                     `json:"heapSize"`
	Resources         v1.ResourceRequirements `json:"resources,omitempty"`
	Storage           Storage                 `json:"storage"`
}

// Scaling defines Kafka parameters for scaling out
type Scaling struct {
	ReassignPartitions              *bool `json:"reassignPartitions,omitempty"`
	BrokerDeploymentScaleInEnabled  *bool `json:"brokerDeploymentScaleInEnabled,omitempty"`
	AllBrokersStartTimeoutSeconds   *int  `json:"allBrokersStartTimeoutSeconds,omitempty"`
	TopicReassignmentTimeoutSeconds *int  `json:"topicReassignmentTimeoutSeconds,omitempty"`
}

// OAuth defines OAuth Kafka settings
type OAuth struct {
	ClockSkew             *int    `json:"clockSkew,omitempty"`
	JwkSourceType         *string `json:"jwkSourceType,omitempty"`
	JwksConnectionTimeout *int    `json:"jwksConnectionTimeout,omitempty"`
	JwksReadTimeout       *int    `json:"jwksReadTimeout,omitempty"`
	JwksSizeLimit         *int    `json:"jwksSizeLimit,omitempty"`
}

// Ssl defines Ssl Kafka settings
type Ssl struct {
	Enabled                 bool     `json:"enabled"`
	SecretName              string   `json:"secretName,omitempty"`
	CipherSuites            []string `json:"cipherSuites,omitempty"`
	EnableTwoWaySsl         bool     `json:"enableTwoWaySsl,omitempty"`
	AllowNonencryptedAccess bool     `json:"allowNonencryptedAccess,omitempty"`
}

// Storage defines volumes of Kafka
type Storage struct {
	ClassName []string `json:"className,omitempty"`
	Size      string   `json:"size"`
	Volumes   []string `json:"volumes,omitempty"`
	Nodes     []string `json:"nodes,omitempty"`
	Labels    []string `json:"labels,omitempty"`
}

type KafkaBrokerStatus struct {
	Brokers []string `json:"brokers,omitempty"`
}

type PartitionsReassignmentStatus struct {
	Status string `json:"status,omitempty"`
}

type KraftMigrationStatus struct {
	Status string `json:"status,omitempty"`
}

// KafkaStatus defines the observed state of Kafka
type KafkaStatus struct {
	KafkaBrokerStatus            KafkaBrokerStatus            `json:"kafkaBrokerStatus,omitempty"`
	PartitionsReassignmentStatus PartitionsReassignmentStatus `json:"partitionsReassignmentStatus,omitempty"`
	Conditions                   []StatusCondition            `json:"conditions,omitempty"`
	KraftMigrationStatus         KraftMigrationStatus         `json:"kraftMigrationStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Kafka is the Schema for the kafkas API
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSpec   `json:"spec,omitempty"`
	Status KafkaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KafkaList contains a list of Kafka
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kafka `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kafka{}, &KafkaList{})
}

// StatusCondition contains description of status of KafkaService
type StatusCondition struct {
	// Type - Can be "In progress", "Failed", "Successful" or "Ready".
	Type string `json:"type"`
	// Status - "True" if condition is successfully done and "False" if condition has failed or in progress type.
	Status string `json:"status"`
	// Reason - One-word CamelCase reason for the condition's state.
	Reason string `json:"reason"`
	// Message - Human-readable message indicating details about last transition.
	Message string `json:"message"`
	// LastTransitionTime - Last time the condition transit from one status to another.
	LastTransitionTime string `json:"lastTransitionTime"`
}
