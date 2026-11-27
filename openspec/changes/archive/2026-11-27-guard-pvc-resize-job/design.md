# Design

## Context

See proposal.md for why the resize job must not be followed by a reconcile or Helm upgrade that still carries the old size. Kafka storage already lives on the Kafka CR (`spec.storage.size` and `spec.migrationController.storage.size`). Backup-daemon storage lives only in the kafka-service chart value `backupDaemon.storage`; the KafkaService CR has no storage field, and the CRD is not being extended.

## Goals / Non-Goals

**Goals:**

- Stop Kafka reconcile before any other resource change when the annotation and the CR size disagree.
- Stop the kafka-service Helm release in that same situation for the backup-daemon PVC.
- Remove the annotation after the matching Kafka PVC update, and from the backup-daemon PVC when the reconciler sees it.

**Non-Goals:**

- A new CRD field or API version.
- A Helm hook Job to strip the annotation.
- Changing ordinary Kafka volume expansion that has no `pvc-changed-by-job-from` annotation.

## Decisions

### Kafka check runs first and is not retried

The Kafka reconciler compares each managed PVC request with the CR size before hashing the spec or writing in-progress status. A mismatch returns a terminal reconcile error, so the work queue does not schedule another attempt. The next attempt happens when the CR changes, which a Helm upgrade does because the Kafka CR carries a changing reconcile trigger.

Alternative considered: return a normal error. That requeues on a backoff forever until the CR changes, which is the behavior we are avoiding.

### Sizes are compared as Kubernetes quantities

`1Gi` and `1024Mi` are the same size. A missing PVC is ignored. An invalid deploy size string fails the check.

### Kafka annotation removal stays on the existing PVC update

The early check already returned when the sizes differ, so the later update of an existing PVC only runs when they match or the annotation is absent. That update deletes `pvc-changed-by-job-from`. `delete` on a nil annotation map is a no-op, so PVCs without the annotation are unchanged by that line.

### Backup-daemon check stays in the chart

Helm `lookup` reads the live PVC during upgrade. `backupDaemon.resizeJobCheck` fails the render when the annotation exists and the request string differs from `backupDaemon.storage`. `helm template` without a cluster gets an empty lookup, so local rendering still succeeds.

Alternative considered: put the size on the KafkaService CR and check it in the operator. That requires a CRD change, which is out of scope.

### Backup-daemon annotation is removed only when present

The backup-daemon reconciler, at the start of its reconcile, loads `data-<name>-backup-daemon`. If the annotation is missing, it returns without an update. If it is present, it deletes the key and updates the PVC. It does not compare sizes; the chart already did that before the upgrade was applied.

## Risks / Trade-offs

- [Backup-daemon reconciler can clear the annotation on any later reconcile, including one that happens before the Helm upgrade] → The chart is the gate for the upgrade. An operator reconcile that runs while the annotation is still the only record of a stale size will remove that record.
- [Helm compares the storage strings, not Kubernetes quantities] → `1Gi` and `1024Mi` fail the chart check even though the operator would treat them as equal.
- [A failed Kafka status write still updates the CR status] → A status update can enqueue one extra reconcile. That pass hits the same terminal error and does not continue into resource updates.

## Migration Plan

Deploy the Kafka operator and the kafka-service chart together. Existing clusters without the annotation are unchanged. Rollback is reverting the operator and the chart; PVCs that already lost the annotation stay without it.

## Open Questions
