{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "kafka-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kafka-service.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kafka-service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kafka-service.globalPodSecurityContext" -}}
runAsNonRoot: true
seccompProfile:
  type: "RuntimeDefault"
{{- with .Values.global.securityContext }}
{{ toYaml . }}
{{- end -}}
{{- end -}}

{{- define "kafka-service.globalContainerSecurityContext" -}}
allowPrivilegeEscalation: false
capabilities:
  drop: ["ALL"]
{{- end -}}

{{/*
Common labels
*/}}
{{- define "kafka-service.labels" -}}
helm.sh/chart: {{ include "kafka-service.chart" . }}
{{ include "kafka-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
The most common Kafka-services operator chart related resources labels
*/}}
{{- define "kafka-services.coreLabels" -}}
app.kubernetes.io/version: '{{ .Values.ARTIFACT_DESCRIPTOR_VERSION | trunc 63 | trimAll "-_." | toString }}'
app.kubernetes.io/part-of: '{{ .Values.PART_OF }}'
{{- end -}}

{{/*
Core Kafka-services operator chart related resources labels with backend component label
*/}}
{{- define "kafka-services.defaultLabels" -}}
{{ include "kafka-services.coreLabels" . }}
app.kubernetes.io/component: 'backend'
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "kafka-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kafka-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "kafka-service.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "kafka-service.fullname" .) .Values.serviceAccount }}
{{- else -}}
    {{ default "default" .Values.serviceAccount }}
{{- end -}}
{{- end -}}

{{/*
Configure Kafka service 'enableDisasterRecovery' property
*/}}
{{- define "kafka-service.enableDisasterRecovery" -}}
{{- if or (eq .Values.global.disasterRecovery.mode "active") (eq .Values.global.disasterRecovery.mode "standby") (eq .Values.global.disasterRecovery.mode "disabled") -}}
    {{- printf "true" }}
{{- else -}}
    {{- printf "false" }}
{{- end -}}
{{- end -}}

{{/*
Configure Disaster Recovery 'topicsBackup' property
*/}}
{{- define "disasterRecovery.topicsBackup" -}}
{{- if (and .Values.global.disasterRecovery.topicsBackup.enabled .Values.backupDaemon.install) -}}
    {{- printf "true" }}
{{- else -}}
    {{- printf "false" }}
{{- end -}}
{{- end -}}

{{/*
Configure Kafka service name
*/}}
{{- define "kafka.name" -}}
{{- coalesce .Values.global.name "kafka" -}}
{{- end -}}

{{/*
Configure Kafka service account
*/}}
{{- define "kafka.serviceAccount" -}}
{{- coalesce .Values.operator.serviceAccount (printf "%s-service-operator" (include "kafka.name" .)) -}}
{{- end -}}


{{/*
DNS names used to generate TLS certificate with "Subject Alternative Name" field
*/}}
{{- define "kafka.certDnsNames" -}}
  {{- $kafkaName := include "kafka.name" . -}}
  {{- $dnsNames := list "localhost" $kafkaName (printf "%s.%s" $kafkaName .Release.Namespace) (printf "%s.%s" $kafkaName "kafka-broker") (printf "%s.%s.%s" $kafkaName "kafka-broker" .Release.Namespace) (printf "%s.%s.svc" $kafkaName .Release.Namespace) -}}
  {{- $brokers := include "kafka.replicas" . -}}
  {{- $kafkaNamespace := .Release.Namespace -}}
  {{- range $i, $e := until ($brokers | int) -}}
    {{- $dnsNames = append $dnsNames (printf "%s-%d" $kafkaName (add $i 1)) -}}
    {{- $dnsNames = append $dnsNames (printf "%s-%d.%s" $kafkaName (add $i 1) $kafkaNamespace) -}}
    {{- $dnsNames = append $dnsNames (printf "%s-%d.kafka-broker.%s" $kafkaName (add $i 1) $kafkaNamespace) -}}
  {{- end -}}
  {{- $dnsNames = concat $dnsNames .Values.kafka.tls.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}

{{/*
IP addresses used to generate TLS certificate with "Subject Alternative Name" field
*/}}
{{- define "kafka.certIpAddresses" -}}
  {{- $ipAddresses := list "127.0.0.1" -}}
  {{- $ipAddresses = concat $ipAddresses .Values.kafka.tls.subjectAlternativeName.additionalIpAddresses -}}
  {{- $ipAddresses | toYaml -}}
{{- end -}}

{{/*
Generate certificates for Kafka broker
*/}}
{{- define "kafka.generateCerts" -}}
  {{- $dnsNames := include "kafka.certDnsNames" . | fromYamlArray -}}
  {{- $ipAddresses := include "kafka.certIpAddresses" . | fromYamlArray -}}
  {{- $duration := default 365 .Values.global.tls.generateCerts.durationDays | int -}}
  {{- $ca := genCA "kafka-ca" $duration -}}
  {{- $kafkaName := include "kafka.name" . -}}
  {{- $cert := genSignedCert $kafkaName $ipAddresses $dnsNames $duration $ca -}}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
ca.crt: {{ $ca.Cert | b64enc }}
{{- end -}}

{{/*
Provider used to generate TLS certificates
*/}}
{{- define "services.certProvider" -}}
  {{- default "helm" .Values.global.tls.generateCerts.certProvider -}}
{{- end -}}

{{/*
Configure Kafka monitoring type
*/}}
{{- define "monitoring.type" -}}
{{- coalesce .Values.monitoring.monitoringType .Values.global.monitoringType "prometheus" -}}
{{- end -}}

{{/*
Configure Kafka Mirror Maker monitoring type
*/}}
{{- define "mirrorMakerMonitoring.type" -}}
{{- coalesce .Values.mirrorMakerMonitoring.monitoringType .Values.global.monitoringType "prometheus" -}}
{{- end -}}


{{/*
Create list of Kafka brokers separated by ",".
For instance, "kafka-1:9092,kafka-2:9092,kafka-3:9092"
*/}}
{{- define "kafka-service.brokersList" -}}
    {{ if .Values.global.externalKafka.enabled }}
      {{- .Values.global.externalKafka.bootstrapServers }}
    {{- else }}
      {{- $brokers := .Values.monitoring.kafkaTotalBrokerCount }}
      {{- $kafkaName := include "kafka.name" . }}
      {{- range $i, $e := until ($brokers | int) }}
      {{- printf "%s-%d:9092" $kafkaName (add $i 1)  }}
      {{- if ne ($brokers | int) (add $i 1) }}
      {{- print "," }}
      {{- end }}
      {{- end }}
    {{- end -}}
{{- end -}}

{{/*
Configure Kafka Service deployment names in disaster recovery health check format.
*/}}
{{- define "kafka-service.deploymentNames" -}}
    {{- $brokers :=  include "kafka.replicas" . }}
    {{- $kafkaName := include "kafka.name" . }}
    {{- $lst := list }}
    {{- range $i, $e := until ($brokers | int) }}
        {{- $lst = append $lst (printf "deployment %s-%d" $kafkaName (add $i 1)) }}
    {{- end }}
    {{- join "," $lst }}
{{- end -}}

{{/*
Resolves Kafka bootstrap servers
*/}}
{{- define "kafka-service.bootstrapServers" -}}
  {{- if .Values.global.externalKafka.enabled -}}
    {{- .Values.global.externalKafka.bootstrapServers -}}
  {{- else -}}
    {{- printf "%s:9092" (include "kafka.name" .) -}}
  {{- end -}}
{{- end -}}

{{/*
Resolves Kafka bootstrap servers for KafkaUser Configurator
*/}}
{{- define "kafka-service.kafkaUserBootstrapServers" -}}
  {{- if .Values.global.externalKafka.enabled -}}
    {{- .Values.global.externalKafka.bootstrapServers -}}
  {{- else -}}
    {{- printf "%s.%s:9092" (include "kafka.name" .) .Release.Namespace -}}
  {{- end -}}
{{- end -}}

{{/*
Kafka SASL mechanism
*/}}
{{- define "kafka-service.saslMechanism" -}}
  {{- if .Values.global.externalKafka.enabled -}}
    {{- .Values.global.externalKafka.saslMechanism | default "SCRAM-SHA-512" -}}
  {{- else -}}
    {{- "SCRAM-SHA-512" -}}
  {{- end -}}
{{- end -}}

{{/*
Whether Kafka TLS enabled
*/}}
{{- define "kafka-service.enableTls" -}}
  {{- if .Values.global.externalKafka.enabled -}}
    {{- .Values.global.externalKafka.enableSsl -}}
  {{- else -}}
    {{- and .Values.kafka.tls.enabled .Values.global.tls.enabled -}}
  {{- end -}}
{{- end -}}

{{/*
Whether BackupDaemon TLS enabled
*/}}
{{- define "backupDaemon.enableTls" -}}
  {{- and .Values.global.tls.enabled .Values.backupDaemon.tls.enabled -}}
{{- end -}}

{{/*
Whether Disaster recovery TLS enabled
*/}}
{{- define "disasterRecovery.enableTls" -}}
  {{- and .Values.global.tls.enabled .Values.global.disasterRecovery.tls.enabled -}}
{{- end -}}

{{/*
Whether Kafka certificates are specified
*/}}
{{- define "kafka.certificatesSpecified" -}}
  {{- $filled := false -}}
  {{- range $key, $value := .Values.kafka.tls.certificates -}}
    {{- if $value -}}
        {{- $filled = true -}}
    {{- end -}}
  {{- end -}}
  {{- $filled -}}
{{ end }}

{{/*
Kafka TLS secret name
*/}}
{{- define "kafka-service.tlsSecretName" -}}
  {{- if .Values.global.externalKafka.enabled -}}
    {{- .Values.global.externalKafka.sslSecretName -}}
  {{- else -}}
    {{- if and .Values.kafka.tls.enabled .Values.global.tls.enabled -}}
      {{- if and (or .Values.global.tls.generateCerts.enabled (eq (include "kafka.certificatesSpecified" .) "true")) (not .Values.kafka.tls.secretName) -}}
        {{- printf "%s-tls-secret" (include "kafka.name" .) -}}
      {{- else -}}
        {{- required "The TLS secret name should be specified in the 'kafka.tls.secretName' parameter when the service is deployed with Kafka and TLS enabled, but without certificates generation." .Values.kafka.tls.secretName -}}
      {{- end -}}
    {{- else -}}
       {{/*
       The empty string is needed for correct prometheus rule configuration in `tls_static_metrics.yaml`
       */}}
      {{- "" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Protocol for DRD
*/}}
{{- define "disasterRecovery.protocol" -}}
{{- if eq (include "disasterRecovery.enableTls" .) "true" -}}
  {{- "https" -}}
{{- else -}}
  {{- "http" -}}
{{- end -}}
{{- end -}}

{{/*
DRD Port
*/}}
{{- define "disasterRecovery.port" -}}
  {{- if eq (include "disasterRecovery.enableTls" .) "true" -}}
    {{- "8443" -}}
  {{- else -}}
    {{- "8080" -}}
  {{- end -}}
{{- end -}}

{{/*
Whether DRD certificates are Specified
*/}}
{{- define "disasterRecovery.certificatesSpecified" -}}
  {{- $filled := false -}}
  {{- range $key, $value := .Values.global.disasterRecovery.tls.certificates -}}
    {{- if $value -}}
        {{- $filled = true -}}
    {{- end -}}
  {{- end -}}
  {{- $filled -}}
{{- end -}}

{{/*
DRD TLS secret name
*/}}
{{- define "disasterRecovery.tlsSecretName" -}}
  {{- if and (eq (include "kafka-service.enableDisasterRecovery" .) "true") .Values.global.disasterRecovery.tls.enabled .Values.global.tls.enabled -}}
    {{- if and (or .Values.global.tls.generateCerts.enabled (eq (include "disasterRecovery.certificatesSpecified" .) "true")) (not .Values.global.disasterRecovery.tls.secretName) -}}
      {{- printf "%s-drd-tls-secret" (include "kafka.name" .) -}}
    {{- else -}}
      {{- required "The TLS secret name should be specified in the 'global.disasterRecovery.tls.secretName' parameter when the service is deployed with Disaster Recovery and TLS enabled, but without certificates generation." .Values.global.disasterRecovery.tls.secretName -}}
    {{- end -}}
  {{- else -}}
    {{/*
    The empty string is needed for correct prometheus rule configuration in `tls_static_metrics.yaml`
    */}}
    {{- "" -}}
  {{- end -}}
{{- end -}}

{{/*
DRD TLS ciphers suites
*/}}
{{- define "disasterRecovery.cipherSuites" -}}
{{- join "," (coalesce .Values.global.disasterRecovery.tls.cipherSuites .Values.global.tls.cipherSuites) -}}
{{- end -}}

{{/*
DNS names used to generate TLS certificate with "Subject Alternative Name" field for DRD
*/}}
{{- define "disasterRecovery.certDnsNames" -}}
  {{- $drdNamespace := .Release.Namespace -}}
  {{- $dnsNames := list "localhost" (printf "%s-site-manager.%s" (include "kafka.name" .) $drdNamespace) (printf "%s-site-manager.%s.svc.cluster.local" (include "kafka.name" .) $drdNamespace) -}}
  {{- $dnsNames = concat $dnsNames .Values.global.disasterRecovery.tls.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}

{{/*
IP addresses used to generate TLS certificate with "Subject Alternative Name" field for DRD
*/}}
{{- define "disasterRecovery.certIpAddresses" -}}
  {{- $ipAddresses := list "127.0.0.1" -}}
  {{- $ipAddresses = concat $ipAddresses .Values.global.disasterRecovery.tls.subjectAlternativeName.additionalIpAddresses -}}
  {{- $ipAddresses | toYaml -}}
{{- end -}}

{{/*
Generate certificates for DRD
*/}}
{{- define "disasterRecovery.generateCerts" -}}
  {{- $dnsNames := include "disasterRecovery.certDnsNames" . | fromYamlArray -}}
  {{- $ipAddresses := include "disasterRecovery.certIpAddresses" . | fromYamlArray -}}
  {{- $duration := default 365 .Values.global.tls.generateCerts.durationDays | int -}}
  {{- $ca := genCA "kafka-drd-ca" $duration -}}
  {{- $drdName := "drd" -}}
  {{- $cert := genSignedCert $drdName $ipAddresses $dnsNames $duration $ca -}}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
ca.crt: {{ $ca.Cert | b64enc }}
{{- end -}}

{{/*
Service Account for Site Manager depending on smSecureAuth
*/}}
{{- define "disasterRecovery.siteManagerServiceAccount" -}}
  {{- if .Values.global.disasterRecovery.httpAuth.smServiceAccountName -}}
    {{- .Values.global.disasterRecovery.httpAuth.smServiceAccountName -}}
  {{- else -}}
    {{- if .Values.global.disasterRecovery.httpAuth.smSecureAuth -}}
      {{- "site-manager-sa" -}}
    {{- else -}}
      {{- "sm-auth-sa" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Backup Daemon Protocol
*/}}
{{- define "backupDaemon.Protocol" -}}
  {{- if and .Values.global.tls.enabled .Values.backupDaemon.tls.enabled -}}
    {{- "https" -}}
  {{- else -}}
    {{- "http" -}}
  {{- end -}}
{{- end -}}

{{/*
Backup Daemon Port
*/}}
{{- define "backupDaemon.port" -}}
  {{- if and .Values.global.tls.enabled .Values.backupDaemon.tls.enabled -}}
    {{- "8443" -}}
  {{- else -}}
    {{- "8080" -}}
  {{- end -}}
{{- end -}}

{{/*
Whether Backup Daemon certificates are Specified
*/}}
{{- define "backupDaemon.certificatesSpecified" -}}
  {{- $filled := false -}}
  {{- range $key, $value := .Values.backupDaemon.tls.certificates -}}
    {{- if $value -}}
        {{- $filled = true -}}
    {{- end -}}
  {{- end -}}
  {{- $filled -}}
{{ end }}

{{/*
Backup Daemon TLS secret name
*/}}
{{- define "backupDaemon.tlsSecretName" -}}
  {{- if and .Values.global.tls.enabled .Values.backupDaemon.tls.enabled -}}
    {{- if and (or .Values.global.tls.generateCerts.enabled (eq (include "backupDaemon.certificatesSpecified" .) "true")) (not .Values.backupDaemon.tls.secretName) -}}
      {{- printf "%s-backup-daemon-tls-secret" (include "kafka.name" .) -}}
    {{- else -}}
      {{- required "The TLS secret name should be specified in the 'backupDaemon.tls.secretNamee' parameter when the service is deployed with Backup Daemon and TLS enabled, but without certificates generation." .Values.backupDaemon.tls.secretName -}}
    {{- end -}}
  {{- else -}}
    {{/*
    The empty string is needed for correct prometheus rule configuration in `tls_static_metrics.yaml`
    */}}
    {{- "" -}}
  {{- end -}}
{{- end -}}

{{/*
Backup Daemon SSL secret name
*/}}
{{- define "backupDaemon.s3.tlsSecretName" -}}
  {{- if .Values.backupDaemon.s3.sslCert -}}
    {{- if .Values.backupDaemon.s3.sslSecretName -}}
      {{- .Values.backupDaemon.s3.sslSecretName -}}
    {{- else -}}
      {{- printf "kafka-backup-daemon-s3-tls-secret" -}}
    {{- end -}}
  {{- else -}}
    {{- if .Values.backupDaemon.s3.sslSecretName -}}
      {{- .Values.backupDaemon.s3.sslSecretName -}}
    {{- else -}}
      {{- printf "" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
DNS names used to generate TLS certificate with "Subject Alternative Name" field for Backup Daemon
*/}}
{{- define "backupDaemon.certDnsNames" -}}
  {{- $backupDaemonNamespace := .Release.Namespace -}}
  {{- $dnsNames := list "localhost" (printf "%s-backup-daemon" (include "kafka.name" .)) (printf "%s-backup-daemon.%s" (include "kafka.name" .) $backupDaemonNamespace) (printf "%s-backup-daemon.%s.svc.cluster.local" (include "kafka.name" .) $backupDaemonNamespace) -}}
  {{- $dnsNames = concat $dnsNames .Values.backupDaemon.tls.subjectAlternativeName.additionalDnsNames -}}
  {{- $dnsNames | toYaml -}}
{{- end -}}

{{/*
IP addresses used to generate TLS certificate with "Subject Alternative Name" field for Backup Daemon
*/}}
{{- define "backupDaemon.certIpAddresses" -}}
  {{- $ipAddresses := list "127.0.0.1" -}}
  {{- $ipAddresses = concat $ipAddresses .Values.backupDaemon.tls.subjectAlternativeName.additionalIpAddresses -}}
  {{- $ipAddresses | toYaml -}}
{{- end -}}

{{/*
Generate certificates for Backup Daemon
*/}}
{{- define "backupDaemon.generateCerts" -}}
  {{- $dnsNames := include "backupDaemon.certDnsNames" . | fromYamlArray -}}
  {{- $ipAddresses := include "backupDaemon.certIpAddresses" . | fromYamlArray -}}
  {{- $duration := default 365 .Values.global.tls.generateCerts.durationDays | int -}}
  {{- $ca := genCA "kafka-backup-daemon-ca" $duration -}}
  {{- $backupDaemonName := "backupDaemon" -}}
  {{- $cert := genSignedCert $backupDaemonName $ipAddresses $dnsNames $duration $ca -}}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
ca.crt: {{ $ca.Cert | b64enc }}
{{- end -}}

{{/*
Kafka admin username.
*/}}
{{- define "kafka.adminUsername" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_ADMIN_USERNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_ADMIN_USERNAME }}
  {{- else -}}
    {{- if .Values.global.externalKafka.enabled -}}
      {{- .Values.global.externalKafka.username -}}
    {{- else -}}
      {{- .Values.global.secrets.kafka.adminUsername}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Kafka admin password.
*/}}
{{- define "kafka.adminPassword" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_ADMIN_PASSWORD | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_ADMIN_PASSWORD }}
  {{- else -}}
    {{- if .Values.global.externalKafka.enabled -}}
      {{- .Values.global.externalKafka.password -}}
    {{- else -}}
      {{- .Values.global.secrets.kafka.adminPassword}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Kafka client username.
*/}}
{{- define "kafka.clientUsername" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_CLIENT_USERNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_CLIENT_USERNAME }}
  {{- else -}}
    {{- if .Values.global.externalKafka.enabled -}}
      {{- .Values.global.externalKafka.username -}}
    {{- else -}}
      {{- .Values.global.secrets.kafka.clientUsername}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Kafka client password.
*/}}
{{- define "kafka.clientPassword" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_CLIENT_PASSWORD | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_CLIENT_PASSWORD }}
  {{- else -}}
    {{- if .Values.global.externalKafka.enabled -}}
      {{- .Values.global.externalKafka.password -}}
    {{- else -}}
      {{- .Values.global.secrets.kafka.clientPassword}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Storage class from various places.
*/}}
{{- define "backupDaemon.storageClassName" -}}
  {{- if and (ne (.Values.STORAGE_RWO_CLASS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.STORAGE_RWO_CLASS | toStrings }}
  {{- else -}}
    {{- .Values.backupDaemon.storageClass -}}
  {{- end -}}
{{- end -}}

{{/*
Monitoring installation required
*/}}
{{- define "monitoring.install" -}}
  {{- if and (ne (.Values.MONITORING_ENABLED | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.MONITORING_ENABLED }}
  {{- else -}}
    {{- .Values.monitoring.install -}}
  {{- end -}}
{{- end -}}

{{/*
Whether ingress for AKHQ enabled
*/}}
{{- define "akhq.ingressEnabled" -}}
  {{- if and (ne (.Values.PRODUCTION_MODE | toString) "<nil>") .Values.global.cloudIntegrationEnabled}}
    {{- (eq .Values.PRODUCTION_MODE false) }}
  {{- else -}}
    {{- not (empty .Values.akhq.ingress.host) }}
  {{- end -}}
{{- end -}}

{{/*
Ingress host for AKHQ
*/}}
{{- define "akhq.ingressHost" -}}
  {{- if .Values.akhq.ingress.host }}
    {{- .Values.akhq.ingress.host }}
  {{- else -}}
    {{- if and (ne (.Values.SERVER_HOSTNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled }}
      {{- printf "akhq-%s.%s" .Release.Namespace .Values.SERVER_HOSTNAME }}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Whether ingress for Cruise Control enabled
*/}}
{{- define "cruise-control.ingressEnabled" -}}
  {{- if and (ne (.Values.PRODUCTION_MODE | toString) "<nil>") .Values.global.cloudIntegrationEnabled}}
    {{- (eq .Values.PRODUCTION_MODE false) }}
  {{- else -}}
    {{- not (empty .Values.cruiseControl.ingress.host) }}
  {{- end -}}
{{- end -}}

{{/*
Ingress host for Cruise Control
*/}}
{{- define "cruise-control.ingressHost" -}}
  {{- if .Values.cruiseControl.ingress.host }}
    {{- .Values.cruiseControl.ingress.host }}
  {{- else -}}
    {{- if and (ne (.Values.SERVER_HOSTNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled }}
      {{- printf "cc-%s.%s" .Release.Namespace .Values.SERVER_HOSTNAME }}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{- define "operator.image" -}}
    {{- printf "%s" .Values.operator.dockerImage -}}
{{- end -}}

{{/*
Find a Kafka image in various places.
*/}}
{{- define "kafka.image" -}}
    {{- printf "%s" .Values.kafka.dockerImage -}}
{{- end -}}

{{/*
Find a Kafka monitoring image in various places.
*/}}
{{- define "monitoring.image" -}}
    {{- printf "%s" .Values.monitoring.dockerImage -}}
{{- end -}}

{{/*
Find a Kafka Lag exporter image in various places.
*/}}
{{- define "lagExporter.image" -}}
    {{- printf "%s" .Values.monitoring.lagExporter.service.image -}}
{{- end -}}

{{/*
Find an AKHQ image in various places.
*/}}
{{- define "akhq.image" -}}
    {{- printf "%s" .Values.akhq.dockerImage -}}
{{- end -}}

{{/*
Find a Kafka Mirror Maker image in various places.
*/}}
{{- define "mirrorMaker.image" -}}
    {{- printf "%s" .Values.mirrorMaker.dockerImage -}}
{{- end -}}

{{/*
Find a Kafka Mirror Maker monitoring image in various places.
*/}}
{{- define "mirrorMakerMonitoring.image" -}}
    {{- printf "%s" .Values.mirrorMakerMonitoring.dockerImage -}}
{{- end -}}

{{/*
Find a Kafka Backup Daemon image in various places.
*/}}
{{- define "backupDaemon.image" -}}
    {{- printf "%s" .Values.backupDaemon.dockerImage -}}
{{- end -}}

{{/*
Find a kafka-integration-tests image in various places.
*/}}
{{- define "kafka-integration-tests.image" -}}
    {{- printf "%s" .Values.integrationTests.image -}}
{{- end -}}

{{- define "disasterRecovery.image" -}}
    {{- printf "%s" .Values.global.disasterRecovery.image -}}
{{- end -}}

{{/*
Calculate Heap Memory size for AKHQ
*/}}
{{- define "akhq.heapSize" -}}
  {{- if eq (include "akhq.heapSizeDefined" .) "true" -}}
    {{- .Values.akhq.heapSize -}}
  {{- else -}}
    {{- $memorySize := regexFind "\\d+" .Values.akhq.resources.requests.memory -}}
    {{- $type := regexFind "[MG]" .Values.akhq.resources.requests.memory -}}
    {{- if eq $type "M" -}}
      {{- div $memorySize 2 -}}
    {{- else if eq $type "G" -}}
      {{- mul $memorySize 512 -}}
    {{- else -}}
      {{- "null" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
TLS Static Metric secret template
Arguments:
Dictionary with:
* "namespace" is a namespace of application
* "application" is name of application
* "service" is a name of service
* "enableTls" is tls enabled for service
* "secret" is a name of tls secret for service
* "certProvider" is a type of tls certificates provider
* "certificate" is a name of CertManager's Certificate resource for service
Usage example:
{{template "global.tlsStaticMetric" (dict "namespace" .Release.Namespace "application" .Chart.Name "service" .global.name "enableTls" (include "global.enableTls" .) "secret" (include "global.tlsSecretName" .) "certProvider" (include "services.certProvider" .) "certificate" (printf "%s-tls-certificate" (include "global.name")) }}
*/}}
{{- define "global.tlsStaticMetric" -}}
- expr: {{ ternary "1" "0" (eq .enableTls "true") }}
  labels:
    namespace: "{{ .namespace }}"
    application: "{{ .application }}"
    service: "{{ .service }}"
    {{ if eq .enableTls "true" }}
    secret: "{{ .secret }}"
    {{ if eq .certProvider "cert-manager" }}
    certificate: "{{ .certificate }}"
    {{ end }}
    {{ end }}
  record: service:tls_status:info
{{- end -}}


{{- define "cruise-control.replicationFactor" -}}
{{- if .Values.global.externalKafka.replicas -}}
    {{- if gt .Values.global.externalKafka.replicas 3.0 -}}
        {{- "3" -}}
    {{- else -}}
        {{- (default 2 .Values.global.externalKafka.replicas ) | toString -}}
    {{- end -}}
{{- else -}}
    {{- if gt (int (include "kafka.replicas" . )) 3 -}}
        {{- "3" -}}
    {{- else -}}
        {{- (default 2 (include "kafka.replicas" . )) | toString -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
kafka replicas.
*/}}
{{- define "kafka.replicas" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_REPLICAS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_REPLICAS }}
  {{- else -}}
    {{- default 3 .Values.kafka.replicas -}}
  {{- end -}}
{{- end -}}


{{/*
Calculate Kafka Broker disk size for Cruise Control config
*/}}
{{- define "cruise-control.brokerDiskSpace" -}}
  {{- if .Values.cruiseControl.capacity.diskSpace -}}
    {{- .Values.cruiseControl.capacity.diskSpace -}}
  {{- else -}}
    {{- $memorySize := regexFind "\\d+" .Values.kafka.storage.size -}}
    {{- $type := regexFind "[MG]" .Values.kafka.storage.size -}}
    {{- if eq $type "M" -}}
      {{- $memorySize -}}
    {{- else if eq $type "G" -}}
      {{- mul $memorySize 1024 -}}
    {{- else -}}
      {{- "" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Find a Kafka Cruise Control image in various places.
*/}}
{{- define "cruise-control.image" -}}
    {{- printf "%s" .Values.cruiseControl.dockerImage -}}
{{- end -}}

{{/*
Configure replicas number for backup-daemon pod
*/}}
{{- define "backupDaemon.replicas" -}}
{{- and (ne .Values.global.disasterRecovery.mode "standby") (ne .Values.global.disasterRecovery.mode "disabled") | ternary 1 0 -}}
{{- end -}}

{{ define "find_image" }}
  {{- $root := index . 0 -}}
  {{- $service_name := index . 1 -}}
  {{- if index $root.Values.deployDescriptor $service_name }}
  {{- index $root.Values.deployDescriptor $service_name "image" }}
  {{- else }}
  {{- "not_found" }}
  {{- end }}
{{- end }}

{{- define "kafka.monitoredImages" -}}
  {{- printf "deployment %s-service-operator kafka-service-operator %s, " (include "kafka.name" .) (include "find_image" (list . "kafka-service")) -}}
  {{- if .Values.monitoring.install }}
    {{- printf "deployment %s-monitoring kafka-monitoring %s, " (include "kafka.name" .) (include "find_image" (list . "kafka-monitoring")) -}}
      {{- if .Values.monitoring.lagExporter.enabled }}
        {{- printf "deployment %s-monitoring kafka-lag-exporter %s, " (include "kafka.name" .) (include "find_image" (list . "prod.platform.streaming_docker-kafka-lag-exporter")) -}}
      {{- end -}}
  {{- end -}}
  {{- if .Values.backupDaemon.install }}
    {{- printf "deployment %s-backup-daemon kafka-backup-daemon %s, " (include "kafka.name" .) (include "find_image" (list . "prod.platform.streaming_kafka-backup-daemon")) -}}
  {{- end -}}
  {{- if .Values.integrationTests.install }}
    {{- printf "deployment %s-integration-tests-runner kafka-integration-tests-runner %s, " (include "kafka.name" .) (include "find_image" (list . "kafka-integration-tests")) -}}
  {{- end -}}
  {{- if and .Values.mirrorMaker.install .Values.mirrorMaker.regionName }}
    {{- printf "deployment %s-%s-mirror-maker kafka-mirror-maker %s, " (.Values.mirrorMaker.regionName) (include "kafka.name" .) (include "find_image" (list . "docker-kafka-mirror-maker")) -}}
  {{- end -}}
  {{- if .Values.mirrorMakerMonitoring.install }}
    {{- printf "deployment %s-kafka-mirror-maker-monitoring kafka-kafka-mirror-maker-monitoring %s, " (include "kafka.name" .) (include "find_image" (list . "prod.platform.streaming_kafka-kafka-mirror-maker-monitoring")) -}}
  {{- end -}}
  {{- if .Values.akhq.install }}
    {{- printf "deployment akhq akhq %s, " (include "find_image" (list . "docker-akhq")) -}}
  {{- end -}}
  {{- if .Values.cruiseControl.install }}
    {{- printf "deployment %s-cruise-control kafka-cruise-control %s, " (include "kafka.name" .) (include "find_image" (list . "prod.platform.streaming_docker-cruise-control")) -}}
  {{- end -}}
{{- end -}}

{{- define "mirrorMaker.tasksMaxDefined" -}}
{{- if .Values.mirrorMaker.tasksMax }}
  {{ ge (int .Values.mirrorMaker.tasksMax) -1 }}
{{- else -}}
  {{ "false" }}
{{- end -}}
{{- end -}}

{{- define "backupDaemon.persistentVolumeDefined" -}}
  {{- if eq (.Values.backupDaemon.persistentVolume | toString) "<nil>" -}}
    {{- "false" }}
  {{- else -}}
    {{- and (ne .Values.backupDaemon.persistentVolume "") (ne .Values.backupDaemon.persistentVolume "null") -}}
  {{- end -}}
{{- end -}}

{{- define "backupDaemon.storageClassDefined" -}}
  {{- if eq (.Values.backupDaemon.storageClass | toString) "<nil>" -}}
    {{- "false" }}
  {{- else -}}
    {{- and (ne .Values.backupDaemon.storageClass "") (ne .Values.backupDaemon.storageClass "null") -}}
  {{- end -}}
{{- end -}}

{{- define "lagExporter.clusterLabelsDefined" -}}
  {{- if eq (.Values.monitoring.lagExporter.cluster.labels | toString) "<nil>" -}}
    {{- "false" }}
  {{- else -}}
    {{- gt (len .Values.monitoring.lagExporter.cluster.labels) 0 -}}
  {{- end -}}
{{- end -}}

{{- define "akhq.heapSizeDefined" -}}
  {{- if eq (.Values.akhq.heapSize | toString) "<nil>" -}}
    {{- "false" }}
  {{- else -}}
    {{- gt (int .Values.akhq.heapSize) 0 -}}
  {{- end -}}
{{- end -}}

{{/*
Find an CRD Init job Docker image in various places.
*/}}
{{- define "crd-init.image" -}}
    {{- printf "%s" .Values.crdInit.dockerImage -}}
{{- end -}}

{{- define "crd-init.crdsToCreate" -}}
  {{- $names := "" -}}
  {{- if .Values.operator.kmmConfiguratorEnabled -}}
    {{- $names = printf "%s%s" $names "kmm_crd.yaml" -}}
  {{- end -}}
  {{- if .Values.operator.akhqConfigurator.enabled -}}
    {{- $names = printf "%s,%s" $names "akhq_autocollect_crd.yaml" -}}
  {{- end -}}
  {{- if .Values.operator.kafkaUserConfigurator.enabled -}}
    {{- $names = printf "%s,%s" $names "kafkauser_crd.yaml" -}}
  {{- end -}}
  {{- printf "%s" $names | trimPrefix "," -}}
{{- end -}}