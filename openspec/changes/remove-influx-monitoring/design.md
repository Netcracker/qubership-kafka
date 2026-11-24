# Design

## Context

See proposal.md for why InfluxDB output is leaving. Spec: `specs/monitoring/prometheus-output/spec.md`.

Today `monitoring.type` and `mirrorMakerMonitoring.type` in `operator/charts/helm/kafka-service/templates/_helpers.tpl` return `monitoring.monitoringType`, then `global.monitoringType`, then `prometheus`. Both charts' `values.schema.json` allow `prometheus` and `influxdb` on `global`, `monitoring`, and `mirrorMakerMonitoring`, plus `smDbHost`, `smDbName`, `smDbUsername`, and `smDbPassword`. The same values file is applied to both charts, so a tighter enum or a removed property fails install for every installation that still has those keys.

`kafka-service` templates switch on that type. `influxdb` renders `outputs.influxdb`, Influx-only processors and inputs, `smDb*` on the KafkaService CR, and `kafka-monitoring-secret.yaml`. It skips ServiceMonitor, Grafana dashboards, and PrometheusRule. The operator adds `SM_DB_HOST` / `SM_DB_NAME` and projects `sm-db-username` / `sm-db-password` only when `spec.MonitoringType == "influxdb"` (`monitoring_provider.go`, `mirror_maker_monitoring_provider.go`). CRD fields are plain strings with no enum.

`data_format = "influx"` on exec inputs is Telegraf line protocol. It is inside the shared input config, not the InfluxDB output branch.

Several templates compare `.Values.global.monitoringType` directly instead of the helper: integration-test env and secret (`ZABBIX_URL` and `CLOUD_DB` only for `influxdb`, `PROMETHEUS_URL` only for `prometheus`), the integration-test secret checksum, and `tls_static_metrics.yaml`.

The `kafka` chart carries the same schema and default `monitoringType: prometheus`. It does not render the monitoring ConfigMap.

## Goals / Non-Goals

**Goals:**

- One Prometheus render path, including when values or an already-stored CR say `influxdb`.
- Schema validation of existing shared values files stays green without editing those files.
- Operator ignores InfluxDB fields on objects that have not been upgraded yet.

**Non-Goals:**

- Deleting Zabbix robot tests, `ZabbixLibrary.py`, `monitoring/zabbix/`, or `integrationTests.zabbixUrl` / `global.secrets.integrationTests.zabbix` from values and schema.
- Removing `data_format = "influx"` or rewriting exec plugins.
- A new API version, a CRD enum, or pruning `monitoringType` / `smDbHost` / `smDbName` from the Go types.
- Turning `influxdb` into a hard schema error.

## Decisions

### Decision 1: Leave the shared schema shape alone

Keep the `prometheus` | `influxdb` enum and the `smDb*` properties in both `values.schema.json` files. Keep the keys in both `values.yaml` files. Change only description text that says InfluxDB is required or is a supported destination. Say the keys are accepted and ignored.

**Why over deleting the enum value:** Helm validates the shared values file against the schema. Dropping `influxdb` or `smDbHost` rejects installations that still set them. `additionalProperties: true` does not save an enum: `monitoringType: influxdb` fails as soon as `influxdb` is not in the enum.

**Alternative:** Accept any string and drop the enum. Rejected. It is a wider schema change than this removal needs, and it is easy for the two chart schemas to drift.

### Decision 2: Delete Influx branches; do not keep them behind a dead condition

In `kafka-service` templates, remove each `eq ... "influxdb"` branch and keep the Prometheus branch that is its `else`. That covers:

- `outputs.influxdb` and the Influx-only processors and inputs in `kafka-monitoring-configuration.yaml` and `kafka-mirror-maker-monitoring-configuration.yaml`
- `smDbHost` / `smDbName` in `cr.yaml`
- `kafka-monitoring-secret.yaml` (the whole template is Influx-only; leave the file rendering nothing, or delete the file)
- `sm-db-*` keys in `kafka-mirror-maker-monitoring-secret.yaml`
- `ne ... "influxdb"` guards on ServiceMonitor, dashboards, PrometheusRule, and `tls_static_metrics.yaml`

Do not implement this by making `monitoring.type` still return `influxdb` while the branches are gone and also do not leave the branches in place "just in case". A helper that can still return `influxdb` with the branches deleted would skip Prometheus objects wherever a `ne influxdb` guard remains.

`cr.yaml` writes `monitoringType: prometheus` for every install, including when the values say `influxdb`. The stored string stops claiming an output the pod does not have. Fields stay on the CRD so an object that still contains `smDbHost` decodes.

**Alternative:** Make the helpers always return `prometheus` and leave the `if` blocks. Rejected. The Influx output text would still be in the chart, and any template that compares the raw values string (integration tests, `tls_static_metrics.yaml`) would keep the old branch.

### Decision 3: Operator ignores the fields; CRD schema is unchanged

Delete the `MonitoringType == "influxdb"` blocks that append `SM_DB_*` and the secret projection in both providers. Pod env and volumes are the Prometheus ones for every monitoring type.

Mark `MonitoringType`, `SmDbHost`, and `SmDbName` on `Monitoring` and `MirrorMakerMonitoring` as deprecated in the Go field comment, same idea as `VaultSecretManagement`: retained so existing objects decode. Do not add a kubebuilder enum. An enum of `prometheus` only would make the API reject a KafkaService that still has `monitoringType: influxdb`.

Run `make generate` / `make manifests` only if the field comment is copied into the CRD description. If the CRD yaml changes, the only expected diff is description text.

### Decision 4: Integration-test wiring follows Prometheus whenever monitoring is installed

Conditions that today check `global.monitoringType == "prometheus"` for `PROMETHEUS_URL`, the test secret projection, and the pod checksum treat "monitoring installed" as enough. The block that sets `ZABBIX_URL` and `CLOUD_DB` only when `global.monitoringType` is `influxdb` is removed, including the `zabbix-user` / `zabbix-password` projection in the integration-test secret.

`CLOUD_DB` is `smDbName`. Leaving that block would keep publishing the Influx database name into the test pod for every shared values file that still says `influxdb`. Zabbix parameters stay in values and schema so those files still validate; the chart no longer reads them. The robot suite files are not part of this change.

### Decision 5: Demo stack drops the InfluxDB service

`demo/monitoring/docker-compose.yml` runs InfluxDB 1.8 and points `kafka-monitoring` at it with `SM_DB_*`. Remove that service, those env vars, and the Grafana Influx datasource. Leave the monitoring container on the Prometheus client port `8096`. Do not rewrite product Grafana dashboards into the demo if they are Influx queries; the demo no longer claims an Influx destination.

## Risks / Trade-offs

- [Shared values still say `influxdb`, so a reader thinks Influx is active] → Schema descriptions and `docs/public/installation.md` state that the value is accepted and ignored and that the output is Prometheus. `helm template` does not fail.
- [An installation that still scrapes InfluxDB loses that feed on upgrade] → This is the intended break. Rollback is the previous chart. No CR edit is required to roll back, because the fields are still in the CRD.
- [Helm upgrade rewrites `monitoringType` on the CR from `influxdb` to `prometheus`] → Intended. Operator already ignores `influxdb` if the CR is applied before the chart rewrite.
- [Deleting a whole `if influxdb` / `else` by mistake drops the Prometheus output or `data_format = "influx"`] → Delete only the Influx branch. Keep the `else` body and every `data_format = "influx"` on exec inputs. `helm template` with `monitoringType: influxdb` must show `outputs.prometheus_client` and must not show `outputs.influxdb`.
- [Two schemas drift] → Edit `kafka` and `kafka-service` `values.schema.json` in the same change. Descriptions may change; enum members and property names may not.

## Migration Plan

1. Upgrade the chart. Existing values files that set `monitoringType: influxdb` and `smDb*` need no edit.
2. Confirm the monitoring pod listens on `8096` and has no `SM_DB_*` env. InfluxDB stops receiving metrics. Prometheus scrape, ServiceMonitor, and rules appear even if the values type is still `influxdb`.
3. Rollback by deploying the previous chart. Stored `smDb*` fields remain valid for that chart.

## Open Questions

None. Schema compatibility is fixed by Decision 1. Zabbix source files stay; only the Influx-gated test env is removed.
