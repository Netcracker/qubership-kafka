---
description: >-
  Diagnostic playbook for Kafka broker, disk, memory, lag, ZooKeeper, and
  KRaft migration failures. Never invoke automatically — only when explicitly
  requested by the user.
applyTo: "**/*"
---

When diagnosing a Kafka problem — a degraded or down cluster, CPU, memory, or
lag pressure, a full data volume, a ZooKeeper connection failure, a KRaft
migration failure, runAsNonRoot, or Permission denied on the data directory —
apply the `troubleshoot-kafka` skill.
