# Spec

## Purpose

Stops a Kafka reconcile from rolling brokers while a resize job has already changed a PVC and the deploy size in the Kafka CR is still the previous size.

## Requirements

### Requirement: Stale resize annotation fails Kafka reconcile

Before creating or updating any other Kafka resource, the operator MUST reject the reconcile when a managed PVC carries the annotation `pvc-changed-by-job-from` and that PVC's requested storage is not equal to the deploy size. The rejection MUST be recorded as a failed status and MUST NOT be retried by the controller work queue.

#### Scenario: Broker PVC is larger than the CR size

- **WHEN** broker PVC `pvc-<kafka-name>-<id>` has annotation `pvc-changed-by-job-from` and its storage request is `2Gi` while `spec.storage.size` is `1Gi`
- **THEN** reconcile fails before services, secrets, or deployments are changed, and the annotation remains on the PVC

#### Scenario: Equivalent quantities are accepted

- **WHEN** the PVC storage request is `2Gi` and the deploy size is `2048Mi`
- **THEN** the sizes are treated as equal and this check does not fail the reconcile

#### Scenario: Missing PVC or missing annotation

- **WHEN** the managed PVC does not exist, or it exists without `pvc-changed-by-job-from`
- **THEN** this check does not fail the reconcile

### Requirement: Managed PVCs covered by the check

The check MUST cover each broker PVC `pvc-<kafka-name>-<broker-id>` for broker ids `1` through `spec.replicas`, using `spec.storage.size`, and the migration-controller PVC `pvc-<kafka-name>-kraft-controller`, using `spec.migrationController.storage.size`.

#### Scenario: Kraft controller PVC is checked separately

- **WHEN** `pvc-<kafka-name>-kraft-controller` has `pvc-changed-by-job-from` and its storage request differs from `spec.migrationController.storage.size`
- **THEN** reconcile fails the same way as for a broker PVC

### Requirement: Annotation is removed when the PVC is updated

After the size check has passed, an update of an existing managed Kafka PVC MUST remove `pvc-changed-by-job-from`. A PVC that does not have the annotation MUST still be updated.

#### Scenario: Matching size clears the annotation

- **WHEN** the deploy size equals the PVC storage request and the operator updates that PVC
- **THEN** `pvc-changed-by-job-from` is no longer present on the PVC
