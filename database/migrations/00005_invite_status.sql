CREATE TYPE inviteStatus AS ENUM ('pending', 'accepted', 'declined', 'canceled');

ALTER TABLE DraftInvites ADD COLUMN Status inviteStatus;

UPDATE DraftInvites SET Status = CASE
    WHEN Accepted IS TRUE THEN 'accepted'::inviteStatus
    WHEN Canceled IS TRUE THEN 'canceled'::inviteStatus
    ELSE 'pending'::inviteStatus
END;

ALTER TABLE DraftInvites
    ALTER COLUMN Status SET NOT NULL;

ALTER TABLE DraftInvites
    DROP COLUMN Accepted,
    DROP COLUMN Canceled;