# Tasks

## 1. Kafka reconcile guard

- [x] 1.1 Before any other Kafka reconcile work, compare each broker PVC `pvc-<name>-<id>` with `spec.storage.size` and `pvc-<name>-kraft-controller` with `spec.migrationController.storage.size` when `pvc-changed-by-job-from` is present. Verify `go test ./controllers/ -run TestCheckPvcResizeJob` accepts `2Gi`/`2048Mi`, rejects a smaller deploy size, and ignores a missing PVC.
- [x] 1.2 On mismatch, write the failed status and return a terminal reconcile error so the work queue does not retry. Verify a mismatch returns before services or deployments are updated.
- [x] 1.3 When an existing Kafka PVC is updated after the check passes, delete `pvc-changed-by-job-from`. Verify `go test ./controllers/ -run TestUpdatePersistentVolumeClaimRemovesJobAnnotation` shows the annotation gone and other annotations kept.

## 2. Backup daemon

- [x] 2.1 Add `backupDaemon.resizeJobCheck` and include it from the backup-daemon PVC template so Helm fails when the annotation is present and the PVC request string differs from `backupDaemon.storage`. Verify `helm template` of `operator/charts/helm/kafka-service` still renders `data-kafka-backup-daemon` when no cluster lookup is available.
- [x] 2.2 At the start of the backup-daemon reconciler, delete `pvc-changed-by-job-from` only when that annotation is present on `data-<name>-backup-daemon`. Verify a PVC without the annotation is not updated, and do not add a CRD field.
