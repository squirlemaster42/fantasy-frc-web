-- +goose Up
-- Add API keys for machine-to-machine draft automation

CREATE TABLE UserApiKeys (
    Id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    UserUuid        UUID NOT NULL REFERENCES Users(UserUuid) ON DELETE CASCADE,
    ClientId        TEXT NOT NULL UNIQUE,
    ClientSecretHash TEXT NOT NULL,
    DisplayName     TEXT NOT NULL,
    Scopes          TEXT[] NOT NULL DEFAULT ARRAY['full_access'],
    Revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    CreatedAt       TIMESTAMPTZ NOT NULL DEFAULT now(),
    LastUsedAt      TIMESTAMPTZ
);

CREATE INDEX idx_user_api_keys_user ON UserApiKeys(UserUuid);

-- +goose Down

DROP TABLE IF EXISTS UserApiKeys;
