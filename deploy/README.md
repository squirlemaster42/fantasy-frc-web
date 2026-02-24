# Fantasy FRC Deployment

This directory contains all deployment scripts, Ansible playbooks, and database migrations for deploying Fantasy FRC to Linux servers.

## Directory Structure

```
deploy/
├── ansible/                    # Ansible deployment files
│   ├── playbook.yml           # Main deployment playbook
│   ├── inventory.ini          # Server inventory (configure your servers here)
│   ├── vars/
│   │   ├── main.yml           # Configuration variables
│   │   └── vault.yml.example  # Template for encrypted secrets
│   └── templates/
│       ├── fantasy-frc.service.j2  # Systemd service unit
│       └── .env.j2                 # Environment file template
├── migrations/                 # Database migration scripts
│   ├── 001_initial.up.sql     # Initial schema (from fantasyFrcDb.sql)
│   ├── 002_uuid.up.sql        # UUID migration (from changeUserIdToGuid.sql)
│   ├── 003_etag_cache.up.sql  # TBA cache table (from etagUpgrade.sql)
│   └── 004_skip_picks.up.sql  # Skip picks feature (from optInSkip.sql)
└── scripts/
    ├── migrate.sh              # Migration runner script
    ├── test-migrations.sh      # Test migrations with Docker
    └── test-vagrant.sh         # Full Vagrant test environment
```

## Prerequisites

### Local Machine
- Ansible 2.12+
- Go 1.24+ (for building)
- Make

### Target Server
- Ubuntu 20.04+ or Debian 11+
- SSH access with sudo privileges

## Quick Start

### 1. Build the Binary

```bash
cd server
make build-linux
```

### 2. Configure Secrets

```bash
cd deploy/ansible/vars
cp vault.yml.example vault.yml
# Edit vault.yml with your values
ansible-vault encrypt vault.yml
```

### 3. Configure Server Inventory

Edit `deploy/ansible/inventory.ini`:

```ini
[fantasy_frc_servers]
production ansible_host=YOUR_SERVER_IP ansible_user=deploy
```

### 4. Deploy

```bash
cd deploy/ansible
ansible-playbook -i inventory.ini playbook.yml --ask-vault-pass
```

## Configuration Variables

Edit `deploy/ansible/vars/main.yml` for non-secret configuration:

| Variable | Description | Default |
|----------|-------------|---------|
| `db_name` | PostgreSQL database name | `fantasyfrc` |
| `db_username` | PostgreSQL username | `fantasyfrc` |
| `server_port` | Server listen port | `8080` |

### Secrets (in vault.yml)

| Variable | Description |
|----------|-------------|
| `vault_db_password` | PostgreSQL password |
| `vault_tba_token` | The Blue Alliance API token |
| `vault_tba_webhook_secret` | TBA webhook secret |
| `vault_session_secret` | Session encryption secret |
| `vault_sentry_dsn` | Sentry DSN (optional) |

## Database Migrations

Migrations are run automatically during deployment. To run migrations manually:

```bash
cd server
source .env  # or set DB_* environment variables
../deploy/scripts/migrate.sh
```

### Creating New Migrations

Create a new file in `deploy/migrations/` with the naming pattern:
- `NNN_description.up.sql` - Migration to apply
- `NNN_description.down.sql` - Rollback (optional)

## Service Management

```bash
# Check status
sudo systemctl status fantasy-frc

# View logs
sudo journalctl -u fantasy-frc -f

# Restart
sudo systemctl restart fantasy-frc

# Stop
sudo systemctl stop fantasy-frc
```

## Updating

1. Build new binary: `cd server && make build-linux`
2. Run deployment: `cd deploy/ansible && ansible-playbook -i inventory.ini playbook.yml --ask-vault-pass`

The playbook will:
- Copy the new binary
- Run any pending migrations
- Restart the service

## Troubleshooting

### Service won't start
```bash
sudo journalctl -u fantasy-frc -n 50
```

### Database connection issues
```bash
sudo -u postgres psql -d fantasyfrc
```

### Redis connection issues
```bash
redis-cli ping
```
