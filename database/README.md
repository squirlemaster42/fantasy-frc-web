# Fantasy FRC Database Migrations

This directory contains all database schema management for Fantasy FRC using [goose](https://github.com/pressly/goose).

## Prerequisites

- [goose](https://github.com/pressly/goose) CLI: `go install github.com/pressly/goose/v3/cmd/goose@latest`
- Docker (for local testing)
- PostgreSQL 16+ (for local development)

## Directory Structure

```
database/
├── Makefile                          # Migration commands
├── migrations/
│   ├── 00001_baseline.sql            # Combined current schema
│   └── 00002_add_discord_fields.sql  # Discord integration fields
├── archive/                          # Original SQL scripts (reference only)
├── test-migrations.sh                # Local Docker-based test
└── Dockerfile                        # Standalone migration image
```

## Workflow

### Create a new migration

```bash
cd database
make create name=add_user_preferences
```

This creates a new `.sql` file in `migrations/` with `-- +goose Up` and `-- +goose Down` stubs.

### Environment Variables

Migration commands require `DB_USERNAME`, `DB_PASSWORD`, `DB_IP`, and `DB_NAME`.

If you have a `.env` file in this directory, load it before running `make`:

```bash
cd database
set -a && source .env && set +a
make up
```

Or export them manually:

```bash
export DB_USERNAME=...
export DB_PASSWORD=...
export DB_IP=...
export DB_NAME=...
make up
```

### Run migrations locally

```bash
cd database
make up
```

### Check migration status

```bash
cd database
make status
```

### Rollback one migration

```bash
cd database
make down
```

### Test migrations (full up/down cycle in Docker)

```bash
cd database
make test
```

This spins up an ephemeral PostgreSQL container, runs all migrations up, verifies tables, then rolls everything down.

## Production Bootstrap

For an existing production database that already has the schema, run the following **once** to mark the baseline as applied:

```sql
INSERT INTO goose_db_version (version_id, is_applied, tstamp)
VALUES (1, true, now());
```

After this, `goose up` will only apply migration `00002` and any future migrations.

## K8s Deployment

The migration image is built separately from the application:

```bash
docker build -t fantasy-frc-migrate:latest ./database
```

Run as a K8s Job:

```bash
kubectl apply -f infra/k8s/database/migrate-job.yaml
kubectl wait --for=condition=complete job/db-migrate
```

## Notes

- Migrations are **manually triggered** and run as K8s Jobs, not on application startup.
- All migrations must have both `-- +goose Up` and `-- +goose Down` sections.
- The baseline migration (`00001`) creates the full schema from scratch. It does not reference historical migration files.
