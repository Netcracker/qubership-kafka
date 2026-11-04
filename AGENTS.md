# AGENTS.md

Qubership Kafka is an enterprise Kafka distribution for Kubernetes/OpenShift.
It bundles a Kubebuilder-based operator, two Helm charts, Apache Kafka
container images, several auxiliary services (Go and Python), Mirror Maker 2
packaging, monitoring assets, and a Robot Framework integration test suite.

## Repository Layout

Monorepo. `go.mod` files live under `operator/` and `monitoring/`. The
`backup-daemon` and `crd-init` services are Python.

- `/operator` — Kubebuilder v3 operator (`projectName: kafka-service-operator`,
  domain `netcracker.com`). Single binary, two run modes selected by env
  `OPERATOR_MODE`:
  - `kafka` — manages the Kafka cluster CR.
  - `kafkaservice` — manages supplementary service CRs.
- `/operator/charts/helm/kafka` — Helm chart that installs the operator (in
  `kafka` mode) and a `Kafka` CR (`netcracker.com/v1`).
- `/operator/charts/helm/kafka-service` — Helm chart that installs the operator
  (in `kafkaservice` mode) and a `KafkaService` CR (`netcracker.com/v7`).
- `/backup-daemon` — Python REST service for topic and ACL backup/restore.
  Stores snapshots on a Persistent Volume (path configured in
  `backup-daemon.conf`). Image is built on `qubership-backup-daemon-go`.
- `/monitoring` — Go entrypoint wrapping Telegraf; ships Grafana dashboards
  and Prometheus alert rules.
- `/mirror-maker`, `/mirror-maker-monitoring` — KIP-382 Mirror Maker 2
  packaging plus a Java extension (`mirror-maker/kafka-mm2-extension`).
- `/lag-exporter` — Prometheus exporter for Kafka consumer lag.
- `/docker-kafka` — Custom Apache Kafka images with SASL, JMX/Prometheus
  exporter and health checks. Subdirs `3/` and `4/` build Kafka 3.x and 4.x.
- `/docker-akhq` — AKHQ Kafka UI image.
- `/docker-cruise-control` — Cruise Control image (rebalance / self-healing).
- `/docker-transfer` — `FROM scratch` image used to ship charts and docs.
- `/crd-init` — Python job that installs CRDs.
- `/integration-tests` — Robot Framework suites at `robot/tests/`.
- `/demo` — Docker Compose stack (Zookeeper + Kafka + integration-tests web
  terminal on `localhost:8090`).
- `/docs/public` — User-facing documentation (architecture, installation,
  troubleshooting, security, etc.).
- `/Makefile` — Drives operator codegen (`make generate`, `make manifests`).

## Operator Code Organization

- `operator/api/v1` — `Kafka`, `KafkaUser`, `KmmConfig`, `AkhqConfig` types.
- `operator/api/v7` — `KafkaService` type.
- `operator/api/kmm` — KMM transformation types (no CRD).
- `operator/controllers/<feature>/` — Reconciliation logic, split into
  `<feature>_controller.go`, `<feature>_reconciler.go`, plus `condition.go` and
  `status_update.go`. The `kafkaservice` package owns reconcilers for AKHQ,
  backup daemon, monitoring, mirror maker, integration tests, etc.
- `operator/controllers/provider/` — Pure builders that construct Kubernetes
  objects (Deployment, Service, ConfigMap, …). Keep manifest construction
  here, not in the reconcilers.
- `operator/workers/` — Worker pool launching per-CR jobs (`KafkaJob`,
  `AkhqJob`, `KmmJob`, `KafkaUserJob`). Each job is started once per
  configured API group (`API_GROUP` plus optional `SECONDARY_API_GROUP`).
- `operator/cfg/cfg.go` — All operator flags and env vars.

## Compatibility Rules

All changes MUST be backward compatible.

### CRDs

- Add new functionality through new optional fields. Do not change the
  semantics or shape of existing fields.
- If a breaking change is unavoidable, stop and ask the user for explicit
  approval before adding a new API version.
- After editing anything under `operator/api/`, run `make generate` and
  `make manifests`. Use the dev-kit if `operator-sdk` is not installed
  locally (see below).

### Helm Charts

- Every new parameter MUST have a sensible default; never break existing
  values files.
- The `kafka` and `kafka-service` charts share parameter names where they
  overlap. Do not introduce conflicting names between the two charts.
- Reuse shared templates in `templates/_helpers.tpl` for labels,
  annotations, images and security contexts.
- Every changes must be tested with `helm template`.

## Dev-Kit (Operator Code Generation)

Use the dev-kit when `operator-sdk` is not installed locally. It runs
`operator-sdk:v1.28` in Docker with the project mounted.

```sh
cd operator/dev-kit
./terminal.sh        # Builds and enters the operator-sdk container
make generate        # Regenerate DeepCopy code from operator/api/
make manifests       # Regenerate CRDs in config/crd/bases
```

## Integration Tests

Robot Framework suites at `integration-tests/robot/tests/`. The `demo/`
directory provides a Docker Compose stack for local testing.

### Mandatory – every change must be tested

- Any change in docker folders where Dockerfile are placed must be tested with Docker build.
- Any code change must ve covered with unit tests.
- Any fynctional feature of components that can be tested without redeploy must be covered with robot framework tests.
- Before submitting any change to `backup-daemon` or `integration-tests` folders, run **at minimum** the `kafka_crud`, `kafka_consumer_producer` and `backup` suites.

```sh
cd demo
docker-compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
docker-compose exec integration-tests robot -i kafka_crudORkafka_consumer_producerORbackup ./tests
docker-compose down
```

See [demo/README.md](demo/README.md) for the full environment reference
(build contexts, environment variables, stopping the stack).
