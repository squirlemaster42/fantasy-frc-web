#!/usr/bin/env bash
# Deploy Postgres for the Fantasy FRC application.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -f "${INFRA_DIR}/config.env" ]]; then
    source "${INFRA_DIR}/config.env"
fi

DB_NAME="${DB_NAME:-appdb}"
DB_USERNAME="${DB_USERNAME:-postgres}"

log() {
    echo -e "\n[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

log "Creating namespace..."
kubectl create namespace database --dry-run=client -o yaml | kubectl apply -f -

log "Generating Postgres password..."
PGPASSWORD="$(openssl rand -base64 32)"

log "Creating secret..."
kubectl create secret generic postgres-secret \
    --namespace=database \
    --from-literal=password="${PGPASSWORD}" \
    --dry-run=client -o yaml | kubectl apply -f -

log "Deploying Postgres..."
envsubst '$DB_NAME $DB_USERNAME' < "${SCRIPT_DIR}/postgres.yaml" | kubectl apply -f -

log "Waiting for Postgres StatefulSet rollout..."
kubectl rollout status statefulset/postgres -n database --timeout=180s

log "Waiting for Postgres pod to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres -n database --timeout=120s

echo ""
echo "Postgres deployed."
echo "Username: ${DB_USERNAME}"
echo "Database: ${DB_NAME}"
echo "Password saved in secret: postgres-secret (namespace: database)"
echo ""
echo "Store this password in Vault for External Secrets Operator:"
echo "  kubectl exec vault-0 -n vault -- env VAULT_TOKEN=\$ROOT_TOKEN vault kv put secret/database/postgres password=${PGPASSWORD}"
