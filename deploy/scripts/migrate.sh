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

run_sql() {
    psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -t -A -c "$1" 2>/dev/null
}

run_sql_file() {
    psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -f "$1" 2>&1 | grep -v "already exists" || true
}

echo "Running migrations from $MIGRATIONS_DIR"
echo "Database: $DB_NAME on $DB_HOST"

# Ensure migration tracking table exists
run_sql "CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(255) PRIMARY KEY, applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW());"

for migration in "$MIGRATIONS_DIR"/*.up.sql; do
    if [ -f "$migration" ]; then
        filename=$(basename "$migration")
        version="${filename%.up.sql}"

        # Check if already applied
        applied=$(run_sql "SELECT version FROM schema_migrations WHERE version = '$version';")
        if [ "$applied" = "$version" ]; then
            echo "Skipping (already applied): $filename"
            continue
        fi

        echo "Running: $filename"
        run_sql_file "$migration"

        # Record as applied
        run_sql "INSERT INTO schema_migrations (version) VALUES ('$version') ON CONFLICT DO NOTHING;"
    fi
done

echo "Migrations complete"
