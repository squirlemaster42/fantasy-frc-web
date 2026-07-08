#!/usr/bin/env bash
# Install the Grafana monitoring stack (Prometheus, Grafana, Loki, Tempo)
# into the Kubernetes cluster.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -f "${INFRA_DIR}/config.env" ]]; then
    source "${INFRA_DIR}/config.env"
fi

HELM="${HELM:-${HOME}/.local/bin/helm}"
NAMESPACE="monitoring"
API_IP="${API_IP:-192.168.1.164}"
GRAFANA_DOMAIN="${GRAFANA_DOMAIN:-grafana.local}"

log() {
    echo -e "\n[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

# -----------------------------------------------------------------------------
# 1. Ensure namespace and Helm repos exist
# -----------------------------------------------------------------------------
log "Ensuring namespace ${NAMESPACE} exists..."
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

log "Adding Helm repositories..."
${HELM} repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
${HELM} repo add grafana https://grafana.github.io/helm-charts 2>/dev/null || true
${HELM} repo update

# -----------------------------------------------------------------------------
# 2. Generate Grafana admin password
# -----------------------------------------------------------------------------
GRAFANA_PASSWORD_FILE="${SCRIPT_DIR}/grafana-admin-password.txt"
if [[ -f "${GRAFANA_PASSWORD_FILE}" ]]; then
    GRAFANA_PASSWORD="$(cat "${GRAFANA_PASSWORD_FILE}")"
    log "Using existing Grafana admin password from ${GRAFANA_PASSWORD_FILE}"
else
    GRAFANA_PASSWORD="$(openssl rand -base64 24)"
    echo "${GRAFANA_PASSWORD}" > "${GRAFANA_PASSWORD_FILE}"
    chmod 600 "${GRAFANA_PASSWORD_FILE}"
    log "Generated Grafana admin password and saved to ${GRAFANA_PASSWORD_FILE}"
fi

# -----------------------------------------------------------------------------
# 3. Install kube-prometheus-stack
# -----------------------------------------------------------------------------
log "Installing kube-prometheus-stack (Prometheus + Grafana)..."
KPS_VALUES="$(mktemp)"
envsubst < "${SCRIPT_DIR}/kube-prometheus-stack-values.yaml" > "${KPS_VALUES}"
trap 'rm -f "${KPS_VALUES}"' EXIT

${HELM} upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
    --namespace "${NAMESPACE}" \
    --version 68.4.3 \
    --values "${KPS_VALUES}" \
    --set grafana.adminPassword="${GRAFANA_PASSWORD}" \
    --wait \
    --timeout 10m

# -----------------------------------------------------------------------------
# 4. Install Loki
# -----------------------------------------------------------------------------
log "Installing Loki..."
${HELM} upgrade --install loki grafana/loki \
    --namespace "${NAMESPACE}" \
    --version 6.25.0 \
    --values "${SCRIPT_DIR}/loki-values.yaml" \
    --wait \
    --timeout 10m

# -----------------------------------------------------------------------------
# 5. Install Tempo
# -----------------------------------------------------------------------------
log "Installing Tempo..."
${HELM} upgrade --install tempo grafana/tempo \
    --namespace "${NAMESPACE}" \
    --version 1.18.1 \
    --values "${SCRIPT_DIR}/tempo-values.yaml" \
    --wait \
    --timeout 10m

# -----------------------------------------------------------------------------
# 6. Verify
# -----------------------------------------------------------------------------
log "Verifying monitoring stack..."
kubectl get pods -n "${NAMESPACE}"

# Print access info
GRAFANA_NODEPORT="$(${HELM} get values kube-prometheus-stack -n "${NAMESPACE}" -o json 2>/dev/null | \
    jq -r '.grafana.ingress.hosts[0] // "grafana.local"' 2>/dev/null || echo "grafana.local")"

INGRESS_HTTP_PORT="$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.spec.ports[?(@.name=="http")].nodePort}' 2>/dev/null || echo "31531")"

echo ""
echo "================================================================================"
echo "  Grafana monitoring stack installed!"
echo "================================================================================"
echo ""
echo "Grafana URL:     http://${API_IP}:${INGRESS_HTTP_PORT}"
echo "Grafana host:    ${GRAFANA_NODEPORT}"
echo "Admin username:  admin"
echo "Admin password:  ${GRAFANA_PASSWORD}"
echo "Password file:   ${GRAFANA_PASSWORD_FILE}"
echo ""
echo "If you access Grafana from a browser, add this line to your /etc/hosts:"
echo "  ${API_IP}  ${GRAFANA_NODEPORT}"
echo ""
echo "Or use the IP and NodePort directly:"
echo "  http://${API_IP}:${INGRESS_HTTP_PORT}"
echo ""
echo "To forward Grafana locally:"
echo "  kubectl port-forward -n ${NAMESPACE} svc/kube-prometheus-stack-grafana 3000:80"
