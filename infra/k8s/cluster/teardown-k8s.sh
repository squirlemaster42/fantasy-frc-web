#!/usr/bin/env bash
# Tear down Kubernetes on this node.
# WARNING: This destroys all cluster data, workloads, and persistent volumes.

set -euo pipefail

if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root or with sudo."
    exit 1
fi

echo "WARNING: This will completely remove Kubernetes and all cluster data."

echo "Draining node if part of a cluster..."
kubectl drain "$(hostname)" --ignore-daemonsets --delete-emptydir-data --force 2>/dev/null || true

echo "Resetting kubeadm (timeout 5m)..."
timeout 300 kubeadm reset -f 2>/dev/null || true

echo "Stopping Kubernetes services..."
systemctl stop kubelet 2>/dev/null || true
systemctl stop containerd 2>/dev/null || true

echo "Killing any remaining containerd tasks..."
if command -v ctr >/dev/null 2>&1; then
    ctr -n k8s.io tasks kill -a -s SIGKILL $(ctr -n k8s.io tasks ls -q 2>/dev/null | tr '\n' ' ') 2>/dev/null || true
fi
pkill -9 -f 'containerd-shim-runc-v2' 2>/dev/null || true

echo "Removing Kubernetes packages..."
apt-mark unhold kubelet kubeadm kubectl 2>/dev/null || true
apt-get remove --purge -y kubelet kubeadm kubectl kubernetes-cni 2>/dev/null || true

echo "Removing containerd..."
apt-get remove --purge -y containerd 2>/dev/null || true

echo "Cleaning up directories..."
rm -rf /etc/kubernetes
rm -rf /var/lib/kubelet
rm -rf /var/lib/etcd
rm -rf /var/lib/cni
rm -rf /etc/cni
rm -rf /opt/cni
rm -rf /run/containerd
rm -rf /var/lib/containerd
rm -rf /etc/containerd
rm -f /etc/apt/sources.list.d/kubernetes.list
rm -f /etc/apt/keyrings/kubernetes-apt-keyring.gpg
rm -f /etc/modules-load.d/k8s.conf
rm -f /etc/sysctl.d/99-k8s.conf

echo "Removing CNI network interfaces..."
ip link delete cni0 2>/dev/null || true
ip link delete flannel.1 2>/dev/null || true
ip link delete vxlan.calico 2>/dev/null || true
ip link delete cali* 2>/dev/null || true

echo "Reloading sysctl..."
sysctl --system

echo "Teardown complete. A reboot is recommended before rebuilding."
