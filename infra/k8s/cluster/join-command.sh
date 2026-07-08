#!/usr/bin/env bash
# Print the current kubeadm join command for adding worker nodes.

set -euo pipefail

if ! command -v kubeadm >/dev/null 2>&1; then
    echo "kubeadm not found. Run this script on the control plane node." >&2
    exit 1
fi

echo ""
echo "Run the following command on a new worker node to join the cluster:"
echo ""
kubeadm token create --print-join-command
echo ""
echo "This token is valid for 24 hours. Use 'kubeadm token create' to generate a new one."
