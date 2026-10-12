-- +goose Up
-- Add per-user, per-draft Discord notification preferences

CREATE TABLE UserDraftNotificationPreferences (
    UserUuid UUID NOT NULL REFERENCES Users(UserUuid) ON DELETE CASCADE,
    DraftId INT NOT NULL REFERENCES Drafts(Id) ON DELETE CASCADE,
    UpcomingMatch BOOLEAN NOT NULL DEFAULT FALSE,
    PickTurn BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (UserUuid, DraftId)
);

CREATE INDEX idx_user_draft_notification_preferences_draft ON UserDraftNotificationPreferences(DraftId);

-- +goose Down
-- Remove per-user, per-draft Discord notification preferences

DROP TABLE IF EXISTS UserDraftNotificationPreferences;
