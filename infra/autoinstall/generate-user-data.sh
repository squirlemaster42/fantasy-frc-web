#!/usr/bin/env bash
# Generate per-node Ubuntu autoinstall user-data files.
#
# Usage:
#   ./generate-user-data.sh <hostname> <username> <ssh-public-key-file> [password-hash]
#
# To generate a password hash:
#   openssl passwd -6 "your-password"

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE="${SCRIPT_DIR}/user-data.template"

if [[ $# -lt 3 ]]; then
    echo "Usage: $0 <hostname> <username> <ssh-public-key-file> [password-hash]"
    exit 1
fi

HOSTNAME="$1"
USERNAME="$2"
SSH_KEY_FILE="$3"
PASSWORD_HASH="${4:-$6$rounds=4096$abcdefghijklmnopqrstuv$abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.}"

if [[ ! -f "${SSH_KEY_FILE}" ]]; then
    echo "ERROR: SSH public key file not found: ${SSH_KEY_FILE}"
    exit 1
fi

SSH_PUBLIC_KEY="$(cat "${SSH_KEY_FILE}")"

cp "${TEMPLATE}" "${SCRIPT_DIR}/user-data-${HOSTNAME}"
sed -i "s/HOSTNAME/${HOSTNAME}/g" "${SCRIPT_DIR}/user-data-${HOSTNAME}"
sed -i "s/USERNAME/${USERNAME}/g" "${SCRIPT_DIR}/user-data-${HOSTNAME}"
sed -i "s|SSH_PUBLIC_KEY|${SSH_PUBLIC_KEY}|g" "${SCRIPT_DIR}/user-data-${HOSTNAME}"
sed -i "s|PASSWORD_HASH|${PASSWORD_HASH}|g" "${SCRIPT_DIR}/user-data-${HOSTNAME}"

echo "Generated: ${SCRIPT_DIR}/user-data-${HOSTNAME}"
