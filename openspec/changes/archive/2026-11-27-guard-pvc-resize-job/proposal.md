# Proposal

## Why

An external resize job replaces Kafka and backup-daemon PVCs with a larger size and records the previous size in the `pvc-changed-by-job-from` annotation. If the deploy parameter is still the old size, a later reconcile or Helm upgrade rolls the cluster against that stale parameter while the volume already has the new size.

## What Changes

- Before any other Kafka reconcile work, fail when a managed PVC has `pvc-changed-by-job-from` and its requested storage does not match the size in the Kafka CR. Do not requeue that failure.
- When the Kafka operator later updates that PVC and the sizes match, remove the annotation.
- Fail a `kafka-service` Helm upgrade when the backup-daemon PVC has the annotation and `backupDaemon.storage` does not match the PVC request.
- When the backup-daemon reconciler runs and the annotation is present, remove it. Do not add a CRD field for the backup-daemon size.

## Capabilities

### New Capabilities

- `kafka/pvc-resize-job`: The Kafka operator refuses to reconcile while a resize-job annotation still matches a stale deploy size, and clears the annotation once the PVC is updated with the matching size.
- `backup-daemon/pvc-resize`: The kafka-service chart blocks an upgrade while the backup-daemon PVC annotation disagrees with `backupDaemon.storage`, and the backup-daemon reconciler removes the annotation when it is present.

### Modified Capabilities

## Impact

- Kafka operator reconcile (`operator/controllers/kafka`) and shared PVC update (`operator/controllers/reconciler.go`).
- kafka-service Helm template `backupDaemon.resizeJobCheck`, included from the backup-daemon PVC template.
- Backup-daemon reconciler in the kafka-service operator. No CRD or API version change.
