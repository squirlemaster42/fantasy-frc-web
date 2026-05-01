-- +goose Up
-- Add Discord webhook and user ID fields for Discord integration

ALTER TABLE Drafts ADD COLUMN DiscordWebhook VARCHAR(150);
ALTER TABLE Users ADD COLUMN DiscordId VARCHAR(50);

-- +goose Down
-- Remove Discord fields

ALTER TABLE Drafts DROP COLUMN IF EXISTS DiscordWebhook;
ALTER TABLE Users DROP COLUMN IF EXISTS DiscordId;
