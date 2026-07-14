#!/usr/bin/env bash
# Deploy the Fantasy FRC web application.
# Assumes Postgres, Redis, Vault, and External Secrets Operator are already deployed.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -f "${INFRA_DIR}/config.env" ]]; then
    source "${INFRA_DIR}/config.env"
fi

FANTASY_FRC_DOMAIN="${FANTASY_FRC_DOMAIN:-fantasy-frc.local}"
REGISTRY_HOST="${REGISTRY_HOST:-registry.fantasy-frc.svc.cluster.local}"
REGISTRY_PORT="${REGISTRY_PORT:-5000}"

log() {
    echo -e "\n[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

render_manifest() {
    local input="$1"
    local output="$(mktemp)"
    envsubst < "${input}" > "${output}"
    echo "${output}"
}

log "Creating namespace..."
kubectl create namespace fantasy-frc --dry-run=client -o yaml | kubectl apply -f -

log "Deploying registry and Redis..."
APP_MANIFEST="$(render_manifest "${SCRIPT_DIR}/registry.yaml")"
kubectl apply -f "${APP_MANIFEST}"
rm -f "${APP_MANIFEST}"

REDIS_MANIFEST="$(render_manifest "${SCRIPT_DIR}/redis.yaml")"
kubectl apply -f "${REDIS_MANIFEST}"
rm -f "${REDIS_MANIFEST}"

log "Deploying ExternalSecret..."
ES_MANIFEST="$(render_manifest "${SCRIPT_DIR}/external-secret.yaml")"
kubectl apply -f "${ES_MANIFEST}"
rm -f "${ES_MANIFEST}"

log "Waiting for Redis..."
kubectl wait --for=condition=ready pod -l app=redis -n fantasy-frc --timeout=120s

log "Deploying app..."
APP_MANIFEST="$(render_manifest "${SCRIPT_DIR}/app.yaml")"
kubectl apply -f "${APP_MANIFEST}"
rm -f "${APP_MANIFEST}"

log "Waiting for app to be ready..."
kubectl wait --for=condition=ready pod -l app=fantasy-frc-web -n fantasy-frc --timeout=120s

echo ""
echo "Fantasy FRC web deployed."
echo "Access: https://${FANTASY_FRC_DOMAIN}:30556"
