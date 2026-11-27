# Spec Delta

## Purpose

Keeps a kafka-service Helm upgrade from proceeding while the backup-daemon volume still records a resize job that the deploy parameter has not caught up with, and clears that annotation once the backup-daemon reconciler runs.

## ADDED Requirements

### Requirement: Helm upgrade fails on a stale backup-daemon resize annotation

When backup daemon is installed, rendering the kafka-service chart MUST fail if PVC `data-<kafka-name>-backup-daemon` has annotation `pvc-changed-by-job-from` and its storage request string is not equal to `backupDaemon.storage` (default `1Gi`). The failure MUST abort the release before resources are applied.

#### Scenario: Storage parameter still differs from the PVC

- **WHEN** the backup-daemon PVC request is `2Gi`, the annotation is present, and `backupDaemon.storage` is `1Gi`
- **THEN** Helm upgrade fails and does not apply the release

#### Scenario: Storage parameter matches the PVC

- **WHEN** the annotation is present and `backupDaemon.storage` equals the PVC storage request
- **THEN** this check does not fail the release

#### Scenario: No annotation or no PVC

- **WHEN** the PVC does not exist, or it exists without `pvc-changed-by-job-from`
- **THEN** this check does not fail the release

### Requirement: Reconciler removes the annotation only when it is present

The backup-daemon reconciler MUST remove `pvc-changed-by-job-from` from PVC `data-<kafka-service-name>-backup-daemon` when that annotation is present. It MUST NOT update the PVC when the annotation is absent, and a missing PVC MUST NOT fail the reconciler. This removal MUST NOT require a new CRD field.

#### Scenario: Annotation is deleted

- **WHEN** the backup-daemon PVC has annotation `pvc-changed-by-job-from`
- **THEN** the reconciler deletes that annotation and updates the PVC

#### Scenario: Annotation is absent

- **WHEN** the backup-daemon PVC has no `pvc-changed-by-job-from` annotation
- **THEN** the reconciler does not update the PVC for this reason
