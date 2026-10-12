# Fantasy FRC Deployment

This directory previously contained Ansible playbooks and bash-based migration scripts. Deprecated. Ansible is now used to deploy to a K8s cluster; see `infra/ansible/README.md`.

## Current State

- **Database migrations** have moved to `database/` and use [goose](https://github.com/pressly/goose).

See `database/README.md` for migration workflow and `infra/k8s/` for Kubernetes manifests.
