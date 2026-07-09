#!/usr/bin/env bash
# Install HashiCorp Vault and External Secrets Operator.
# Vault must be initialized and unsealed manually after this script.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -f "${INFRA_DIR}/config.env" ]]; then
    source "${INFRA_DIR}/config.env"
fi

HELM="${HELM:-${HOME}/.local/bin/helm}"

log() {
    echo -e "\n[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

log "Installing Vault..."
${HELM} repo add hashicorp https://helm.releases.hashicorp.com 2>/dev/null || true
${HELM} repo update
kubectl create namespace vault --dry-run=client -o yaml | kubectl apply -f -
${HELM} upgrade --install vault hashicorp/vault \
    --namespace vault \
    --version 0.29.1 \
    --values "${SCRIPT_DIR}/vault-values.yaml" \
    --wait \
    --timeout 10m

log "Installing External Secrets Operator..."
${HELM} repo add external-secrets https://charts.external-secrets.io 2>/dev/null || true
${HELM} repo update
kubectl create namespace external-secrets --dry-run=client -o yaml | kubectl apply -f -
${HELM} upgrade --install external-secrets external-secrets/external-secrets \
    --namespace external-secrets \
    --version 0.14.0 \
    --set installCRDs=true \
    --wait \
    --timeout 10m

echo ""
echo "Vault and External Secrets Operator installed."
echo ""
echo "NEXT STEPS:"
echo "1. Initialize and unseal Vault:"
echo "   kubectl exec -it vault-0 -n vault -- vault operator init -key-shares=5 -key-threshold=3 -format=json > vault-init.json"
echo "2. Save vault-init.json securely."
echo "3. Unseal Vault using 3 of the 5 keys."
echo "4. Run configure-vault.sh"
echo "5. Apply cluster-secret-store.yaml and your ExternalSecret resources."
