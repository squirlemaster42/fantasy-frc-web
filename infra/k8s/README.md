# Fantasy FRC Kubernetes Infrastructure

This directory contains all the Kubernetes manifests, Helm values, and scripts needed to replicate the Fantasy FRC stack on a new machine.

> For **high-availability multi-node clusters**, use the Ansible playbooks in `infra/ansible/` to provision the Kubernetes nodes. This directory contains the application-level manifests and Helm values used after the cluster is running.

## What is included

```
infra/k8s/
├── config.env.example      # Example configuration (copy to config.env)
├── .gitignore              # Excludes secrets and generated artifacts
├── README.md               # This file
├── cluster/                # Single-node cluster bootstrap scripts
│   ├── setup-k8s.sh
│   ├── worker-setup.sh
│   └── join-command.sh
├── cert-manager/           # TLS certificates
│   ├── ca-setup.yaml
│   ├── certificates.yaml
│   ├── homelab-ca.crt
│   └── install-cert-manager.sh
├── postgres/               # Postgres database
│   ├── postgres.yaml
│   └── install-postgres.sh
├── monitoring/             # Grafana/Prometheus/Loki/Tempo
│   ├── kube-prometheus-stack-values.yaml
│   ├── loki-values.yaml
│   ├── tempo-values.yaml
│   └── install-monitoring.sh
├── vault/                  # HashiCorp Vault + External Secrets Operator
│   ├── vault-values.yaml
│   ├── configure-vault.sh
│   ├── cluster-secret-store.yaml
│   ├── external-secrets.yaml
│   └── install-vault.sh
├── redis/                  # Redis cache
│   ├── redis.yaml
│   └── install-redis.sh
├── fantasy-frc/            # Fantasy FRC web application
│   ├── app.yaml
│   ├── external-secret.yaml
│   ├── registry.yaml
│   ├── kaniko-build.yaml
│   ├── kaniko-build-migrations-context.yaml
│   ├── migrations-Dockerfile
│   └── install-fantasy-frc.sh
└── database/               # Database migrations
    ├── migrate-job.yaml
    └── migrate-job-vault.yaml
```

## Replication steps

### Single-node setup

For a single-node cluster, follow the steps below using `infra/k8s/cluster/setup-k8s.sh`.

### Multi-node HA setup

For 3+ nodes with HA control plane, use the Ansible playbooks in `infra/ansible/` instead:

```bash
cd infra/ansible
cp inventory/hosts.ini.example inventory/hosts.ini
# Edit inventory/hosts.ini and group_vars/all.yaml
ansible-playbook -i inventory/hosts.ini site.yaml
```

Then return here to deploy the applications.

### 1. Prepare configuration

Copy the example config and update values for your environment:

```bash
cd infra/k8s
cp config.env.example config.env
# Edit config.env with your API_IP, domains, etc.
```

At minimum, update:

- `API_IP` — static IP of your control-plane node
- `FANTASY_FRC_DOMAIN`, `GRAFANA_DOMAIN`, `VAULT_DOMAIN` — local domains

### 2. Set up the Kubernetes cluster (single-node)

On the control-plane node:

```bash
sudo bash infra/k8s/cluster/setup-k8s.sh
```

For additional worker nodes:

```bash
sudo bash infra/k8s/cluster/worker-setup.sh
# Then run the join command printed on the control plane
```

### 3. Install core infrastructure

```bash
# Postgres
bash infra/k8s/postgres/install-postgres.sh

# Redis
bash infra/k8s/redis/install-redis.sh

# Vault + External Secrets Operator
bash infra/k8s/vault/install-vault.sh

# Initialize and unseal Vault
kubectl exec -it vault-0 -n vault -- vault operator init \
  -key-shares=5 -key-threshold=3 -format=json > infra/k8s/vault-init.json

# Save vault-init.json securely, then unseal with 3 keys
kubectl exec vault-0 -n vault -- vault operator unseal "<key1>"
kubectl exec vault-0 -n vault -- vault operator unseal "<key2>"
kubectl exec vault-0 -n vault -- vault operator unseal "<key3>"

# Configure Vault
VAULT_INIT_FILE=infra/k8s/vault-init.json bash infra/k8s/vault/configure-vault.sh

# Create ClusterSecretStore
kubectl apply -f infra/k8s/vault/cluster-secret-store.yaml
```

### 4. Store secrets in Vault

At minimum, store the Postgres and Redis passwords, plus app secrets:

```bash
ROOT_TOKEN=$(jq -r '.root_token' infra/k8s/vault-init.json)

# Postgres password (from install-postgres.sh output)
kubectl exec vault-0 -n vault -- env VAULT_TOKEN=${ROOT_TOKEN} vault kv put secret/database/postgres \
  password="<postgres-password>"

# Redis password (from install-redis.sh output)
kubectl exec vault-0 -n vault -- env VAULT_TOKEN=${ROOT_TOKEN} vault kv patch secret/fantasy-frc/app \
  REDIS_PASSWORD="<redis-password>"

# App secrets
kubectl exec vault-0 -n vault -- env VAULT_TOKEN=${ROOT_TOKEN} vault kv put secret/fantasy-frc/app \
  DB_USERNAME="postgres" \
  DB_PASSWORD="<postgres-password>" \
  DB_IP="postgres.database.svc.cluster.local" \
  DB_NAME="appdb" \
  SERVER_PORT="8080" \
  TBA_TOKEN="your_real_tba_token" \
  TBA_WEBHOOK_SECRET="$(openssl rand -base64 32)" \
  METRIC_SECRET="$(openssl rand -base64 32)" \
  SECURE_HTTP_COOKIE="false" \
  CSRF_SECRET="$(openssl rand -base64 48)" \
  TRUST_PROXY="true" \
  ALLOWED_ORIGIN="http://${FANTASY_FRC_DOMAIN}:31531" \
  REDIS_ADDR="redis.fantasy-frc.svc.cluster.local:6379" \
  REDIS_PASSWORD="<redis-password>" \
  REDIS_RATE_LIMIT_DB="1" \
  REDIS_AVATAR_DB="2" \
  RATE_LIMIT_POSTS_PER_MINUTE="100" \
  RATE_LIMIT_ENABLED="true" \
  OTEL_EXPORTER_OTLP_ENDPOINT="http://tempo.monitoring.svc.cluster.local:4318" \
  OTEL_RESOURCE_ATTRIBUTES="service.name=fantasy-frc-web" \
  MIN_PASSWORD_LENGTH="12"
```

### 5. Install monitoring and TLS

```bash
bash infra/k8s/monitoring/install-monitoring.sh
bash infra/k8s/cert-manager/install-cert-manager.sh
```

### 6. Build and deploy the web application

The web application image is built inside the cluster using Kaniko and pushed to the local registry. Because containerd requires extra configuration for insecure local registries, images are exported as tarballs and imported manually.

```bash
# Build the web image
kubectl apply -f infra/k8s/fantasy-frc/kaniko-build.yaml
kubectl logs -n fantasy-frc job/kaniko-build -f

# Export the image to a tarball
kubectl apply -f infra/k8s/fantasy-frc/export-image-to-host.yaml
kubectl wait --for=condition=complete job/export-image -n fantasy-frc

# On the Kubernetes node, import and tag the image
sudo bash /path/to/import-image-to-containerd.sh
sudo bash /path/to/tag-imported-image.sh

# Build the migrations image
rm -rf infra/k8s/fantasy-frc/migrations-context/migrations
cp -r ../database/migrations infra/k8s/fantasy-frc/migrations-context/
kubectl apply -f infra/k8s/fantasy-frc/kaniko-build-migrations-context.yaml
kubectl wait --for=condition=complete job/kaniko-build-migrations -n fantasy-frc

# Export, import, and tag the migrations image
kubectl apply -f infra/k8s/fantasy-frc/export-migrations-image.yaml
kubectl wait --for=condition=complete job/export-migrations-image -n fantasy-frc
sudo bash /path/to/import-migrations-image.sh
sudo bash /path/to/tag-migrations-image.sh

# Run migrations
kubectl apply -f infra/k8s/fantasy-frc/migrate-job.yaml
kubectl wait --for=condition=complete job/fantasy-frc-migrate -n fantasy-frc

# Deploy the app
bash infra/k8s/fantasy-frc/install-fantasy-frc.sh
```

> Note: The containerd import scripts are node-specific helper scripts. They are not committed to git. See the `k8sSetup/fantasy-frc/` directory on the original server for examples, or create your own using `ctr -n k8s.io image import`.

### 7. Access the applications

Add to your local `/etc/hosts`:

```
<API_IP>  fantasy-frc.local
<API_IP>  grafana.local
<API_IP>  vault.local
```

Access URLs:

```
https://fantasy-frc.local:30556
https://grafana.local:30556
https://vault.local:30556
```

To remove browser warnings, trust `infra/k8s/cert-manager/homelab-ca.crt` on your local machine.

## Important notes

- **Never commit `config.env` or `vault-init.json`.** They contain sensitive values.
- The local container registry is a workaround for bare-metal clusters without an external registry. If you have a registry (Docker Hub, GHCR, etc.), update image references and skip the Kaniko/export/import steps.
- For production with a real domain, replace `homelab-ca-issuer` with a Let's Encrypt `ClusterIssuer`.
