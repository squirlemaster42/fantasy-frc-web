# Database Migrations

Database versioning and migration procedures for Fantasy FRC using [goose](https://github.com/pressly/goose).

## 📋 Migration Files

All migrations are located in `database/migrations/` and follow the goose naming convention:

```
NNNN_description.sql
```

Where `NNNN` is a zero-padded sequence number and `description` briefly explains the migration.

### Migration History

| Version | File | Description |
|---------|------|-------------|
| 00001 | `00001_baseline.sql` | Combined current schema. Creates core tables: Users, Teams, Drafts, Matches, Matches_Teams, DraftPlayers, DraftInvites, Picks, UserSessions, DraftReaders, TbaCache. Enables `uuid-ossp` and `pg_stat_statements` extensions. |
| 00002 | `00002_add_discord_fields.sql` | Adds `DiscordWebhook` to Drafts and `DiscordId` to Users for Discord integration. |
| 00003 | `00003_premove_pick_list.sql` | Creates the `PickMoves` table for future premove pick-list support. |
| 00004 | `00004_timezone_timestamptz.sql` | Converts all timestamp columns to `TIMESTAMPTZ` so the database stores UTC instants. |
| 00005 | `00005_draftinvite_status.sql` | Replaces `accepted`/`canceled` boolean columns on `DraftInvites` with a single `Status` enum (`pending`, `accepted`, `declined`, `canceled`). |
| 00006 | `00006_draft_invite_player_constraints.sql` | Adds unique constraints preventing duplicate players in a draft and duplicate pending invites. |
| 00007 | `00007_add_pick_timing.sql` | Adds `TimingType`, `IncrementTimeSec`, and `PerPickExpTimeSec` to Drafts and `RemainingPickTimeSec` to DraftPlayers; drops the old `Interval` column. |
| 00008 | `00008_add_user_draft_notification_preferences.sql` | Creates `UserDraftNotificationPreferences` for per-user, per-draft Discord notification opt-ins. |

### Legacy SQL Files

The `database/archive/` directory contains the original ad-hoc scripts from before goose was adopted. They are kept for reference only and should not be run directly:

| File | Notes |
|------|-------|
| `fantasyFrcDb.sql` | Original baseline schema script |
| `changeUserIdToGuid.sql` | UUID migration script |
| `etagUpgrade.sql` | TBA cache table creation |
| `optInSkip.sql` | `skipPicks` column addition |

## 🚀 Running Migrations

### Local Development

The `database/Makefile` loads environment variables from `database/.env` automatically if present.

```bash
cd database
make up
```

You can also export variables manually (these take precedence over `.env`):

```bash
export DB_USERNAME=...
export DB_PASSWORD=...
export DB_IP=localhost
export DB_NAME=fantasy_frc

cd database
make up
```

### Available Commands

```bash
cd database

# Create a new migration
make create name=add_my_feature

# Apply pending migrations
make up

# Check migration status
make status

# Rollback one migration
make down

# Test full up/down cycle in Docker
make test
```

### Manual Execution

If you need to run goose directly:

```bash
cd database
goose postgres "user=$DB_USERNAME password=$DB_PASSWORD host=$DB_IP dbname=$DB_NAME sslmode=disable" up
```

## 🔄 Rollback Support

Each migration file must contain both `-- +goose Up` and `-- +goose Down` sections. Roll back one migration at a time with:

```bash
cd database
make down
```

To roll back multiple migrations, run `make down` repeatedly or use goose directly with a target version.

## 📝 Creating New Migrations

1. Create a new file from the `database/` directory:
   ```bash
   cd database
   make create name=my_feature_description
   ```
2. Write the `-- +goose Up` and `-- +goose Down` SQL.
3. Test locally using `make up` and `make down`.
4. Run the Docker-based test with `make test`.
5. Update this documentation and [`schema.md`](./schema.md) with the new table/column details.

## 🔗 Related Documentation

- [Database Schema](./schema.md) - Complete table structure and relationships
- [Schema Visual Guide](./schema-visual.md) - Visual database diagrams
- [Database README](../../database/README.md) - Full goose workflow and K8s deployment notes

---

*Last updated: 2026-09-04*
