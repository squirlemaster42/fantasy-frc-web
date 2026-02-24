#!/bin/bash
set -e

echo "=== Testing Database Migrations Locally ==="

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Error: Docker not found"
    exit 1
fi

DB_NAME="fantasyfrc_test"
DB_USER="fantasyfrc"
DB_PASS="testpassword"

# Start PostgreSQL container
echo "Starting PostgreSQL container..."
docker rm -f fantasy-frc-postgres 2>/dev/null || true
docker run -d \
    --name fantasy-frc-postgres \
    -e POSTGRES_DB="$DB_NAME" \
    -e POSTGRES_USER="$DB_USER" \
    -e POSTGRES_PASSWORD="$DB_PASS" \
    -p 5432:5432 \
    postgres:16

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL..."
sleep 5

for i in {1..10}; do
    if docker exec fantasy-frc-postgres pg_isready -U "$DB_USER" > /dev/null 2>&1; then
        break
    fi
    sleep 2
done

# Run migrations
echo "Running migrations..."
export PGPASSWORD="$DB_PASS"
for migration in ../deploy/migrations/*.up.sql; do
    if [ -f "$migration" ]; then
        filename=$(basename "$migration")
        echo "  Running: $filename"
        docker exec -i fantasy-frc-postgres psql -U "$DB_USER" -d "$DB_NAME" < "$migration" 2>&1 | grep -v "already exists" || true
    fi
done

# Verify tables
echo ""
echo "Tables created:"
docker exec -i fantasy-frc-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "\dt"

echo ""
echo "Migration test complete!"

# Cleanup
echo ""
read -p "Press Enter to stop PostgreSQL container..."
docker stop fantasy-frc-postgres > /dev/null
docker rm fantasy-frc-postgres > /dev/null
echo "Container stopped and removed."
