# Spec

## Purpose

Allows MirrorMaker 2 (MM2) configuration properties to be specified using their native Kafka Connect
/ MM2 names (e.g. `refresh.topics.interval.seconds`) without requiring the encoded `CONF_*`
environment variable form.

## Requirements

### Requirement: Native MirrorMaker config map field

The KafkaService CRD spec and the `kafka-service` Helm chart SHALL expose a `config` field on the
`mirrorMaker` section that accepts a map of native MirrorMaker property names to string values.
Keys SHALL be preserved exactly as specified, with no case conversion, no prefix, and no
underscore/dot substitution.

#### Scenario: Common properties appear in mm2.properties under original names

- **WHEN** `mirrorMaker.config` is set with entries such as
  `refresh.topics.interval.seconds: "30"` and `sync.group.offsets.enabled: "true"`
- **THEN** `mm2.properties` inside the running MirrorMaker container SHALL contain
  `refresh.topics.interval.seconds = 30` and `sync.group.offsets.enabled = true` under
  those exact property names

#### Scenario: Property names containing arrows or special characters are preserved

- **WHEN** `mirrorMaker.config` contains a key such as `"source->target.sync.group.offsets.enabled": "false"`
- **THEN** `mm2.properties` SHALL contain `source->target.sync.group.offsets.enabled = false`
  exactly as specified

#### Scenario: Empty or absent config map has no effect

- **WHEN** `mirrorMaker.config` is absent or set to an empty map
- **THEN** the generated `mm2.properties` SHALL be identical to what would have been produced
  using only the existing `environmentVariables` mechanism

### Requirement: Precedence of environmentVariables over config

When the same MirrorMaker property is set through both `mirrorMaker.config` and
`mirrorMaker.environmentVariables`, the value from `environmentVariables` SHALL take effect and
the value from `config` SHALL be ignored.

#### Scenario: environmentVariables wins over config for the same property

- **WHEN** `mirrorMaker.config` contains `refresh.topics.interval.seconds: "30"` and
  `mirrorMaker.environmentVariables` contains `CONF_REFRESH_TOPICS_INTERVAL_SECONDS=60`
- **THEN** `mm2.properties` SHALL contain `refresh.topics.interval.seconds = 60`

#### Scenario: Non-conflicting entries from both mechanisms are both applied

- **WHEN** `mirrorMaker.config` contains `refresh.topics.interval.seconds: "30"` and
  `mirrorMaker.environmentVariables` contains `CONF_TASKS_MAX_PER_DATA_PLANE=10`
- **THEN** `mm2.properties` SHALL contain both entries

### Requirement: Invalid entries do not corrupt the properties file

The system SHALL not corrupt `mm2.properties` or block other configuration entries when a
`mirrorMaker.config` entry is invalid. The invalid entry SHALL be logged and skipped.

#### Scenario: Invalid entry is skipped; valid entries still applied

- **WHEN** `mirrorMaker.config` contains one invalid key and one valid key such as
  `refresh.topics.interval.seconds: "30"`
- **THEN** `mm2.properties` SHALL contain the valid entry and the invalid entry SHALL be reported
  in the operator or container logs without stopping the deployment

### Requirement: Backward compatibility

Deployments that do not set `mirrorMaker.config` SHALL continue to operate exactly as before with
no change to generated `mm2.properties` or to pod specifications.

#### Scenario: Existing deployment without mirrorMaker.config is unaffected

- **WHEN** a KafkaService deployment is upgraded via Helm and `mirrorMaker.config` is not set
- **THEN** the operator SHALL not modify the MirrorMaker pod template in any way related to native
  config, and `mm2.properties` SHALL be identical to what the previous release produced

### Requirement: No manual steps beyond Helm upgrade

Adding, changing, or removing entries in `mirrorMaker.config` SHALL take effect through the normal
Kafka-service Helm upgrade and MirrorMaker pod rollout without any additional operator action.

#### Scenario: Config change triggers pod rollout

- **WHEN** `mirrorMaker.config` is updated in the Helm values and the chart is upgraded
- **THEN** the MirrorMaker pods SHALL be rolled out with the updated `mm2.properties` reflecting
  the new config entries
