#!/usr/bin/env bash
# Install cert-manager and configure the homelab CA and certificates.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -f "${INFRA_DIR}/config.env" ]]; then
    source "${INFRA_DIR}/config.env"
fi

HELM="${HELM:-${HOME}/.local/bin/helm}"
FANTASY_FRC_DOMAIN="${FANTASY_FRC_DOMAIN:-fantasy-frc.local}"
GRAFANA_DOMAIN="${GRAFANA_DOMAIN:-grafana.local}"
VAULT_DOMAIN="${VAULT_DOMAIN:-vault.local}"

log() {
    echo -e "\n[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

log "Installing cert-manager..."
${HELM} repo add jetstack https://charts.jetstack.io 2>/dev/null || true
${HELM} repo update
kubectl create namespace cert-manager --dry-run=client -o yaml | kubectl apply -f -
${HELM} upgrade --install cert-manager jetstack/cert-manager \
    --namespace cert-manager \
    --version 1.17.1 \
    --set crds.enabled=true \
    --wait \
    --timeout 10m

log "Creating CA and ClusterIssuer..."
kubectl apply -f "${SCRIPT_DIR}/ca-setup.yaml"

log "Creating certificates..."
CERTS_VALUES="$(mktemp)"
envsubst < "${SCRIPT_DIR}/certificates.yaml" > "${CERTS_VALUES}"
trap 'rm -f "${CERTS_VALUES}"' EXIT

kubectl apply -f "${CERTS_VALUES}"

log "Waiting for certificates to be ready..."
kubectl wait --for=condition=ready certificate -n fantasy-frc fantasy-frc-tls --timeout=60s || true
kubectl wait --for=condition=ready certificate -n monitoring grafana-tls --timeout=60s || true
kubectl wait --for=condition=ready certificate -n vault vault-tls --timeout=60s || true

log "Exporting CA certificate..."
kubectl get secret homelab-ca-secret -n cert-manager -o jsonpath='{.data.ca\.crt}' | base64 -d > "${SCRIPT_DIR}/homelab-ca.crt"

echo ""
echo "cert-manager installed and certificates issued."
echo "CA certificate saved to: ${SCRIPT_DIR}/homelab-ca.crt"
