package cfg

type OpMode string

const KafkaMode = OpMode("kafka")
const KafkaServiceMode = OpMode("kafkaservice")

type Cfg struct {
	MetricsAddr                              string  `long:"metrics-bind-address" description:"The address the metric endpoint binds to." default:":8082"`
	ProbeAddr                                string  `long:"health-probe-bind-address" description:"The address the probe endpoint binds to." default:":8081"`
	EnableLeaderElection                     bool    `long:"leader-elect" description:"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager."`
	OwnNamespace                             string  `long:"own-namespace" description:"The own namespace" env:"OWN_NAMESPACE"`
	Mode                                     OpMode  `long:"mode" description:"The operation mode" env:"OPERATOR_MODE"`
	ApiGroup                                 string  `long:"api-group" description:"The API group" env:"API_GROUP" default:"qubership.org"`
	SecondaryApiGroup                        string  `long:"secondary-api-group" description:"The additional API group" optional:"true" env:"SECONDARY_API_GROUP"`
	KmmEnabled                               bool    `long:"kmm-enabled" description:"Enable kmm manager" env:"KMM_ENABLED"`
	KmmConfigurationReconcilePeriodSecs      int     `long:"kmm-configuration-reconcile-period-seconds" description:"Reconcilation period for Kafka KMM configuration" env:"KMM_CONFIG_RECONCILE_PERIOD_SECONDS" default:"60"`
	WatchAkhqCollectNamespace                *string `long:"watch-akhq-collect-namespace" description:"Namespace to watch for Akhq collect" env:"WATCH_AKHQ_COLLECT_NAMESPACE"`
	WatchKafkaUsersCollectNamespace          *string `long:"watch-kafka-users-collect-namespace" description:"Namespace to watch for Kafka Users collect" env:"WATCH_KAFKA_USERS_COLLECT_NAMESPACE"`
	KafkaUserSecretCreatingEnabled           bool    `long:"kafka-user-secret-creating-enabled" description:"Enable Kafka User secret creation" env:"KAFKA_USER_SECRET_CREATING_ENABLED"`
	KafkaUserConfiguratorReconcilePeriodSecs int     `long:"kafka-user-configurator-reconcile-period-seconds" description:"Reconciliation period for Kafka User Configurator in seconds" default:"60" env:"KAFKA_USER_CONFIGURATOR_RECONCILE_PERIOD_SECONDS"`
	KafkaBootstrapServers                    string  `long:"kafka-bootstrap-servers" description:"Kafka bootstrap servers" env:"BOOTSTRAP_SERVERS" optional:"true"`
	KafkaSecret                              string  `long:"kafka-secret" description:"Kafka secret" env:"KAFKA_SECRET"`
	KafkaSaslMechanism                       string  `long:"kafka-sasl-mechanism" description:"Kafka SASL mechanism" env:"KAFKA_SASL_MECHANISM"`
	KafkaSslEnabled                          bool    `long:"kafka-ssl-enabled" description:"Enable Kafka SSL" env:"KAFKA_SSL_ENABLED"`
	KafkaSslSecret                           string  `long:"kafka-ssl-secret" description:"Kafka SSL secret" env:"KAFKA_SSL_SECRET"`
	ClusterName                              string  `long:"cluster-name" description:"Cluster name" env:"CLUSTER_NAME"`
	OperatorNamespace                        string  `long:"operator-namespace" description:"Namespace of the operator" env:"OPERATOR_NAMESPACE"`
	OperatorName                             string  `long:"operator-name" description:"Name of the operator" env:"OPERATOR_NAME"`
}
