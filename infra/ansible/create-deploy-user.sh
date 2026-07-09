#!/usr/bin/env bash
# Create a dedicated deployment user with passwordless sudo for Ansible.
# Run as root or with sudo.

set -euo pipefail

DEPLOY_USER="deploy"
SSH_KEY_FILE="${1:-${HOME}/.ssh/id_rsa.pub}"

if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root or with sudo."
    exit 1
fi

if [[ ! -f "${SSH_KEY_FILE}" ]]; then
    echo "ERROR: SSH public key not found: ${SSH_KEY_FILE}"
    echo "Usage: $0 /path/to/id_rsa.pub"
    exit 1
fi

echo "Creating user ${DEPLOY_USER}..."
id -u "${DEPLOY_USER}" >/dev/null 2>&1 || useradd -m -s /bin/bash "${DEPLOY_USER}"
usermod -aG sudo "${DEPLOY_USER}"

echo "Configuring passwordless sudo..."
echo "${DEPLOY_USER} ALL=(ALL) NOPASSWD: ALL" > "/etc/sudoers.d/${DEPLOY_USER}"
chmod 440 "/etc/sudoers.d/${DEPLOY_USER}"

echo "Adding SSH key..."
mkdir -p "/home/${DEPLOY_USER}/.ssh"
cp "${SSH_KEY_FILE}" "/home/${DEPLOY_USER}/.ssh/authorized_keys"
chown -R "${DEPLOY_USER}:${DEPLOY_USER}" "/home/${DEPLOY_USER}/.ssh"
chmod 700 "/home/${DEPLOY_USER}/.ssh"
chmod 600 "/home/${DEPLOY_USER}/.ssh/authorized_keys"

echo ""
echo "Deployment user '${DEPLOY_USER}' created with passwordless sudo."
echo "Update your Ansible inventory to use:"
echo "  ansible_user=${DEPLOY_USER}"
