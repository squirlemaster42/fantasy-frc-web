#!/bin/bash
# Reset a Kubernetes node for a fresh Ansible rebuild.
# This does NOT flush all iptables rules (which can drop your SSH session).
# kubeadm reset removes Kubernetes-managed rules itself.

set -euo pipefail

echo "==> Resetting kubeadm state..."
sudo kubeadm reset -f || true

echo "==> Removing Kubernetes directories..."
sudo rm -rf \
  /etc/kubernetes \
  /var/lib/etcd \
  /var/lib/kubelet \
  /etc/cni/net.d \
  "$HOME/.kube"

echo "==> Restarting container runtime and kubelet..."
sudo systemctl restart containerd || true
sudo systemctl restart kubelet || true

echo "==> Node reset complete. Run the Ansible playbook when ready."
