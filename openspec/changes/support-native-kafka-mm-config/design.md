# Design

## Context

See proposal.md – Why for motivation.

**Current configuration flow (Kafka):**

1. User sets `kafka.environmentVariables: ["CONF_KAFKA_NUM_IO_THREADS=16"]` in Helm values.
2. Helm renders this into the Kafka CRD spec (`spec.environmentVariables`).
3. The operator's `buildEnvs()` function (`operator/controllers/provider/kafkaservice_provider.go:82`)
   parses `KEY=VALUE` strings and creates `corev1.EnvVar` objects on the pod.
4. Inside the Kafka container, `docker-kafka/3/docker-entrypoint.sh:816–828` iterates all `CONF_KAFKA_*`
   env vars, strips the prefix, lowercases, replaces `_` with `.`, then writes/updates the property
   in `server.properties`.

**Current configuration flow (MirrorMaker):**

1. `mirrorMaker.environmentVariables` strings are processed the same way by `buildEnvs()`.
2. `mirror-maker/docker/docker-entrypoint.sh:193–241` iterates `CONF_*`, `<CLUSTER>_CONF_*`, and
   `<SOURCE>_<TARGET>_CONF_*` env vars with the same prefix-strip + lowercase + `_`→`.` transform
   and writes entries to `mm2.properties`.
3. A separate `KMM_CONF_INJECT` file path already allows a ConfigMap-mounted file to seed
   `mm2.properties` before env-var processing begins.

**Key constraint:** Native Kafka property names contain `.` (dot), which is illegal in POSIX
environment variable names. Therefore the value cannot be passed as an env var with the native key
as the name.

## Goals / Non-Goals

**Goals:**
- Allow users to specify `kafka.config` and `mirrorMaker.config` as native-name maps.
- Translate them into the existing `CONF_KAFKA_*` / `CONF_*` env-var layer internally so Docker
  image changes are minimal.
- Give `environmentVariables` explicit precedence over `config` for conflicting properties.
- Zero impact when neither `config` field is set.

**Non-Goals:**
- Scoped MirrorMaker config keys (`<CLUSTER>_CONF_*`, `<SOURCE>_<TARGET>_CONF_*`): the new
  `mirrorMaker.config` field covers only common (unscoped) properties. Scoped config remains via
  `environmentVariables`.
- Changing the `CONF_KAFKA_*` env-var convention or the Docker entrypoint's existing loop.
- Validating property names against a known Kafka property set.

## Decisions

### Decision 1: Operator-side encoding — translate `config` to `CONF_KAFKA_*` env vars

**Chosen:** The operator encodes each `kafka.config` entry into the equivalent `CONF_KAFKA_*` env
var (`num.io.threads` → uppercase + replace `.` with `_` + `CONF_KAFKA_` prefix), adds it to the
pod env via `buildEnvs()`, and lets the existing entrypoint loop handle it unchanged.

**Why over ConfigMap/volume injection:**
- Requires no changes to Docker images — the existing entrypoint loop already handles the result.
- Follows the same path as `environmentVariables`, making behavior consistent and observable
  (env vars are visible in pod describe).
- The MirrorMaker `KMM_CONF_INJECT` mechanism already exists but only seeds the file; env-var
  encoding keeps MirrorMaker on the same code path as Kafka.

**Precedence implementation:** `config`-derived env vars are added first in `buildEnvs()` before
`environmentVariables`-derived vars. Because Kubernetes pod env vars must be unique, the operator
MUST detect conflicts and drop any `config`-derived var whose encoded name already appears in
`environmentVariables`. This gives `environmentVariables` explicit precedence without relying on
ordering.

**Limitation:** Property names that are not round-trip-safe through the encoding
(e.g. a name containing `_` where another name differs only by having `.`) could silently collide.
In practice, Apache Kafka and MirrorMaker property names exclusively use `.` as separator, so
this is not a realistic concern. A collision is logged and the `environmentVariables` value wins.

**Alternative considered: ConfigMap/volume mount with entrypoint reading a properties file**
Rejected because it requires coordinated changes to every Docker image (kafka 3, kafka 4,
mirror-maker) and adds a new code path in three entrypoint scripts. The value is low given that
operator-side encoding achieves the same user-visible result.

### Decision 2: New `Config map[string]string` field on existing CRD structs

Add `Config map[string]string` (JSON: `"config"`) to:
- `operator/api/v1/kafka_types.go` → `Kafka` struct
- `operator/api/v7/kafkaservice_types.go` → `MirrorMaker` struct

Both fields are optional (omitempty). This is purely additive and backward compatible.

**Why not a `[]string` like `environmentVariables`:**
A map type enforces unique keys at the API level, makes it obvious that duplicate keys are rejected,
and allows per-key schema validation in the Helm chart JSON schema.

### Decision 3: Helm schema uses `additionalProperties` with scalar-value types

The `kafka.config` and `mirrorMaker.config` fields use:

```json
{
  "type": "object",
  "additionalProperties": {
    "oneOf": [
      { "type": "string" },
      { "type": "integer" },
      { "type": "number" },
      { "type": "boolean" }
    ]
  }
}
```

Values are coerced to string when passed to the CRD. This matches the pattern used for
`cruiseControl.config` in the existing `kafka` chart schema (line 1789 of `values.schema.json`).

### Decision 4: MirrorMaker encoding — `CONF_` prefix (common scope only)

`mirrorMaker.config` entries encode as `CONF_<UPPER_SNAKE>` env vars (same rule as Kafka but with
`CONF_` instead of `CONF_KAFKA_`). The entrypoint's existing `^CONF_` loop writes them to
`mm2.properties` as common (unscoped) properties. Scoped properties (per-cluster, per-flow) remain
`environmentVariables`-only for this change.

## Risks / Trade-offs

- **Dot-underscore ambiguity in property names** → Mitigation: log a warning when the encoded form
  of a `config` key collides with an explicit `environmentVariables` name; the latter always wins.
- **env var name length limit (Linux: 256 bytes)** → Very long property names could exceed the
  limit after encoding. In practice Kafka property names are well under this limit. No mitigation
  needed.
- **`CONF_CONFIG_PROVIDERS` conflict** → The MirrorMaker entrypoint already sets
  `CONF_CONFIG_PROVIDERS=secret`. If a user puts `config.providers: x` in `mirrorMaker.config`
  it would encode to `CONF_CONFIG_PROVIDERS`. The operator MUST treat any explicit
  `environmentVariables` entry as winning, which handles this case.
- **`make generate` / `make manifests` must run after API changes** → The dev-kit instruction in
  `AGENTS.md` covers this; it is a task-level concern, not a design risk.

## Migration Plan

- This is a purely additive change. No migration is required.
- Rollback: remove `kafka.config` / `mirrorMaker.config` from values; the operator omits the
  encoded env vars and pod template is unchanged.
- Deployments using only `environmentVariables` are unaffected (no new fields, no pod changes).
