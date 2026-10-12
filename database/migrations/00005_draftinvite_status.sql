-- +goose Up
-- Add invite status field
CREATE TYPE inviteStatus AS ENUM ('pending', 'accepted', 'declined', 'canceled');

ALTER TABLE DraftInvites ADD COLUMN Status inviteStatus;

UPDATE DraftInvites SET Status = CASE
    WHEN Accepted IS TRUE THEN 'accepted'::inviteStatus
    WHEN Canceled IS TRUE THEN 'canceled'::inviteStatus
    ELSE 'pending'::inviteStatus
END;

ALTER TABLE DraftInvites
    ALTER COLUMN Status SET NOT NULL;

-- drop old columns

ALTER TABLE DraftInvites
    DROP COLUMN Accepted,
    DROP COLUMN Canceled;

-- +goose Down

-- add old columns
ALTER TABLE DraftInvites
    ADD COLUMN Accepted BOOLEAN,
    ADD COLUMN Canceled BOOLEAN;

-- set values based on status
UPDATE DraftInvites SET
    Accepted = (Status = 'accepted'),
    Canceled = (Status = 'canceled');

-- remove status
ALTER TABLE DraftInvites
    DROP COLUMN Status;

DROP TYPE inviteStatus;