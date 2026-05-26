# Database Migrations

Database versioning and migration procedures for Fantasy FRC.

## 📋 Migration Files

All migrations are located in `deploy/migrations/` and follow the naming convention:

```
NNN_description.up.sql
```

Where `NNN` is a zero-padded sequence number and `description` briefly explains the migration.

### Migration History

| Version | File | Description |
|---------|------|-------------|
| 001 | `001_initial.up.sql` | Initial schema creation. Creates core tables: Users, Teams, Drafts, Matches, Matches_Teams, DraftPlayers, DraftInvites, Picks, UserSessions, DraftReaders. Includes indexes, constraints, and initial column additions (Status, Description, StartTime, EndTime, Interval, ExpirationTime, Skipped, AvailableTime). Renames `RankingScore` to `AllianceScore`. |
| 002 | `002_uuid.up.sql` | Migrates all user ID references from integer `SERIAL` to `UUID` using the `uuid-ossp` extension. Updates Users, UserSessions, DraftPlayers, DraftReaders, DraftInvites, and Drafts tables with new foreign key relationships. |
| 003 | `003_etag_cache.up.sql` | Creates the `TbaCache` table (`url`, `etag`, `responseBody`) for caching The Blue Alliance API responses and reducing external API calls. |
| 004 | `004_skip_picks.up.sql` | Adds `skipPicks` boolean column to `DraftPlayers` to support the opt-in automatic skip picks feature. |
| 005 | `005_pg_stat_stats.up.sql` | Enables the `pg_stat_statements` PostgreSQL extension for query performance monitoring and analysis. |

### Legacy SQL Files

The `database/` directory contains the original ad-hoc scripts from which the numbered migrations were derived:

| File | Corresponding Migration |
|------|------------------------|
| `fantasyFrcDb.sql` | Source for `001_initial.up.sql` |
| `changeUserIdToGuid.sql` | Source for `002_uuid.up.sql` |
| `etagUpgrade.sql` | Source for `003_etag_cache.up.sql` |
| `optInSkip.sql` | Source for `004_skip_picks.up.sql` |
| `enable_pg_stat_statements.sql` | Source for `005_pg_stat_stats.up.sql` |
| `addDiscordWebhook.sql` | **Not yet migrated** — adds `DiscordWebhook` to Drafts and `DiscordId` to Users. Pending conversion to `006_*` migration. |

## 🚀 Running Migrations

### Local Development

```bash
# Set environment variables or source your .env file
export DB_PASSWORD=your_password
export DB_USERNAME=fantasyfrc
export DB_NAME=fantasyfrc
export DB_IP=localhost

# Run migrations
cd server
make migrate
```

### Manual Execution

```bash
cd deploy/scripts
./migrate.sh
```

The migration script iterates over all `*.up.sql` files in `deploy/migrations/` in alphabetical order and applies them via `psql`. It suppresses "already exists" errors so migrations can be safely re-run.

### Docker Testing

```bash
cd deploy/scripts
./test-migrations.sh
```

This spins up a temporary PostgreSQL Docker container and applies all migrations for validation.

## 🔄 Rollback Support

Rollback migrations follow the naming convention:

```
NNN_description.down.sql
```

Currently, **no `.down.sql` files exist** in the repository. Rollback support is documented as a future enhancement. To rollback changes, restore from a database backup or manually reverse the migration SQL.

## 📝 Creating New Migrations

1. Create a new file in `deploy/migrations/` with the next sequence number:
   ```bash
   touch deploy/migrations/006_my_feature.up.sql
   ```
2. Write the migration SQL
3. (Optional) Create a corresponding `.down.sql` file for rollback support
4. Test locally using `make migrate` or `deploy/scripts/test-migrations.sh`
5. Update this documentation with the new migration entry

## 🔗 Related Documentation

- [Database Schema](./schema.md) - Complete table structure and relationships
- [Schema Visual Guide](./schema-visual.md) - Visual database diagrams
- [Deployment Guide](../../deploy/README.md) - Production deployment and migration automation
