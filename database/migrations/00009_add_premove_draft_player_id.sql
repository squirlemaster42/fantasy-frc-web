-- +goose Up
-- Link each premove to the draft player that created it.
-- The column was previously added out-of-band in some environments, so this
-- migration is idempotent and safe to run where it already exists.

ALTER TABLE PickMoves ADD COLUMN IF NOT EXISTS DraftPlayerId INT;

-- A premove with no owning draft player is unusable, so remove any such rows
-- before enforcing NOT NULL.
DELETE FROM PickMoves WHERE DraftPlayerId IS NULL;

ALTER TABLE PickMoves ALTER COLUMN DraftPlayerId SET NOT NULL;

ALTER TABLE PickMoves DROP CONSTRAINT IF EXISTS PickMoves_DraftPlayerId_fkey;
ALTER TABLE PickMoves
    ADD CONSTRAINT PickMoves_DraftPlayerId_fkey
    FOREIGN KEY (DraftPlayerId) REFERENCES DraftPlayers(Id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_pickmoves_draft_player_status_pick_order
    ON PickMoves(DraftPlayerId, Status, PickOrder);

-- +goose Down
DROP INDEX IF EXISTS idx_pickmoves_draft_player_status_pick_order;

ALTER TABLE PickMoves DROP CONSTRAINT IF EXISTS PickMoves_DraftPlayerId_fkey;

ALTER TABLE PickMoves DROP COLUMN IF EXISTS DraftPlayerId;
