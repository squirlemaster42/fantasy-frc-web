#!/usr/bin/env bash
# Full setup of the Kubernetes home lab and Fantasy FRC application stack.
# Run this on the control-plane node after teardown/reboot.
set -euo pipefail

if [[ $EUID -eq 0 ]]; then
    echo "ERROR: Do not run this script with sudo or as root."
    echo "Run as the regular user (e.g., jmisbach). The script will prompt for sudo when needed."
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Preflight checks..."
missing=()
for cmd in ansible-playbook git jq; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        missing+=("$cmd")
    fi
done
if [[ ${#missing[@]} -gt 0 ]]; then
    echo "ERROR: Missing required tools: ${missing[*]}"
    echo "Install them before running this script."
    exit 1
fi

echo "==> Provisioning Kubernetes cluster..."
ansible-playbook -i "${SCRIPT_DIR}/../inventory/hosts.ini" "${SCRIPT_DIR}/../site.yaml" -K

echo "==> Building container images..."
ansible-playbook -i "${SCRIPT_DIR}/../inventory/hosts.ini" "${SCRIPT_DIR}/../build-images.yaml" -K

echo "==> Deploying application stack..."
ansible-playbook -i "${SCRIPT_DIR}/../inventory/hosts.ini" "${SCRIPT_DIR}/../deploy-apps.yaml"

echo "==> Setup complete. Run scripts/validate.sh to verify."
