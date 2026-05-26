# Fantasy FRC Deployment

This directory previously contained Ansible playbooks and bash-based migration scripts.

## Current State

- **Database migrations** have moved to `database/` and use [goose](https://github.com/pressly/goose).
- **K8s manifests** live in `infra/k8s/`.
- **Ansible deployment** has been removed in favor of container-based deployment.

See `database/README.md` for migration workflow and `infra/k8s/` for Kubernetes manifests.
