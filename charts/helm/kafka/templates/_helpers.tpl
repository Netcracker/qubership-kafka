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
Common Kafka operator chart related resources labels
*/}}
{{- define "kafka.defaultLabels" -}}
app.kubernetes.io/version: '{{ .Values.ARTIFACT_DESCRIPTOR_VERSION | trunc 63 | trimAll "-_." }}'
app.kubernetes.io/component: 'backend'
app.kubernetes.io/part-of: '{{ .Values.PART_OF }}'
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
Configure Kafka service name
*/}}
{{- define "kafka.name" -}}
{{- coalesce .Values.global.name "kafka" -}}
{{- end -}}

{{/*
Configure Kafka service account
*/}}
{{- define "kafka.serviceAccount" -}}
{{- coalesce .Values.operator.serviceAccount (printf "%s-operator" (include "kafka.name" .)) -}}
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
Compute the maximum number of unavailable replicas for the PodDisruptionBudget. This defaults to 1.
Add a special case for replicas=1, where it should default to 0 as well.
*/}}
{{- define "kafka.pdb.maxUnavailable" -}}
{{- if eq (int (include "kafka.replicas" . )) 1 -}}
{{ 0 }}
{{- else if .Values.kafka.disruptionBudget.maxUnavailable -}}
{{ .Values.kafka.disruptionBudget.maxUnavailable -}}
{{- else -}}
{{- 1 -}}
{{- end -}}
{{- end -}}

{{/*
DNS names used to generate TLS certificate with "Subject Alternative Name" field
*/}}
{{- define "kafka.certDnsNames" -}}
  {{- $kafkaName := include "kafka.name" . -}}
  {{- $dnsNames := list "localhost" $kafkaName (printf "%s.%s" $kafkaName .Release.Namespace) (printf "%s.%s" $kafkaName "kafka-broker") (printf "%s.%s.%s" $kafkaName "kafka-broker" .Release.Namespace) (printf "%s.%s.svc" $kafkaName .Release.Namespace) (printf "%s-kraft-controller" $kafkaName) -}}
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
Create the name for service registration in Consul
*/}}
{{- define "kafka-service.registeredServiceName" -}}
    {{ printf "%s-%s" (include "kafka.name" .) .Release.Namespace }}
{{- end -}}

{{/*
Create list of Kafka brokers separated by ",".
For instance, "kafka-1:9092,kafka-2:9092,kafka-3:9092"
*/}}
{{- define "kafka-service.brokersList" -}}
    {{- $brokers := .Values.monitoring.kafkaTotalBrokerCount }}
    {{- $kafkaName := include "kafka.name" . }}
    {{- range $i, $e := until ($brokers | int) }}
    {{- printf "%s-%d:9092" $kafkaName (add $i 1)  }}
    {{- if ne ($brokers | int) (add $i 1) }}
    {{- print "," }}
    {{- end }}
    {{- end }}
{{- end -}}

{{/*
Configure Kafka Service deployment names in disaster recovery health check format.
*/}}
{{- define "kafka-service.deploymentNames" -}}
    {{- $brokers := include "kafka.replicas" . }}
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
  {{- printf "%s:9092" (include "kafka.name" .) -}}
{{- end -}}

{{/*
Kafka SASL mechanism
*/}}
{{- define "kafka-service.saslMechanism" -}}
  {{- "SCRAM-SHA-512" -}}
{{- end -}}

{{/*
Whether Kafka TLS enabled
*/}}
{{- define "kafka-service.enableTls" -}}
  {{- and .Values.kafka.tls.enabled .Values.global.tls.enabled -}}
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
  {{- if and .Values.kafka.tls.enabled .Values.global.tls.enabled -}}
    {{- if and (or .Values.global.tls.generateCerts.enabled (eq (include "kafka.certificatesSpecified" .) "true")) (not .Values.kafka.tls.secretName) -}}
      {{- printf "%s-tls-secret" (include "kafka.name" .) -}}
    {{- else -}}
      {{- .Values.kafka.tls.secretName -}}
    {{- end -}}
  {{- else -}}
    {{- "" -}}
  {{- end -}}
{{- end -}}

{{/*
Kafka admin username.
*/}}
{{- define "kafka.adminUsername" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_ADMIN_USERNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_ADMIN_USERNAME }}
  {{- else -}}
    {{- .Values.global.secrets.kafka.adminUsername -}}
  {{- end -}}
{{- end -}}

{{/*
Kafka admin password.
*/}}
{{- define "kafka.adminPassword" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_ADMIN_PASSWORD | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_ADMIN_PASSWORD }}
  {{- else -}}
    {{- .Values.global.secrets.kafka.adminPassword -}}
  {{- end -}}
{{- end -}}

{{/*
Kafka client username.
*/}}
{{- define "kafka.clientUsername" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_CLIENT_USERNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_CLIENT_USERNAME }}
  {{- else -}}
    {{- .Values.global.secrets.kafka.clientUsername -}}
  {{- end -}}
{{- end -}}

{{/*
Kafka client password.
*/}}
{{- define "kafka.clientPassword" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_CLIENT_PASSWORD | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_CLIENT_PASSWORD }}
  {{- else -}}
    {{- .Values.global.secrets.kafka.clientPassword -}}
  {{- end -}}
{{- end -}}

{{/*
ZooKeeper client username.
*/}}
{{- define "kafka.zooKeeperClientUsername" -}}
  {{- if and (ne (.Values.INFRA_ZOOKEEPER_CLIENT_USERNAME | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_ZOOKEEPER_CLIENT_USERNAME }}
  {{- else -}}
    {{- .Values.global.secrets.kafka.zookeeperClientUsername -}}
  {{- end -}}
{{- end -}}

{{/*
ZooKeeper client password.
*/}}
{{- define "kafka.zooKeeperClientPassword" -}}
  {{- if and (ne (.Values.INFRA_ZOOKEEPER_CLIENT_PASSWORD | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_ZOOKEEPER_CLIENT_PASSWORD }}
  {{- else -}}
    {{- .Values.global.secrets.kafka.zookeeperClientPassword -}}
  {{- end -}}
{{- end -}}

{{/*
ZooKeeper address.
*/}}
{{- define "kafka.zooKeeperConnect" -}}
  {{- if and (ne (.Values.INFRA_KAFKA_ZOOKEEPER_ADDRESS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.INFRA_KAFKA_ZOOKEEPER_ADDRESS }}
  {{- else -}}
    {{- .Values.kafka.zookeeperConnect -}}
  {{- end -}}
{{- end -}}

{{/*
Storage class from various places.
*/}}
{{- define "kafka.storageClassName" -}}
  {{- if and (ne (.Values.STORAGE_RWO_CLASS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.STORAGE_RWO_CLASS | toStrings }}
  {{- else -}}
    {{- default "" .Values.kafka.storage.className -}}
  {{- end -}}
{{- end -}}

{{/*
Storage class from various places.
*/}}
{{- define "kafka.migrationController.storageClassName" -}}
  {{- if and (ne (.Values.STORAGE_RWO_CLASS | toString) "<nil>") .Values.global.cloudIntegrationEnabled -}}
    {{- .Values.STORAGE_RWO_CLASS | toStrings }}
  {{- else -}}
    {{- if .Values.kafka.migrationController.storage.className -}}
        {{- .Values.kafka.migrationController.storage.className -}}
    {{- else -}}
        {{- default "" .Values.kafka.storage.className -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Find a Kafka service operator image in various places.
Image can be found from:
* SaaS/App deployer (or groovy.deploy.v3) from .Values.deployDescriptor "kafka-service" "image"
* DP.Deployer from .Values.deployDescriptor.kafkaOperator.image
* or from default values .Values.operator.dockerImage
*/}}
{{- define "operator.image" -}}
    {{- printf "%s" .Values.operator.dockerImage -}}
{{- end -}}

{{/*
Find a Kafka image in various places.
*/}}
{{- define "kafka.image" -}}
    {{- printf "%s" .Values.kafka.dockerImage -}}
{{- end -}}