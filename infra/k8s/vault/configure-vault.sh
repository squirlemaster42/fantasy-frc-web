#!/usr/bin/env bash
# Configure Vault for External Secrets Operator integration.
# Run after Vault is initialized and unsealed.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
VAULT_INIT_FILE="${VAULT_INIT_FILE:-${INFRA_DIR}/vault-init.json}"

if [[ ! -f "${VAULT_INIT_FILE}" ]]; then
    echo "ERROR: Vault init file not found at ${VAULT_INIT_FILE}" >&2
    echo "Set VAULT_INIT_FILE to the path of your vault-init.json" >&2
    exit 1
fi

ROOT_TOKEN="$(jq -r '.root_token' "${VAULT_INIT_FILE}")"
VAULT_CMD="kubectl exec vault-0 -n vault -- env VAULT_TOKEN=${ROOT_TOKEN} vault"

log() {
    echo -e "\n[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

log "Enabling KV v2 secrets engine at path 'secret'..."
${VAULT_CMD} secrets enable -path=secret kv-v2 2>/dev/null || log "KV v2 already enabled"

log "Creating Vault policy for External Secrets Operator..."
kubectl exec -i vault-0 -n vault -- sh -c 'cat > /tmp/eso-policy.hcl' <<'EOF'
path "secret/data/*" {
  capabilities = ["read"]
}
path "secret/metadata/*" {
  capabilities = ["read", "list"]
}
EOF
kubectl exec vault-0 -n vault -- env VAULT_TOKEN=${ROOT_TOKEN} vault policy write external-secrets /tmp/eso-policy.hcl

log "Enabling Kubernetes auth method..."
${VAULT_CMD} auth enable kubernetes 2>/dev/null || log "Kubernetes auth already enabled"

log "Configuring Kubernetes auth..."
${VAULT_CMD} write auth/kubernetes/config \
    kubernetes_host="https://kubernetes.default.svc" \
    kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
    token_reviewer_jwt=@/var/run/secrets/kubernetes.io/serviceaccount/token \
    disable_issuer_verification=true

log "Creating Kubernetes auth role for External Secrets Operator..."
${VAULT_CMD} write auth/kubernetes/role/external-secrets \
    bound_service_account_names=external-secrets \
    bound_service_account_namespaces=external-secrets \
    policies=external-secrets \
    ttl=1h

echo ""
echo "Vault configured for External Secrets Operator."
