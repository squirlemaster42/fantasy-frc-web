#!/usr/bin/env bash
# Re-execute with bash if accidentally invoked with sh/dash
if [ -z "${BASH_VERSION:-}" ]; then
    exec bash "$0" "$@"
fi
set -euo pipefail

# Kubernetes single-node setup for Ubuntu 24.04
# Run as root: sudo bash infra/k8s/cluster/setup-k8s.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Load configuration if present
if [[ -f "${INFRA_DIR}/config.env" ]]; then
    source "${INFRA_DIR}/config.env"
fi

# Configuration (override via config.env or environment variables)
K8S_VERSION="${K8S_VERSION:-1.32}"
POD_CIDR="${POD_CIDR:-192.168.0.0/16}"
API_IP="${API_IP:-192.168.1.164}"
REGULAR_USER="${SUDO_USER:-${USER}}"
HOME_DIR="$(eval echo ~"${REGULAR_USER}")"

log() {
    echo -e "\n[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root or with sudo."
    exit 1
fi

# -----------------------------------------------------------------------------
# 1. Update packages and install prerequisites
# -----------------------------------------------------------------------------
log "Updating packages and installing prerequisites..."
apt-get update
apt-get install -y apt-transport-https ca-certificates curl gpg software-properties-common

# -----------------------------------------------------------------------------
# 2. Disable swap permanently
# -----------------------------------------------------------------------------
log "Disabling swap..."
swapoff -a || true
sed -i.bak '/\sswap\s/s/^/#/' /etc/fstab
if grep -qE '^[^#].*\sswap\s' /etc/fstab; then
    echo "ERROR: swap still enabled in /etc/fstab" >&2
    exit 1
fi

# -----------------------------------------------------------------------------
# 3. Load kernel modules and configure sysctl
# -----------------------------------------------------------------------------
log "Configuring kernel modules and sysctl..."
modprobe overlay
modprobe br_netfilter

cat > /etc/modules-load.d/k8s.conf <<EOF
overlay
br_netfilter
EOF

cat > /etc/sysctl.d/99-k8s.conf <<EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sysctl --system

# -----------------------------------------------------------------------------
# 4. Configure containerd cgroup driver (drop-in override)
# -----------------------------------------------------------------------------
log "Ensuring containerd uses systemd cgroup driver..."
mkdir -p /etc/containerd/conf.d
cat > /etc/containerd/conf.d/99-k8s-cgroup.toml <<'EOF'
[plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.runc.options]
  SystemdCgroup = true
EOF

systemctl restart containerd
systemctl enable containerd

# -----------------------------------------------------------------------------
# 5. Install kubeadm, kubelet, kubectl
# -----------------------------------------------------------------------------
log "Installing Kubernetes ${K8S_VERSION} packages..."
mkdir -p /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION}/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION}/deb/ /" \
    > /etc/apt/sources.list.d/kubernetes.list

apt-get update
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl

systemctl enable kubelet

# -----------------------------------------------------------------------------
# 6. Initialize Kubernetes cluster
# -----------------------------------------------------------------------------
log "Initializing Kubernetes cluster..."
if [[ -f /etc/kubernetes/admin.conf ]]; then
    log "Cluster already initialized (/etc/kubernetes/admin.conf exists). Skipping kubeadm init."
else
    kubeadm init \
        --apiserver-advertise-address="${API_IP}" \
        --pod-network-cidr="${POD_CIDR}" \
        --node-name="$(hostname)" \
        --upload-certs \
        --v=5
fi

# -----------------------------------------------------------------------------
# 7. Configure kubectl for the regular user
# -----------------------------------------------------------------------------
log "Setting up kubectl config for user: ${REGULAR_USER}..."
mkdir -p "${HOME_DIR}/.kube"
cp -f /etc/kubernetes/admin.conf "${HOME_DIR}/.kube/config"
chown -R "${REGULAR_USER}:${REGULAR_USER}" "${HOME_DIR}/.kube"
chmod 600 "${HOME_DIR}/.kube/config"

# Make kubectl available in current shell for the rest of the script
export KUBECONFIG=/etc/kubernetes/admin.conf

# -----------------------------------------------------------------------------
# 8. Install Calico CNI
# -----------------------------------------------------------------------------
log "Installing Calico CNI..."
if kubectl get ds -n kube-system calico-node >/dev/null 2>&1; then
    log "Calico already installed."
else
    kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/tigera-operator.yaml
    kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/custom-resources.yaml
fi

# Wait for Calico to be ready
log "Waiting for Calico pods to be ready (this may take a minute)..."
kubectl wait --for=condition=ready pod -l k8s-app=calico-node -n kube-system --timeout=180s || true

# -----------------------------------------------------------------------------
# 9. Remove control-plane taint for single-node scheduling
# -----------------------------------------------------------------------------
log "Removing control-plane taint to allow workload scheduling..."
kubectl taint nodes --all node-role.kubernetes.io/control-plane- || true

# -----------------------------------------------------------------------------
# 10. Install local-path storage provisioner
# -----------------------------------------------------------------------------
log "Installing local-path-provisioner..."
if kubectl get storageclass local-path >/dev/null 2>&1; then
    log "local-path StorageClass already exists."
else
    kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.30/deploy/local-path-storage.yaml
    kubectl patch storageclass local-path -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
fi

# -----------------------------------------------------------------------------
# 11. Install NGINX Ingress Controller
# -----------------------------------------------------------------------------
log "Installing NGINX Ingress Controller..."
if kubectl get deployment -n ingress-nginx ingress-nginx-controller >/dev/null 2>&1; then
    log "NGINX Ingress Controller already installed."
else
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.12.0/deploy/static/provider/baremetal/deploy.yaml
fi

# -----------------------------------------------------------------------------
# 12. Final verification
# -----------------------------------------------------------------------------
log "Verifying cluster health..."
kubectl get nodes -o wide
kubectl get pods -n kube-system
kubectl get storageclass

echo ""
echo "================================================================================"
echo "  Kubernetes setup complete!"
echo "================================================================================"
echo ""
echo "kubectl config: ${HOME_DIR}/.kube/config"
echo "API server:     https://${API_IP}:6443"
echo ""
echo "To join additional worker nodes later, run this command on the new node:"
kubeadm token create --print-join-command 2>/dev/null || echo "  (run 'kubeadm token create --print-join-command' to generate a new join command)"
echo ""
echo "Ingress controller NodePort is on ports 80 (HTTP) and 443 (HTTPS) via"
echo "the ingress-nginx-controller Service in namespace ingress-nginx."
echo ""
echo "Run 'kubectl get nodes' and 'kubectl get pods -A' to verify."
