#!/usr/bin/env bash
# Deploy Redis for the Fantasy FRC application.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -f "${INFRA_DIR}/config.env" ]]; then
    source "${INFRA_DIR}/config.env"
fi

log() {
    echo -e "\n[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

log "Creating namespace..."
kubectl create namespace fantasy-frc --dry-run=client -o yaml | kubectl apply -f -

render_manifest() {
    local input="$1"
    local output="$(mktemp)"
    envsubst < "${input}" > "${output}"
    echo "${output}"
}

log "Generating Redis password..."
REDIS_PASSWORD="$(openssl rand -base64 32)"
export REDIS_PASSWORD

log "Deploying Redis..."
MANIFEST="$(render_manifest "${SCRIPT_DIR}/redis.yaml")"
kubectl apply -f "${MANIFEST}"
rm -f "${MANIFEST}"

log "Waiting for Redis StatefulSet rollout..."
kubectl rollout status statefulset/redis -n fantasy-frc --timeout=180s

log "Waiting for Redis pod to be ready..."
kubectl wait --for=condition=ready pod -l app=redis -n fantasy-frc --timeout=120s

echo ""
echo "Redis deployed."
echo "Password saved in secret: redis-secret (namespace: fantasy-frc)"
echo ""
echo "Store this password in Vault for External Secrets Operator:"
echo "  kubectl exec vault-0 -n vault -- env VAULT_TOKEN=\$ROOT_TOKEN vault kv put secret/fantasy-frc/app REDIS_PASSWORD=${REDIS_PASSWORD}"
