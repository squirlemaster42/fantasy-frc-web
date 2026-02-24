#!/bin/bash
set -e

echo "=== Fantasy FRC Deployment Test ==="

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v vagrant &> /dev/null; then
    echo "Error: Vagrant not found. Install from https://www.vagrantup.com/"
    exit 1
fi

if ! command -v ansible &> /dev/null; then
    echo "Error: Ansible not found. Install: pip3 install ansible"
    exit 1
fi

if ! command -v VirtualBox &> /dev/null; then
    echo "Error: VirtualBox not found. Install from https://www.virtualbox.org/"
    exit 1
fi

echo "All prerequisites satisfied."
echo ""
echo "Starting Vagrant VM..."
vagrant up

echo ""
echo "=== VM Running ==="
echo "SSH to test: vagrant ssh"
echo "View logs: vagrant ssh -c 'sudo journalctl -u fantasy-frc -f'"
echo ""
echo "To destroy: vagrant destroy -f"
