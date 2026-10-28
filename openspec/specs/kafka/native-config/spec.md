# Spec

## Purpose

Allows Kafka broker configuration properties to be specified using their native Apache Kafka names
(e.g. `num.io.threads`) rather than requiring the encoded `CONF_KAFKA_*` environment variable form.

## Requirements

### Requirement: Native Kafka config map field

The Kafka CRD spec and the `kafka` Helm chart SHALL expose a `config` field that accepts a map of
native Kafka broker property names to string values. Keys SHALL be preserved exactly as specified,
with no case conversion, no prefix, and no underscore/dot substitution.

#### Scenario: Properties appear in server.properties under original names

- **WHEN** `kafka.config` is set with entries such as `num.io.threads: "16"` and
  `group.initial.rebalance.delay.ms: "3000"`
- **THEN** `server.properties` inside the running Kafka container SHALL contain
  `num.io.threads=16` and `group.initial.rebalance.delay.ms=3000` under those exact property names

#### Scenario: Empty or absent config map has no effect

- **WHEN** `kafka.config` is absent or set to an empty map
- **THEN** the generated `server.properties` SHALL be identical to what would have been produced
  using only the existing `environmentVariables` mechanism

### Requirement: Precedence of environmentVariables over config

When the same Kafka property is set through both `kafka.config` and `kafka.environmentVariables`,
the value from `environmentVariables` SHALL take effect and the value from `config` SHALL be ignored.

#### Scenario: environmentVariables wins over config for the same property

- **WHEN** `kafka.config` contains `num.io.threads: "8"` and `kafka.environmentVariables` contains
  `CONF_KAFKA_NUM_IO_THREADS=32`
- **THEN** `server.properties` SHALL contain `num.io.threads=32`

#### Scenario: Non-conflicting entries from both mechanisms are both applied

- **WHEN** `kafka.config` contains `group.initial.rebalance.delay.ms: "3000"` and
  `kafka.environmentVariables` contains `CONF_KAFKA_NUM_IO_THREADS=16`
- **THEN** `server.properties` SHALL contain both `group.initial.rebalance.delay.ms=3000` and
  `num.io.threads=16`

### Requirement: Invalid entries do not corrupt the properties file

The system SHALL not corrupt `server.properties` or block other configuration entries when a
`kafka.config` entry is invalid (e.g. an empty key or a key that cannot be used as a valid property
name). The invalid entry SHALL be logged and skipped.

#### Scenario: Invalid entry is skipped; valid entries still applied

- **WHEN** `kafka.config` contains one invalid key and one valid key such as `num.io.threads: "16"`
- **THEN** `server.properties` SHALL contain `num.io.threads=16` and the invalid entry SHALL be
  reported in the operator or container logs without stopping the deployment

### Requirement: Backward compatibility

Deployments that do not set `kafka.config` SHALL continue to operate exactly as before with no
change to generated `server.properties` or to pod specifications.

#### Scenario: Existing deployment without kafka.config is unaffected

- **WHEN** a Kafka deployment is upgraded via Helm and `kafka.config` is not set in values
- **THEN** the operator SHALL not modify the pod template in any way related to native config, and
  `server.properties` SHALL be identical to what the previous release produced

### Requirement: No manual steps beyond Helm upgrade

Adding, changing, or removing entries in `kafka.config` SHALL take effect through the normal Kafka
Helm upgrade and pod rollout without any additional operator action.

#### Scenario: Config change triggers pod rollout

- **WHEN** `kafka.config` is updated in the Helm values and the chart is upgraded
- **THEN** the Kafka pods SHALL be rolled out with the updated `server.properties` reflecting the
  new config entries
