#!/usr/bin/env bash
# Re-execute with bash if accidentally invoked with sh/dash
if [ -z "${BASH_VERSION:-}" ]; then
    exec bash "$0" "$@"
fi
set -euo pipefail

# Prepare a new worker node to join the Kubernetes cluster.
# Must match the control-plane Kubernetes version.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -f "${INFRA_DIR}/config.env" ]]; then
    source "${INFRA_DIR}/config.env"
fi

K8S_VERSION="${K8S_VERSION:-1.32}"

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
# 4. Install containerd
# -----------------------------------------------------------------------------
log "Installing containerd..."
apt-get install -y containerd

# -----------------------------------------------------------------------------
# 5. Configure containerd with systemd cgroup driver
# -----------------------------------------------------------------------------
log "Configuring containerd..."
mkdir -p /etc/containerd/conf.d
cat > /etc/containerd/conf.d/99-k8s-cgroup.toml <<'EOF'
[plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.runc.options]
  SystemdCgroup = true
EOF

systemctl restart containerd
systemctl enable containerd

# -----------------------------------------------------------------------------
# 6. Install kubeadm, kubelet, kubectl
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
# 7. Final instructions
# -----------------------------------------------------------------------------
echo ""
echo "================================================================================"
echo "  Worker node preparation complete!"
echo "================================================================================"
echo ""
echo "On the control plane node, run:"
echo ""
echo "    kubeadm token create --print-join-command"
echo ""
echo "Then run the resulting 'kubeadm join' command on this worker node."
