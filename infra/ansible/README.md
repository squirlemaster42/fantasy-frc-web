# Ansible Kubernetes Deployment

This directory contains Ansible playbooks for deploying the Fantasy FRC stack on a Kubernetes cluster running Ubuntu 24.04.

The default inventory is a **single-node control plane** suitable for a home lab. The playbooks also support a **multi-node HA control plane** with worker nodes if you expand the inventory.

## What the playbooks install

- `site.yaml` — OS packages, containerd, kubeadm, kubelet, Helm, nerdctl, jq; Kubernetes cluster; Calico CNI; Longhorn storage.
- `build-images.yaml` — Build the Fantasy FRC web and migration container images locally using nerdctl + BuildKit.
- `deploy-apps.yaml` — Ingress-NGINX, cert-manager, Postgres, Redis, HashiCorp Vault, External Secrets Operator, Grafana/Loki/Tempo monitoring, and the Fantasy FRC web app.

## Requirements

- Target node(s) running Ubuntu 24.04 Server.
- Internet access from the nodes to download packages, Helm charts, container images, and release binaries.
- The repository checked out at `/home/{{ regular_user }}/fantasy-frc-web` (used by the image build role).
- On the admin/control-plane node:
  - `ansible-playbook`
  - `git`
  - `jq`
  - SSH key-based access to remote nodes (only needed for multi-node)

`scripts/setup.sh` checks for `ansible-playbook`, `git`, and `jq` before running.

## Inventory

The committed `inventory/hosts.ini` is configured for a single-node home lab:

```ini
[control_plane]
cp1 ansible_host=192.168.1.164 ansible_connection=local

[workers]

[k8s:children]
control_plane
workers

[all:vars]
ansible_user=jmisbach
ansible_ssh_private_key_file=~/.ssh/id_rsa
api_vip=192.168.1.164
```

For a multi-node HA cluster, copy `inventory/hosts.ini.example` and edit it:

```bash
cp inventory/hosts.ini.example inventory/hosts.ini
```

Then add your nodes and set a dedicated `api_vip` that is in the same subnet but outside your DHCP pool.

## Configuration

Edit `group_vars/all.yaml`:

```yaml
api_vip: "192.168.1.164"   # For single-node, use the node IP; for HA, use a dedicated VIP
api_ip: "192.168.1.164"    # Static IP used for ingress access
fantasy_frc_domain: "fantasy-frc.local"
grafana_domain: "grafana.local"
vault_domain: "vault.local"
regular_user: "jmisbach"
```

At minimum, update the IP addresses and domains for your network.

## New machine setup

The easiest path for a single-node home lab is the helper script. It runs `site.yaml`, `build-images.yaml`, and `deploy-apps.yaml` in order.

```bash
cd infra/ansible

# Wipes any existing cluster, containerd data, and persistent volumes, then reboots
./scripts/teardown.sh

# After the node reboots and you log back in:
./scripts/setup.sh
./scripts/validate.sh
```

`setup.sh` will prompt for your sudo password when Ansible needs it.

### Running the playbooks manually

If you prefer to run the playbooks individually:

```bash
# Provision the cluster and storage (requires sudo)
ansible-playbook -i inventory/hosts.ini site.yaml -K

# Build the Fantasy FRC container images (requires sudo)
ansible-playbook -i inventory/hosts.ini build-images.yaml -K

# Deploy the application stack (does NOT require sudo)
ansible-playbook -i inventory/hosts.ini deploy-apps.yaml
```

## Verify

```bash
kubectl get nodes
kubectl get pods -n kube-system
kubectl get storageclass
./scripts/validate.sh
```

## Updating the application after a code change

When you push new webapp code, rebuild and redeploy:

```bash
ansible-playbook -i inventory/hosts.ini build-images.yaml -K
ansible-playbook -i inventory/hosts.ini deploy-apps.yaml
```

Images are tagged with the current git short SHA and also `:latest`.

### Local uncommitted changes

If you edited code without committing, the git short SHA has not changed, so the playbook may skip the build thinking the image already exists. Force a rebuild:

```bash
ansible-playbook -i inventory/hosts.ini build-images.yaml -K -e force_build=true
ansible-playbook -i inventory/hosts.ini deploy-apps.yaml
```

### Forcing a rebuild with the `:latest` tag

If you are using `:latest` and want to rebuild even though the image already exists:

```bash
ansible-playbook -i inventory/hosts.ini build-images.yaml -K -e force_build=true
```

## Access the applications

Add the control-plane IP to your local `/etc/hosts`:

```
192.168.1.164  fantasy-frc.local grafana.local vault.local
```

Then browse to:

- `https://fantasy-frc.local:30556`
- `https://grafana.local:30556`
- `https://vault.local:30556`

Trust `infra/k8s/cert-manager/homelab-ca.crt` on your local machine to avoid browser warnings.

## Important notes

- `site.yaml` and `build-images.yaml` need sudo (`-K`). `deploy-apps.yaml` does not.
- The playbooks are idempotent: re-running them will skip nodes already in the cluster and skip Vault/app secrets that already exist.
- The first control-plane node must finish initializing before additional control-plane or worker nodes can join. `site.yaml` handles this ordering.
- For HA, `api_vip` must be a dedicated static IP outside your DHCP pool.
- Keep `infra/k8s/vault-init.json` safe. If Vault is initialized and this file is lost, the playbook will stop with an error telling you to restore it or tear down.
