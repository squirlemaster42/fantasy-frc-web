-- +goose Up
-- Convert all timestamp columns to TIMESTAMPTZ so the database stores UTC instants.
-- We set the session timezone to America/New_York before converting so existing
-- naive wall-clock values that were intended as Eastern are preserved correctly.

SET TIME ZONE 'America/New_York';

ALTER TABLE Drafts
    ALTER COLUMN StartTime TYPE TIMESTAMPTZ,
    ALTER COLUMN EndTime TYPE TIMESTAMPTZ;

ALTER TABLE DraftInvites
    ALTER COLUMN sentTime TYPE TIMESTAMPTZ,
    ALTER COLUMN acceptedTime TYPE TIMESTAMPTZ;

ALTER TABLE Picks
    ALTER COLUMN pickTime TYPE TIMESTAMPTZ,
    ALTER COLUMN ExpirationTime TYPE TIMESTAMPTZ,
    ALTER COLUMN AvailableTime TYPE TIMESTAMPTZ;

ALTER TABLE UserSessions
    ALTER COLUMN expirationTime TYPE TIMESTAMPTZ;

SET TIME ZONE 'UTC';

-- +goose Down
-- Revert to naive TIMESTAMP columns. We assume the stored instants should be
-- rendered as America/New_York wall-clock times in the down direction.

SET TIME ZONE 'America/New_York';

ALTER TABLE Drafts
    ALTER COLUMN StartTime TYPE TIMESTAMP,
    ALTER COLUMN EndTime TYPE TIMESTAMP;

ALTER TABLE DraftInvites
    ALTER COLUMN sentTime TYPE TIMESTAMP,
    ALTER COLUMN acceptedTime TYPE TIMESTAMP;

ALTER TABLE Picks
    ALTER COLUMN pickTime TYPE TIMESTAMP,
    ALTER COLUMN ExpirationTime TYPE TIMESTAMP,
    ALTER COLUMN AvailableTime TYPE TIMESTAMP;

ALTER TABLE UserSessions
    ALTER COLUMN expirationTime TYPE TIMESTAMP;

SET TIME ZONE 'UTC';
