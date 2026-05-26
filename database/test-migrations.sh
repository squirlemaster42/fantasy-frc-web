#!/bin/bash
set -e

echo "=== Testing Database Migrations ==="

# Check prerequisites
if ! command -v docker &> /dev/null; then
    echo "Error: Docker not found"
    exit 1
fi

if ! command -v goose &> /dev/null; then
    echo "Error: goose CLI not found. Install with: go install github.com/pressly/goose/v3/cmd/goose@latest"
    exit 1
fi

DB_NAME="fantasyfrc_test"
DB_USER="fantasyfrc"
DB_PASS="testpassword"
DB_PORT="5433"
CONTAINER_NAME="fantasy-frc-postgres-test"
MIGRATIONS_DIR="$(dirname "$0")/migrations"

export PGPASSWORD="${DB_PASS}"

echo "Starting PostgreSQL container on port ${DB_PORT}..."
docker rm -f "${CONTAINER_NAME}" 2>/dev/null || true
docker run -d \
    --name "${CONTAINER_NAME}" \
    -e POSTGRES_DB="${DB_NAME}" \
    -e POSTGRES_USER="${DB_USER}" \
    -e POSTGRES_PASSWORD="${DB_PASS}" \
    -p "${DB_PORT}:5432" \
    postgres:16

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL..."
for i in {1..15}; do
    if docker exec "${CONTAINER_NAME}" pg_isready -U "${DB_USER}" > /dev/null 2>&1; then
        break
    fi
    sleep 1
done

DB_URL="postgresql://${DB_USER}@localhost:${DB_PORT}/${DB_NAME}?sslmode=disable"

echo ""
echo "Running UP migrations..."
goose -dir "${MIGRATIONS_DIR}" postgres "${DB_URL}" up

echo ""
echo "Migration status after UP:"
goose -dir "${MIGRATIONS_DIR}" postgres "${DB_URL}" status

echo ""
echo "Verifying tables exist..."
docker exec -i "${CONTAINER_NAME}" psql -U "${DB_USER}" -d "${DB_NAME}" -c "\dt" | grep -E "teams|users|drafts|matches|picks" && echo "Tables verified."

echo ""
echo "Running DOWN migrations (full rollback)..."
goose -dir "${MIGRATIONS_DIR}" postgres "${DB_URL}" down-to 0

echo ""
echo "Migration status after DOWN:"
goose -dir "${MIGRATIONS_DIR}" postgres "${DB_URL}" status

echo ""
echo "Cleaning up container..."
docker stop "${CONTAINER_NAME}" > /dev/null
docker rm "${CONTAINER_NAME}" > /dev/null

echo ""
echo "=== Migration test complete ==="
