{{- define "defaultAlerts" -}}
    {{ .Release.Namespace }}-{{ .Release.Name }}:
      rules:
        KafkaIsDegradedAlert:
          annotations:
            description: 'Kafka is Degraded'
            summary: Some of Kafka Service pods are down
          expr: kafka_cluster_status{namespace="{{ .Release.Namespace }}",container="{{ template "kafka.name" . }}-monitoring"} == 6
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        KafkaMetricsAreAbsent:
          annotations:
            description: 'Kafka metrics are absent on {{ .Release.Namespace }}.'
            summary: Kafka metrics are absent
          expr: absent(kafka_cluster_status{namespace="{{ .Release.Namespace }}"}) == 1
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        KafkaIsDownAlert:
          annotations:
            description: 'Kafka is Down'
            summary: All of Kafka Service pods are down
          expr: kafka_cluster_status{namespace="{{ .Release.Namespace }}",container="{{ template "kafka.name" . }}-monitoring"} == 10
          for: 3m
          labels:
            severity: critical
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        KafkaCPUUsageAlert:
          annotations:
            description: 'Kafka CPU usage is higher than 95 percents'
            summary: Some of Kafka Service pods load CPU higher then 95 percents
          expr: max(rate(container_cpu_usage_seconds_total{namespace="{{ .Release.Namespace }}",pod=~"{{ template "kafka.name" . }}-[0-9].*",container="kafka"}[5m])) / max(kube_pod_container_resource_limits_cpu_cores{exported_namespace="{{ .Release.Namespace }}",exported_pod=~"{{ template "kafka.name" . }}-[0-9].*"}) > 0.95
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        KafkaMemoryUsageAlert:
          annotations:
            description: 'Kafka memory usage is higher than 95 percents'
            summary: Some of Kafka Service pods use memory higher then 95 percents
          expr: max(container_memory_working_set_bytes{namespace="{{ .Release.Namespace }}",pod=~"{{ template "kafka.name" . }}-[0-9].*",container="kafka"}) / max(kube_pod_container_resource_limits_memory_bytes{exported_namespace="{{ .Release.Namespace }}",exported_pod=~"{{ template "kafka.name" . }}-[0-9].*"}) > 0.95
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        KafkaHeapMemoryUsageAlert:
          annotations:
            description: 'Kafka heap memory usage is higher than 95 percents'
            summary: Some of Kafka Service pods use heap memory higher then 95 percents
          expr: max(java_Memory_HeapMemoryUsage_used{namespace="{{ .Release.Namespace }}",broker=~"{{ template "kafka.name" . }}-[0-9].*"}) / max(java_Memory_HeapMemoryUsage_max{namespace="{{ .Release.Namespace }}", broker=~"{{ template "kafka.name" . }}-[0-9].*"}) > 0.95
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        KafkaGCCountAlert:
          annotations:
            description: 'Some of Kafka Service pods have Garbage collections count rate higher than {{ .Values.thresholds.gcCountAlert }}'
            summary: Some of Kafka Service pods have Garbage collections count rate higher than {{ .Values.thresholds.gcCountAlert }}
          expr: max(rate(java_GarbageCollector_CollectionCount_total{namespace="{{ .Release.Namespace }}", broker=~"{{ template "kafka.name" . }}-[0-9].*"}[5m])) > {{ .Values.thresholds.gcCountAlert }}
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        KafkaLagAlert:
          annotations:
            description: 'Some of Kafka Service pods have partition lag higher than {{ .Values.thresholds.lagAlert }}'
            summary: Some of Kafka Service pods have partition lag higher than {{ .Values.thresholds.lagAlert }}
          expr: max(kafka_consumergroup_group_lag{namespace="{{ .Release.Namespace }}"}) > {{ .Values.thresholds.lagAlert }}
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        {{- if .Values.thresholds.partitionCountAlert }}
        KafkaPartitionCountAlert:
          annotations:
            description: 'Kafka Partition count for {{`{{ $labels.broker }}`}} broker is higher than {{ .Values.thresholds.partitionCountAlert }}'
            summary: Some of Kafka Partition count is higher than {{ .Values.thresholds.partitionCountAlert }}
          expr: kafka_server_ReplicaManager_Value{name="PartitionCount", namespace="{{ .Release.Namespace }}", broker=~"{{ template "kafka.name" . }}-[0-9].*"} > {{ .Values.thresholds.partitionCountAlert }}
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        {{- end }}
        {{- if .Values.thresholds.brokerSkewAlert }}
        KafkaBrokerSkewAlert:
          annotations:
            description: 'Kafka Broker Skew for {{`{{ $labels.broker }}`}} broker is higher than {{ .Values.thresholds.brokerSkewAlert }} percent'
            summary: Some of Kafka Broker Skew is higher than {{ .Values.thresholds.brokerSkewAlert }} percent
          expr: (kafka_broker_skew{namespace="{{ .Release.Namespace }}", container="{{ template "kafka.name" . }}-monitoring", broker=~"{{ template "kafka.name" . }}-[0-9].*"} > {{ .Values.thresholds.brokerSkewAlert }}) and on(broker, namespace) (kafka_server_ReplicaManager_Value{name="PartitionCount", namespace="{{ .Release.Namespace }}",  broker=~"{{ template "kafka.name" . }}-[0-9].*"} > 3 )
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        {{- end }}
        {{- if .Values.thresholds.brokerLeaderSkewAlert }}
        KafkaBrokerLeaderSkewAlert:
          annotations:
            description: 'Kafka Broker Leader Skew for {{`{{ $labels.broker }}`}} broker is higher than {{ .Values.thresholds.brokerLeaderSkewAlert }} percent'
            summary: Some of Kafka Broker Leader Skew is higher than {{ .Values.thresholds.brokerLeaderSkewAlert }} percent
          expr: (kafka_broker_leader_skew{namespace="{{ .Release.Namespace }}", container="{{ template "kafka.name" . }}-monitoring", broker=~"{{ template "kafka.name" . }}-[0-9].*"} > {{ .Values.thresholds.brokerLeaderSkewAlert }}) and on(broker, namespace) (kafka_server_ReplicaManager_Value{name="PartitionCount", namespace="{{ .Release.Namespace }}",  broker=~"{{ template "kafka.name" . }}-[0-9].*"} > 3 )
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
        {{- end }}
        SupplementaryServicesCompatibilityAlert:
          annotations:
            description: 'Kafka supplementary services in namespace {{`{{ $labels.namespace }}`}} is not compatible with Kafka version {{`{{ $labels.application_version }}`}}'
            summary: 'Kafka supplementary services in namespace {{`{{ $labels.namespace }}`}} is not compatible with Kafka version {{`{{ $labels.application_version }}`}}, allowed range is {{`{{ $labels.min_version }}`}} - {{`{{ $labels.max_version }}`}}'
          expr: supplementary_services_version_compatible{application="kafka", namespace="{{ .Release.Namespace }}"} != 1
          for: 3m
          labels:
            severity: warning
            namespace: {{ .Release.Namespace }}
            service: {{ .Release.Name }}
{{- end }}


