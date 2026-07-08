# Ubuntu Autoinstall

This directory contains an Ubuntu 24.04 autoinstall configuration for quickly provisioning physical Kubernetes nodes.

## Files

- `user-data.template` — Base autoinstall config
- `generate-user-data.sh` — Generate per-node `user-data` files

## Generate per-node user-data

1. Generate a password hash (optional; you can use SSH keys only):

```bash
openssl passwd -6 "your-password"
```

2. Generate the user-data file for each node:

```bash
./generate-user-data.sh cp1 jmisbach ~/.ssh/id_rsa.pub "$6$rounds..."
./generate-user-data.sh cp2 jmisbach ~/.ssh/id_rsa.pub "$6$rounds..."
./generate-user-data.sh cp3 jmisbach ~/.ssh/id_rsa.pub "$6$rounds..."
```

This creates `user-data-cp1`, `user-data-cp2`, etc.

## Create bootable USB

1. Download the Ubuntu 24.04 Server ISO:

```bash
wget https://releases.ubuntu.com/24.04/ubuntu-24.04.2-live-server-amd64.iso
```

2. Extract the ISO or use a tool like `cubic` / `autoinstall-quickstart` to inject the `user-data` file.

For a quick test, you can boot the ISO and press `e` on the installer entry, then add:

```
autoinstall ds=nocloud-net;s=https://your-internal-server/cp1/
```

Where the URL serves the per-node `user-data` and an empty `meta-data` file.

## Simpler approach for a home lab

If you prefer not to remaster the ISO, you can:

1. Boot the standard Ubuntu Server ISO manually on each machine
2. Install Ubuntu with the same username/hostname pattern
3. Use the Ansible playbooks in `infra/ansible/` to do all Kubernetes-specific configuration automatically

The autoinstall is mainly useful if you are reinstalling nodes frequently or want a fully hands-off process.

## After OS install

Once the node is installed and reachable via SSH, add it to `infra/ansible/inventory/hosts.ini` and run:

```bash
cd infra/ansible
ansible-playbook -i inventory/hosts.ini site.yaml
```
