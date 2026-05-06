#!/usr/bin/env bash
# Security hardening smoke tests for the kafka Helm chart.
# Usage: bash tests/security_hardening_test.sh
# Requires: helm (>=3), grep

set -euo pipefail

CHART_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0
FAIL=0

check() {
  local desc="$1"
  local count="$2"
  if [[ "$count" -gt 0 ]]; then
    echo "  PASS: $desc ($count occurrences)"
    PASS=$(( PASS + 1 ))
  else
    echo "  FAIL: $desc (0 occurrences — expected ≥1)"
    FAIL=$(( FAIL + 1 ))
  fi
}

echo "=== kafka chart security hardening smoke tests ==="

RENDERED=$(helm template kafka "$CHART_DIR" \
  --set kafka.install=true \
  --set 'kafka.zookeeperConnect=zk:2181' \
  --set PAAS_PLATFORM=KUBERNETES \
  --set groupMigration.enabled=true \
  2>&1)

check "readOnlyRootFilesystem: true" "$(echo "$RENDERED" | grep -c 'readOnlyRootFilesystem: true' || true)"
check "runAsNonRoot: true"           "$(echo "$RENDERED" | grep -c 'runAsNonRoot: true' || true)"
check "allowPrivilegeEscalation: false" "$(echo "$RENDERED" | grep -c 'allowPrivilegeEscalation: false' || true)"
check "seccompProfile RuntimeDefault" "$(echo "$RENDERED" | grep -c 'type: RuntimeDefault' || true)"
check "capabilities drop ALL"        "$(echo "$RENDERED" | grep -c '"ALL"' || true)"
check "runAsUser: 1000 (KUBERNETES)" "$(echo "$RENDERED" | grep -c 'runAsUser: 1000' || true)"
check "/tmp emptyDir volume"         "$(echo "$RENDERED" | grep -c 'name: tmp' || true)"

echo ""
echo "Results: $PASS passed, $FAIL failed"
[[ "$FAIL" -eq 0 ]]
