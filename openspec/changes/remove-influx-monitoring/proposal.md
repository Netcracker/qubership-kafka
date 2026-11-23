# Proposal

## Why

InfluxDB is no longer a supported metrics destination. Kafka and Mirror Maker monitoring already default to Prometheus, and the InfluxDB Telegraf output, System Monitoring database settings, and demo stack are unused. The Helm values schema is shared across installations and across the `kafka` and `kafka-service` charts, so deleting `influxdb` from the enum or removing `smDb*` properties would reject existing values files on `helm template` / install.

## What Changes

- Stop producing an InfluxDB metrics path. Telegraf configuration MUST NOT contain `outputs.influxdb`. Monitoring and Mirror Maker monitoring MUST NOT receive `SM_DB_HOST` / `SM_DB_NAME` or mount `sm-db-username` / `sm-db-password`. The Influx-only monitoring secret MUST NOT be rendered.
- Always render the Prometheus path (Prometheus client output, ServiceMonitor, Grafana dashboards, PrometheusRule) whenever monitoring is installed, including when `monitoringType` is `influxdb`.
- Keep the shared schema valid for current values files. Both charts' `values.schema.json` MUST keep `monitoringType` enum `prometheus` | `influxdb` and MUST keep `smDbHost`, `smDbName`, `smDbUsername`, and `smDbPassword`. A values file that still sets `monitoringType: influxdb` and the `smDb*` keys MUST pass schema validation.
- Keep CR fields `monitoringType`, `smDbHost`, and `smDbName` on Kafka monitoring and Mirror Maker monitoring so existing `KafkaService` objects still apply. The operator MUST NOT use those fields to select an InfluxDB output. No new API version.
- **BREAKING** for any installation that still consumes the InfluxDB push: metrics are no longer written to InfluxDB. Install and schema validation do not fail; Prometheus output is used instead.
- Update user docs and the local `demo/monitoring` stack so they no longer present InfluxDB as a configured destination.
- Leave Telegraf line protocol (`data_format = "influx"`) in place. That format is how exec plugins emit samples and is not the InfluxDB output.
- Zabbix templates, tests, and Helm parameters stay unchanged.

## Capabilities

### New Capabilities

- `monitoring/prometheus-output`: Kafka and Mirror Maker monitoring expose metrics for Prometheus. Legacy InfluxDB Helm and CR settings remain accepted and do not change the output.

### Modified Capabilities

<!-- No existing spec covers monitoring output. `kafka/native-config` and `mirror-maker/native-config` are unchanged. -->

## Impact

- `operator/charts/helm/kafka-service/templates/` — monitoring and mirror-maker monitoring ConfigMaps, secrets, ServiceMonitors, dashboards, PrometheusRule, and the KafkaService CR template. Influx branches go away; objects that were skipped for `influxdb` are rendered for every installed monitoring stack.
- `operator/charts/helm/kafka/values.schema.json` and `operator/charts/helm/kafka-service/values.schema.json` — enum and `smDb*` properties stay. Descriptions stop saying InfluxDB is required.
- `operator/charts/helm/kafka/values.yaml` and `operator/charts/helm/kafka-service/values.yaml` — defaults stay `prometheus`. Existing `smDb*` keys stay so shared values files keep a place to set them.
- `operator/controllers/provider/monitoring_provider.go` and `mirror_maker_monitoring_provider.go` — drop the InfluxDB env vars and secret projection.
- `operator/api/v7/kafkaservice_types.go` and generated CRDs — fields stay. Mark them deprecated in comments. Do not bump the API version. Run `make generate` and `make manifests` only if comments or markers require regeneration.
- `docs/public/installation.md`, `docs/public/monitoring.md`, `docs/public/security.md`, `docker-kafka/README.md`, `monitoring/README.md` — remove InfluxDB as a supported destination. Port `8086` in the security port table goes away.
- `demo/monitoring/` — the InfluxDB 1.8 service and Grafana datasource are not part of the supported path.
