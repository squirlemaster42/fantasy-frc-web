#!/bin/bash
set -e

MIGRATIONS_DIR="$(dirname "$0")/../migrations"
DB_HOST="${DB_IP:-localhost}"
DB_NAME="${DB_NAME:-fantasyfrc}"
DB_USER="${DB_USERNAME:-fantasyfrc}"

if [ -z "$DB_PASSWORD" ]; then
    echo "Error: DB_PASSWORD environment variable not set"
    exit 1
fi

export PGPASSWORD="$DB_PASSWORD"

echo "Running migrations from $MIGRATIONS_DIR"
echo "Database: $DB_NAME on $DB_HOST"

for migration in "$MIGRATIONS_DIR"/*.up.sql; do
    if [ -f "$migration" ]; then
        filename=$(basename "$migration")
        echo "Running: $filename"
        psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -f "$migration" 2>&1 | grep -v "already exists" || true
    fi
done

echo "Migrations complete"
