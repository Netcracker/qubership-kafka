# Tasks

## 1. CRD API field additions

- [x] 1.1 Add `Config map[string]string \`json:"config,omitempty"\`` field to the `Kafka` spec struct
  in `operator/api/v1/kafka_types.go` (alongside the existing `EnvironmentVariables` field) and
  verify the file compiles with `go build ./operator/...`
- [x] 1.2 Add `Config map[string]string \`json:"config,omitempty"\`` field to the `MirrorMaker`
  struct in `operator/api/v7/kafkaservice_types.go` and verify the file compiles
- [x] 1.3 Run `make generate` from `operator/` (or via dev-kit `./terminal.sh` + `make generate`)
  and verify `operator/api/v1/zz_generated.deepcopy.go` and
  `operator/api/v7/zz_generated.deepcopy.go` are updated with `DeepCopyInto` changes for the new
  map fields
- [x] 1.4 Run `make manifests` and verify `operator/config/crd/bases/qubership.org_kafkas.yaml`
  and `qubership.org_kafkaservices.yaml` include the new `config` field
- [x] 1.5 Copy the updated CRD YAML files into
  `operator/charts/helm/kafka/crds/crd.yaml` and
  `operator/charts/helm/kafka-service/crds/crd.yaml` and verify `helm template` succeeds for both
  charts

## 2. Operator encoding logic

- [x] 2.1 Add helper function `configMapToEnvVars(config map[string]string, envPrefix string,
  existingEnvVars []string, logger logr.Logger) []corev1.EnvVar` in
  `operator/controllers/provider/kafkaservice_provider.go` that:
  (a) for each key, converts it to `<envPrefix><UPPER_SNAKE>` (uppercase, replace `.` with `_`);
  (b) skips any entry whose encoded name already appears in `existingEnvVars` (logs a warning);
  (c) returns the resulting slice.
  Verify with a unit test in `operator/controllers/provider/` covering the encoding rule, conflict
  detection, and empty/nil map inputs
- [x] 2.2 Update `kafka_provider.go` to call `configMapToEnvVars(krp.spec.Config, "CONF_KAFKA_", ...)` and
  prepend the returned vars before the `environmentVariables`-derived vars in both
  `createDeploymentContainers` call sites (lines ~540 and ~732). Verify via unit test that:
  (a) `kafka.config` entries appear as `CONF_KAFKA_*` env vars on the pod;
  (b) when the same property is in both `config` and `environmentVariables`, the
  `environmentVariables` value is used
- [x] 2.3 Update `mirror_maker_provider.go` to call
  `configMapToEnvVars(spec.Config, "CONF_", ...)` and prepend returned vars before
  `environmentVariables`-derived vars when building the MirrorMaker pod. Verify via unit test the
  same two behaviors as 2.2

## 3. Helm chart changes — kafka chart

- [x] 3.1 Add `kafka.config: {}` (commented example showing native property names) to
  `operator/charts/helm/kafka/values.yaml` near the existing `environmentVariables` entry and verify
  `helm template operator/charts/helm/kafka` renders without error
- [x] 3.2 Add `"config"` property to the `kafka` object definition in
  `operator/charts/helm/kafka/values.schema.json` (following the `cruiseControl.config` pattern at
  line 1789 — `type: object`, `additionalProperties` with scalar types) and verify
  `helm lint operator/charts/helm/kafka` passes
- [x] 3.3 Update `operator/charts/helm/kafka/templates/cr.yaml` to pass `kafka.config` to the CR
  spec: add a block `{{- if .Values.kafka.config }} config: {{- toYaml .Values.kafka.config | nindent 4 }} {{- end }}`
  after the `environmentVariables` block and verify `helm template` output includes the `config`
  field when values contain it

## 4. Helm chart changes — kafka-service chart

- [x] 4.1 Add `mirrorMaker.config: {}` (commented example) to
  `operator/charts/helm/kafka-service/values.yaml` near the `mirrorMaker.environmentVariables`
  comment and verify `helm template operator/charts/helm/kafka-service` renders without error
- [x] 4.2 Add `"config"` property to the `mirrorMaker` object definition in
  `operator/charts/helm/kafka-service/values.schema.json` (same scalar-valued additionalProperties
  pattern) and verify `helm lint operator/charts/helm/kafka-service` passes
- [x] 4.3 Update `operator/charts/helm/kafka-service/templates/cr.yaml` to pass
  `mirrorMaker.config` to the CR spec inside the `{{- if .Values.mirrorMaker.install }}` block and
  verify `helm template` output includes the `config` field when values contain it

## 5. Integration tests (Robot Framework)

- [x] 5.1 Add a Robot Framework test in `integration-tests/robot/tests/` covering the `kafka.config`
  happy path: set at least two native Kafka properties via `kafka.config`, deploy or upgrade, and
  verify those properties appear in `server.properties` on the running pod with correct values.
  Tag the test `kafka_crud` or an appropriate existing tag so it runs in the mandatory suite
- [x] 5.2 Add a Robot Framework test verifying the `environmentVariables` precedence rule for
  `kafka.config`: set a property via both mechanisms and verify the `environmentVariables` value wins
- [x] 5.3 Add a Robot Framework test for `mirrorMaker.config` happy path: set at least two native
  MirrorMaker properties, deploy or upgrade, and verify they appear in `mm2.properties` with correct
  values (if MirrorMaker is part of the test environment)
- [x] 5.4 Run the mandatory integration test suites
  (`kafka_crud`, `kafka_consumer_producer`, `backup`) using the demo Docker Compose stack and verify
  all pass with no regressions:
  `cd demo && docker-compose -f docker-compose.yml -f docker-compose.build.yml up -d --build &&
  docker-compose exec integration-tests robot -i kafka_crudORkafka_consumer_producerORbackup ./tests`
  (skipped per user decision — changes are additive, new tests are skip-guarded, CI covers this)

## 6. Documentation

- [x] 6.1 Add a documentation section to `docs/public/` (or update an existing configuration page)
  covering `kafka.config` and `mirrorMaker.config`: include a description of each parameter,
  YAML examples with native property names, the precedence rule over `environmentVariables`, and
  backward compatibility guarantees. Verify the docs render correctly (no broken links, valid
  Markdown)
