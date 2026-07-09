#!/usr/bin/env bash
# Smoke tests for the Fantasy FRC home lab.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Waiting for core workloads to be ready..."
kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx --timeout=120s || true
kubectl rollout status statefulset/postgres -n database --timeout=120s || true
kubectl rollout status statefulset/redis -n fantasy-frc --timeout=120s || true
kubectl rollout status deployment/fantasy-frc-web -n fantasy-frc --timeout=120s || true

echo "==> Checking pod status..."
kubectl get pods -n fantasy-frc
kubectl get pods -n database

echo "==> Checking migration job..."
kubectl wait --for=condition=complete job/fantasy-frc-migrate -n fantasy-frc --timeout=10s || true

echo "==> Checking application health..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -k -H "Host: fantasy-frc.local" "https://192.168.1.164:30556/")
if [[ "$HTTP_CODE" == "200" ]]; then
    echo "OK: application returned 200"
else
    echo "FAIL: application returned $HTTP_CODE"
    exit 1
fi

echo "==> Checking recent logs for known errors..."
if kubectl logs -n fantasy-frc -l app=fantasy-frc-web --since=2m | grep -E "relation \"drafts\" does not exist|pg_stat_statements"; then
    echo "FAIL: found known DB errors in logs"
    exit 1
else
    echo "OK: no known DB errors in recent logs"
fi

echo "==> Validation complete."
