-- +goose Up

-- Remove any duplicate draft players that were created before this constraint existed.
DELETE FROM DraftPlayers
WHERE Id NOT IN (
    SELECT MIN(Id)
    FROM DraftPlayers
    GROUP BY draftId, UserUuid
);

-- Prevent the same user from being added to the same draft twice.
ALTER TABLE DraftPlayers
    ADD CONSTRAINT DraftPlayers_draftId_userUuid_unique
    UNIQUE (draftId, UserUuid);

-- Prevent duplicate pending invites for the same user in the same draft.
CREATE UNIQUE INDEX DraftInvites_draftId_invitedUserUuid_pending_unique
    ON DraftInvites(draftId, InvitedUserUuid)
    WHERE Status = 'pending';

-- +goose Down

DROP INDEX DraftInvites_draftId_invitedUserUuid_pending_unique;

ALTER TABLE DraftPlayers
    DROP CONSTRAINT DraftPlayers_draftId_userUuid_unique;
