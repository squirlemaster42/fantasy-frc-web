# Ansible Kubernetes Deployment

This directory contains Ansible playbooks for deploying a high-availability Kubernetes cluster on Ubuntu 24.04.

## Architecture

- **3 control-plane nodes** running kubeadm with a virtual IP (VIP)
- **Optional worker nodes**
- **kube-vip** for API server HA and LoadBalancer services
- **Calico** for CNI
- **Longhorn** for replicated storage

## Requirements

- Ansible installed on your admin machine
- SSH key-based access to all nodes
- Nodes running Ubuntu 24.04 Server
- A free static IP for the control-plane VIP in the same subnet as the nodes

## Inventory

Copy the example inventory and update it:

```bash
cp inventory/hosts.ini.example inventory/hosts.ini
```

Example:

```ini
[control_plane]
cp1 ansible_host=192.168.1.171
cp2 ansible_host=192.168.1.172
cp3 ansible_host=192.168.1.173

[workers]
worker1 ansible_host=192.168.1.174

[k8s:children]
control_plane
workers

[all:vars]
ansible_user=jmisbach
ansible_ssh_private_key_file=~/.ssh/id_rsa
```

## Configuration

Edit `group_vars/all.yaml`:

```yaml
api_vip: "192.168.1.170"
pod_cidr: "192.168.0.0/16"
service_cidr: "10.96.0.0/12"
lb_ip_range: "192.168.1.200-192.168.1.210"
regular_user: "jmisbach"
```

## Run the cluster setup

```bash
cd infra/ansible
ansible-playbook -i inventory/hosts.ini site.yaml
```

This will:

1. Prepare all nodes (OS packages, containerd, kubeadm, kubelet)
2. Deploy kube-vip static pod manifests on all control-plane nodes
3. Initialize the first control-plane node
4. Join the remaining control-plane nodes
5. Join worker nodes
6. Install Calico
7. Install Longhorn and set it as the default StorageClass

## Verify

```bash
ssh cp1
kubectl get nodes
kubectl get pods -n kube-system
kubectl get storageclass
```

## Next steps

After the cluster is up, deploy the applications from `infra/k8s/`:

1. Vault + External Secrets Operator
2. cert-manager
3. Monitoring stack
4. Postgres + Redis
5. Fantasy FRC web app

See `infra/k8s/README.md` for details.

## Notes

- The first control-plane node must be reachable before the others can join. `site.yaml` handles this ordering.
- If a node is already part of the cluster, the playbooks skip it (idempotent).
- The VIP (`api_vip`) must not be in your DHCP pool.
