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

package v7

import (
	"github.com/Netcracker/qubership-kafka/operator/api/kmm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Global shows configuration of parameters that are used by all services
type Global struct {
	WaitForPodsReady   bool              `json:"waitForPodsReady"`
	PodsReadyTimeout   int               `json:"podReadinessTimeout"`
	CustomLabels       map[string]string `json:"customLabels,omitempty"`
	DefaultLabels      map[string]string `json:"defaultLabels,omitempty"`
	KafkaSaslMechanism string            `json:"kafkaSaslMechanism,omitempty"`
	KafkaSsl           KafkaSsl          `json:"kafkaSsl,omitempty"`
	Kraft              Kraft             `json:"kraft,omitempty"`
}

// Kraft defines Kafka parameters for Kraft
type Kraft struct {
	Enabled bool `json:"enabled,omitempty"`
}

// KafkaSsl shows ssl configuration
type KafkaSsl struct {
	Enabled    bool   `json:"enabled"`
	SecretName string `json:"secretName,omitempty"`
}

// DisasterRecovery shows Disaster Recovery configuration
type DisasterRecovery struct {
	Mode                   string                 `json:"mode"`
	Region                 string                 `json:"region"`
	NoWait                 bool                   `json:"noWait,omitempty"`
	MirrorMakerReplication MirrorMakerReplication `json:"mirrorMakerReplication,omitempty"`
	TopicsBackup           TopicsBackup           `json:"topicsBackup,omitempty"`
}

// Kafka shows Kafka configuration
type Kafka struct {
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
	ZookeeperConnect        string                  `json:"zookeeperConnect"`
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
	DebugContainer          bool                    `json:"debugContainer,omitempty"`
	CCMetricReporterEnabled bool                    `json:"ccMetricReporterEnabled,omitempty"`
}

// Akhq shows AKHQ configuration
type Akhq struct {
	DockerImage          string                  `json:"dockerImage"`
	Affinity             v1.Affinity             `json:"affinity,omitempty"`
	Tolerations          []v1.Toleration         `json:"tolerations,omitempty"`
	PriorityClassName    string                  `json:"priorityClassName,omitempty"`
	Resources            v1.ResourceRequirements `json:"resources,omitempty"`
	SecurityContext      v1.PodSecurityContext   `json:"securityContext,omitempty"`
	KafkaPollTimeout     *int64                  `json:"kafkaPollTimeout,omitempty"`
	EnableAccessLog      *bool                   `json:"enableAccessLog,omitempty"`
	BootstrapServers     string                  `json:"bootstrapServers,omitempty"`
	KafkaEnableSsl       bool                    `json:"kafkaEnableSsl,omitempty"` // for backwards compatibility, may be removed in the future
	CustomLabels         map[string]string       `json:"customLabels,omitempty"`
	EnvironmentVariables []string                `json:"environmentVariables,omitempty"`
	HeapSize             *int                    `json:"heapSize,omitempty"`
	SchemaRegistryUrl    string                  `json:"schemaRegistryUrl,omitempty"`
	SchemaRegistryType   string                  `json:"schemaRegistryType,omitempty"`
	Ldap                 *LdapConfig             `json:"ldap,omitempty"`
}

// Monitoring shows Kafka Monitoring configuration
type Monitoring struct {
	DockerImage                string                  `json:"dockerImage"`
	Affinity                   v1.Affinity             `json:"affinity,omitempty"`
	Tolerations                []v1.Toleration         `json:"tolerations,omitempty"`
	PriorityClassName          string                  `json:"priorityClassName,omitempty"`
	Resources                  v1.ResourceRequirements `json:"resources,omitempty"`
	SecurityContext            v1.PodSecurityContext   `json:"securityContext,omitempty"`
	MonitoringType             string                  `json:"monitoringType"`
	SmDbHost                   string                  `json:"smDbHost,omitempty"`
	SmDbName                   string                  `json:"smDbName,omitempty"`
	KafkaMeasurementPrefixName string                  `json:"kafkaMeasurementPrefixName,omitempty"`
	DataCollectionInterval     string                  `json:"dataCollectionInterval,omitempty"`
	KafkaExecPluginTimeout     string                  `json:"kafkaExecPluginTimeout,omitempty"`
	SecretName                 string                  `json:"secretName"`
	MinVersion                 string                  `json:"minVersion,omitempty"`
	MaxVersion                 string                  `json:"maxVersion,omitempty"`
	BootstrapServers           string                  `json:"bootstrapServers,omitempty"`
	KafkaEnableSsl             bool                    `json:"kafkaEnableSsl,omitempty"` // for backwards compatibility, may be removed in the future
	KafkaTotalBrokerCount      int                     `json:"kafkaTotalBrokerCount"`
	LagExporter                *LagExporter            `json:"lagExporter,omitempty"`
	CustomLabels               map[string]string       `json:"customLabels,omitempty"`
}

// MirrorMaker shows Kafka Mirror Maker configuration
type MirrorMaker struct {
	DockerImage                  string                  `json:"dockerImage"`
	Affinity                     v1.Affinity             `json:"affinity,omitempty"`
	Tolerations                  []v1.Toleration         `json:"tolerations,omitempty"`
	PriorityClassName            string                  `json:"priorityClassName,omitempty"`
	HeapSize                     int                     `json:"heapSize"`
	Resources                    v1.ResourceRequirements `json:"resources,omitempty"`
	Replicas                     int                     `json:"replicas"`
	Clusters                     []Cluster               `json:"clusters"`
	TopicsToReplicate            string                  `json:"topicsToReplicate,omitempty"`
	ReplicationFactor            int                     `json:"replicationFactor"`
	ConfiguratorEnabled          bool                    `json:"configuratorEnabled,omitempty"`
	RefreshTopicsIntervalSeconds *int32                  `json:"refreshTopicsIntervalSeconds,omitempty"`
	RefreshGroupsIntervalSeconds *int32                  `json:"refreshGroupsIntervalSeconds,omitempty"`
	ConfigurationName            string                  `json:"configurationName"`
	SecretName                   string                  `json:"secretName"`
	SecurityContext              v1.PodSecurityContext   `json:"securityContext,omitempty"`
	EnvironmentVariables         []string                `json:"environmentVariables,omitempty"`
	RegionName                   string                  `json:"regionName,omitempty"`
	RepeatedReplication          *bool                   `json:"repeatedReplication,omitempty"`
	CustomLabels                 map[string]string       `json:"customLabels,omitempty"`
	ReplicationFlowEnabled       bool                    `json:"replicationFlowEnabled,omitempty"`
	ReplicationPrefixEnabled     bool                    `json:"replicationPrefixEnabled,omitempty"`
	Transformation               *kmm.Transformation     `json:"transformation,omitempty"`
	TasksMax                     *int32                  `json:"tasksMax,omitempty"`
	InternalRestEnabled          *bool                   `json:"internalRestEnabled,omitempty"`
	// Deprecated: it is kept for backward compatibility.
	JolokiaPort *int32 `json:"jolokiaPort,omitempty"`
}

type Cluster struct {
	Name             string `json:"name"`
	BootstrapServers string `json:"bootstrapServers"`
	NodeLabel        string `json:"nodeLabel,omitempty"`
	EnableSsl        bool   `json:"enableSsl,omitempty"`
	SslSecretName    string `json:"sslSecretName,omitempty"`
	SaslMechanism    string `json:"saslMechanism,omitempty"`
}

// MirrorMakerMonitoring shows Kafka Mirror Maker Monitoring configuration
type MirrorMakerMonitoring struct {
	DockerImage           string                  `json:"dockerImage"`
	Affinity              v1.Affinity             `json:"affinity,omitempty"`
	Tolerations           []v1.Toleration         `json:"tolerations,omitempty"`
	PriorityClassName     string                  `json:"priorityClassName,omitempty"`
	Resources             v1.ResourceRequirements `json:"resources,omitempty"`
	SecurityContext       v1.PodSecurityContext   `json:"securityContext,omitempty"`
	MonitoringType        string                  `json:"monitoringType"`
	SmDbHost              string                  `json:"smDbHost,omitempty"`
	SmDbName              string                  `json:"smDbName,omitempty"`
	KmmExecPluginTimeout  string                  `json:"kmmExecPluginTimeout,omitempty"`
	KmmCollectionInterval string                  `json:"kmmCollectionInterval,omitempty"`
	SecretName            string                  `json:"secretName"`
	CustomLabels          map[string]string       `json:"customLabels,omitempty"`
	// Deprecated: it is kept for backward compatibility.
	JolokiaUrls []string `json:"jolokiaUrls,omitempty"`
}

// VaultSecretManagement shows Vault secret management configuration
// Deprecated
type VaultSecretManagement struct {
	DockerImage                 string      `json:"dockerImage"`
	Enabled                     bool        `json:"enabled,omitempty"`
	Path                        string      `json:"path,omitempty"`
	Url                         string      `json:"url,omitempty"`
	Role                        string      `json:"role,omitempty"`
	Method                      string      `json:"method,omitempty"`
	PasswordGenerationMechanism string      `json:"passwordGenerationMechanism,omitempty"`
	WritePolicies               bool        `json:"writePolicies,omitempty"`
	SecretPaths                 SecretPaths `json:"secretPaths,omitempty"`
}

type SecretPaths struct {
	Kafka                 map[string]string `json:"kafka,omitempty"`
	Monitoring            map[string]string `json:"monitoring,omitempty"`
	MirrorMaker           map[string]string `json:"mirrorMaker,omitempty"`
	MirrorMakerMonitoring map[string]string `json:"mirrorMakerMonitoring,omitempty"`
}

// IntegrationTests shows Integration Tests configuration
type IntegrationTests struct {
	ServiceName      string `json:"serviceName"`
	WaitForResult    bool   `json:"waitForResult"`
	Timeout          int    `json:"timeout"`
	RandomRunTrigger string `json:"randomRunTrigger,omitempty"`
}

// BackupDaemon defines the specific Kafka Backup Daemon configuration
type BackupDaemon struct {
	Name string `json:"name"`
}

type KafkaStatus struct {
	Brokers []string `json:"brokers,omitempty"`
}

type PartitionsReassignmentStatus struct {
	Status string `json:"status,omitempty"`
}

type AkhqStatus struct {
	Nodes []string `json:"nodes,omitempty"`
}

type MonitoringStatus struct {
	Nodes []string `json:"nodes,omitempty"`
}

type MirrorMakerStatus struct {
	Nodes []string `json:"nodes,omitempty"`
}

type DisasterRecoveryStatus struct {
	Mode    string `json:"mode"`
	Status  string `json:"status"`
	Comment string `json:"comment,omitempty"` // deprecated
	Message string `json:"message,omitempty"`
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

// KafkaServiceSpec defines the desired state of KafkaService
type KafkaServiceSpec struct {
	Global                *Global                `json:"global,omitempty"`
	DisasterRecovery      *DisasterRecovery      `json:"disasterRecovery,omitempty"`
	Kafka                 *Kafka                 `json:"kafka,omitempty"`
	Akhq                  *Akhq                  `json:"akhq,omitempty"`
	Monitoring            *Monitoring            `json:"monitoring,omitempty"`
	MirrorMaker           *MirrorMaker           `json:"mirrorMaker,omitempty"`
	MirrorMakerMonitoring *MirrorMakerMonitoring `json:"mirrorMakerMonitoring,omitempty"`
	VaultSecretManagement *VaultSecretManagement `json:"vaultSecretManagement,omitempty"`
	IntegrationTests      *IntegrationTests      `json:"integrationTests,omitempty"`
	BackupDaemon          *BackupDaemon          `json:"backupDaemon,omitempty"`
}

// KafkaServiceStatus defines the observed state of KafkaService
type KafkaServiceStatus struct {
	KafkaStatus                  KafkaStatus                  `json:"kafkaStatus,omitempty"`
	PartitionsReassignmentStatus PartitionsReassignmentStatus `json:"partitionsReassignmentStatus,omitempty"`
	AkhqStatus                   AkhqStatus                   `json:"akhqStatus,omitempty"`
	MonitoringStatus             MonitoringStatus             `json:"monitoringStatus,omitempty"`
	MirrorMakerStatus            MirrorMakerStatus            `json:"mirrorMakerStatus,omitempty"`
	VaultSecretManagementStatus  VaultSecretManagementStatus  `json:"vaultSecretManagementStatus,omitempty"`
	DisasterRecoveryStatus       DisasterRecoveryStatus       `json:"disasterRecoveryStatus,omitempty"`
	Conditions                   []StatusCondition            `json:"conditions,omitempty"`
}

type VaultSecretManagementStatus struct {
	SecretVersions map[string]int64 `json:"secretVersions,omitempty"`
}

type MirrorMakerReplication struct {
	Enabled bool `json:"enabled,omitempty"`
}

type TopicsBackup struct {
	Enabled bool `json:"enabled,omitempty"`
}

// Storage defines volumes of Kafka
type Storage struct {
	ClassName []string `json:"className,omitempty"`
	Size      string   `json:"size"`
	Volumes   []string `json:"volumes,omitempty"`
	Nodes     []string `json:"nodes,omitempty"`
	Labels    []string `json:"labels,omitempty"`
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
	CipherSuites            []string `json:"cipherSuites,omitempty"`
	EnableTwoWaySsl         bool     `json:"enableTwoWaySsl,omitempty"`
	AllowNonencryptedAccess bool     `json:"allowNonencryptedAccess,omitempty"`
}

// Scaling defines Kafka parameters for scaling out
type Scaling struct {
	ReassignPartitions              *bool `json:"reassignPartitions,omitempty"`
	BrokerDeploymentScaleInEnabled  *bool `json:"brokerDeploymentScaleInEnabled,omitempty"`
	AllBrokersStartTimeoutSeconds   *int  `json:"allBrokersStartTimeoutSeconds,omitempty"`
	TopicReassignmentTimeoutSeconds *int  `json:"topicReassignmentTimeoutSeconds,omitempty"`
}

type LagExporter struct {
	DockerImage string `json:"dockerImage,omitempty"`
	MetricsPort int    `json:"metricsPort,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// KafkaService is the Schema for the kafkaservices API
type KafkaService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KafkaServiceSpec   `json:"spec,omitempty"`
	Status            KafkaServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KafkaServiceList contains a list of KafkaService
type KafkaServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaService `json:"items"`
}

type LdapConfig struct {
	Enabled      bool              `json:"enabled"`
	EnableSsl    bool              `json:"enableSsl"`
	TrustedCerts map[string]string `json:"trustedCerts"`
	Server       LdapServer        `json:"server"`
	UsersConfig  LdapUsersConfig   `json:"usersconfig"`
}

type LdapServer struct {
	Context LdapServerContext `json:"context"`
	Search  LdapServerSearch  `json:"search"`
	Groups  LdapServerGroups  `json:"groups"`
}

type LdapServerContext struct {
	Server      string `json:"server"`
	ManagerDn   string `json:"managerDn"`
	ManagerPass string `json:"managerPassword"`
}

type LdapServerSearch struct {
	Base   string `json:"base"`
	Filter string `json:"filter"`
}

type LdapServerGroups struct {
	Enabled bool `json:"enabled"`
}

type LdapUsersConfig struct {
	Groups []LdapGroup `json:"groups"`
	Users  []LdapUser  `json:"users"`
}

type LdapGroup struct {
	Name   string   `json:"name"`
	Groups []string `json:"groups"`
}

type LdapUser struct {
	Username string   `json:"username"`
	Groups   []string `json:"groups"`
}

func init() {
	SchemeBuilder.Register(&KafkaService{}, &KafkaServiceList{})
}
