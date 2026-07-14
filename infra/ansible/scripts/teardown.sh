#!/usr/bin/env bash
# Full teardown of the Kubernetes home lab on this node.
# WARNING: This destroys all cluster data, workloads, and persistent volumes.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

echo "WARNING: This will completely remove Kubernetes, containerd, and all cluster data."
read -r -p "Are you sure? [y/N] " response
if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo "Aborted."
    exit 0
fi

echo "==> Pre-stopping container runtime to avoid kubeadm reset hangs..."
sudo systemctl stop kubelet 2>/dev/null || true
sudo systemctl stop containerd 2>/dev/null || true
sudo pkill -9 -f 'containerd-shim-runc-v2' 2>/dev/null || true
sudo pkill -9 -f 'kube-apiserver|kube-controller-manager|kube-scheduler|etcd' 2>/dev/null || true
sleep 5

echo "==> Running Kubernetes teardown..."
sudo "${REPO_ROOT}/infra/k8s/cluster/teardown-k8s.sh"

echo "==> Unmounting containerd rootfs and shm mounts..."
sudo find /run/containerd -type d \( -name rootfs -o -name shm \) -exec umount -l {} \; 2>/dev/null || true

echo "==> Removing Longhorn volume data..."
sudo rm -rf /var/lib/longhorn

echo "==> Removing any leftover hostpath data..."
sudo rm -rf /var/lib/postgresql

echo "==> Teardown complete. Rebooting now..."
sudo reboot
