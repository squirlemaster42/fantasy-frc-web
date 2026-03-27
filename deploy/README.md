# Fantasy FRC Deployment

This directory contains all deployment scripts, Ansible playbooks, and database migrations for deploying Fantasy FRC to a Linux server.

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
│   ├── 001_initial.up.sql
│   ├── 002_uuid.up.sql
│   ├── 003_etag_cache.up.sql
│   ├── 004_skip_picks.up.sql
│   └── 005_migration_tracking.up.sql
├── nginx/
│   └── fantasy-frc.conf       # Reference Nginx reverse proxy config
└── scripts/
    ├── migrate.sh              # Migration runner script (with tracking)
    ├── test-migrations.sh      # Test migrations with Docker
    └── test-vagrant.sh         # Full Vagrant test environment
```

## Architecture

The application runs as a single Go binary on the server. Nginx (configured by the host) sits in front as a reverse proxy.

```
Internet --> Nginx (TLS termination, port 80/443)
                |
                v
            Go App (port 8080, plain HTTP)
                |
            +---+---+
            |       |
            v       v
        Postgres  Redis
        (5432)   (6379)
```

All components run on the same VM.

## Prerequisites

### Local Machine (for building and deploying)
- Go 1.24+
- Make
- Ansible 2.12+
- SSH access to the target server

### Target Server
- Ubuntu 20.04+ or Debian 11+
- SSH access with sudo privileges
- Nginx already installed and configured by the host

## First-Time Setup

### 1. Configure Server Inventory

Edit `deploy/ansible/inventory.ini` with the target server:

```ini
[fantasy_frc_servers]
production ansible_host=YOUR_SERVER_IP ansible_user=YOUR_SSH_USER
```

### 2. Configure Secrets

```bash
cd deploy/ansible/vars
cp vault.yml.example vault.yml
```

Edit `vault.yml` with your actual values:

```yaml
vault_db_password: "your-strong-db-password"
vault_tba_token: "your-tba-api-token"
vault_tba_webhook_secret: "your-webhook-secret"
vault_session_secret: "your-session-secret"
vault_sentry_dsn: ""  # optional
```

Encrypt the vault file:

```bash
ansible-vault encrypt vault.yml
```

### 3. Build the Binary

```bash
cd server
make build-linux
```

This produces a statically-linked Linux binary at `server/server`.

### 4. Deploy

```bash
cd deploy/ansible
ansible-playbook -i inventory.ini playbook.yml --ask-vault-pass
```

Enter the vault password when prompted. The playbook will:
- Install PostgreSQL and Redis
- Create the `fantasyfrc` system user and directories
- Copy the binary and migrations
- Create the database and user
- Run all migrations (with tracking)
- Start the service

### 5. Configure Nginx

The host needs to configure Nginx to reverse proxy to `127.0.0.1:8080`. A reference config is at `deploy/nginx/fantasy-frc.conf`.

Key requirements for the Nginx config:
- Proxy all requests to `http://127.0.0.1:8080`
- The `/u/draft/:id/pickNotifier` endpoint uses Server-Sent Events (SSE), so the Nginx location block needs:
  ```
  proxy_set_header Connection '';
  proxy_http_version 1.1;
  chunked_transfer_encoding off;
  proxy_buffering off;
  proxy_read_timeout 86400s;
  ```

## Environment Variables

The app is configured via `/opt/fantasy-frc/config/.env`. These are set by Ansible during deployment.

| Variable | Description | Default |
|----------|-------------|---------|
| `TBA_TOKEN` | The Blue Alliance API token | (required) |
| `DB_PASSWORD` | PostgreSQL password | (required) |
| `DB_USERNAME` | PostgreSQL username | `fantasyfrc` |
| `DB_IP` | PostgreSQL host | `localhost` |
| `DB_NAME` | PostgreSQL database name | `fantasyfrc` |
| `SESSION_SECRET` | Session encryption key | (required) |
| `SERVER_PORT` | HTTP listen port | `8080` |
| `SENTRY_DSN` | Sentry error tracking DSN | (optional) |
| `TBA_WEBHOOK_SECRET_FILE` | Path to webhook secret file | `/opt/fantasy-frc/config/tba_webhook_secret.txt` |
| `REDIS_URL` | Redis connection URL | `redis://localhost:6379/0` |
| `DB_MAX_CONNS` | Max open database connections | `25` |
| `DB_MAX_IDLE_CONNS` | Max idle database connections | `10` |
| `DB_CONN_MAX_LIFETIME_MINUTES` | Max connection lifetime | `5` |

## Database Migrations

Migrations are tracked in the `schema_migrations` table. Each `.up.sql` file in `deploy/migrations/` is applied once and recorded. Re-running the migration script skips already-applied migrations.

### Running Migrations Manually

On the server:

```bash
source /opt/fantasy-frc/config/.env
/opt/fantasy-frc/migrations/migrate.sh
```

Or from your local machine (requires `psql` client):

```bash
cd server
source .env  # or set DB_* variables
../deploy/scripts/migrate.sh
```

### Creating New Migrations

1. Create a new file in `deploy/migrations/` following the pattern:
   - `NNN_description.up.sql` — migration to apply
   - `NNN_description.down.sql` — rollback (optional, for reference)

2. Number sequentially after the last migration.

3. The migration will be automatically applied on the next deployment.

## Service Management

```bash
# Check status
sudo systemctl status fantasy-frc

# View logs (follow mode)
sudo journalctl -u fantasy-frc -f

# View recent logs
sudo journalctl -u fantasy-frc -n 100

# Restart
sudo systemctl restart fantasy-frc

# Stop (graceful shutdown, 15s timeout)
sudo systemctl stop fantasy-frc
```

The service handles `SIGTERM` for graceful shutdown:
- Stops accepting new HTTP connections
- Drains in-flight requests (up to 10s)
- Stops the draft daemon and scorer
- Closes database and Redis connections

## Updating the Application

1. Build the new binary:
   ```bash
   cd server
   make build-linux
   ```

2. Deploy:
   ```bash
   cd deploy/ansible
   ansible-playbook -i inventory.ini playbook.yml --ask-vault-pass
   ```

The playbook will:
- Copy the new binary
- Run any new migrations (skipping already-applied ones)
- Restart the service with graceful shutdown

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

### Check migration status

```bash
export PGPASSWORD="your-password"
psql -h localhost -U fantasyfrc -d fantasyfrc -c "SELECT * FROM schema_migrations ORDER BY version;"
```

### Rollback (manual)

There is no automated rollback. To revert a migration, connect to the database and:
1. Apply the corresponding `.down.sql` if it exists
2. Or manually undo the changes
3. Remove the version from `schema_migrations`:
   ```sql
   DELETE FROM schema_migrations WHERE version = 'NNN_description';
   ```
