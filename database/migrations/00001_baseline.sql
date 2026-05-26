-- +goose Up
-- Baseline schema representing the current state after all historical migrations
-- (excluding Discord fields which are in migration 00002)

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Teams table
CREATE TABLE Teams (
    tbaId VARCHAR(10) PRIMARY KEY,
    name VARCHAR(255),
    allianceScore SMALLINT
);

-- Users table with UUID primary key
CREATE TABLE Users (
    UserUuid UUID PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    password VARCHAR(100) NOT NULL,
    isAdmin BOOLEAN
);

CREATE UNIQUE INDEX idx_user_username ON Users(username);

-- Drafts table
CREATE TABLE Drafts (
    Id SERIAL PRIMARY KEY,
    DisplayName VARCHAR(255) NOT NULL,
    OwnerUserUuid UUID NOT NULL REFERENCES Users(UserUuid),
    Status VARCHAR,
    Description TEXT,
    StartTime TIMESTAMP,
    EndTime TIMESTAMP,
    Interval INTERVAL
);

-- Matches table
CREATE TABLE Matches (
    tbaId VARCHAR(20) PRIMARY KEY,
    played BOOLEAN,
    redScore SMALLINT,
    blueScore SMALLINT
);

-- Match-Team associations
CREATE TABLE Matches_Teams (
    team_tbaId VARCHAR(10) REFERENCES Teams(tbaId) ON UPDATE CASCADE ON DELETE CASCADE,
    match_tbaId VARCHAR(20) REFERENCES Matches(tbaId) ON UPDATE CASCADE,
    alliance VARCHAR(4),
    isDqed BOOLEAN,
    CONSTRAINT match_team_pkey PRIMARY KEY (team_tbaId, match_tbaId)
);

-- Draft players
CREATE TABLE DraftPlayers (
    Id SERIAL PRIMARY KEY,
    draftId INT NOT NULL REFERENCES Drafts(Id),
    playerOrder SMALLINT,
    UserUuid UUID NOT NULL REFERENCES Users(UserUuid),
    skipPicks BOOLEAN DEFAULT FALSE
);

-- Draft invites
CREATE TABLE DraftInvites (
    Id SERIAL PRIMARY KEY,
    draftId INT NOT NULL REFERENCES Drafts(Id),
    InvitedUserUuid UUID NOT NULL REFERENCES Users(UserUuid),
    InvitingUserUuid UUID NOT NULL REFERENCES Users(UserUuid),
    sentTime TIMESTAMP NOT NULL,
    acceptedTime TIMESTAMP,
    accepted BOOLEAN,
    canceled BOOLEAN
);

-- Picks
CREATE TABLE Picks (
    Id SERIAL PRIMARY KEY,
    player INT NOT NULL REFERENCES DraftPlayers(Id),
    pick VARCHAR(10) REFERENCES Teams(tbaId),
    pickTime TIMESTAMP,
    ExpirationTime TIMESTAMP NOT NULL,
    Skipped BOOLEAN DEFAULT FALSE,
    AvailableTime TIMESTAMP
);

-- User sessions
CREATE TABLE UserSessions (
    Id SERIAL PRIMARY KEY,
    UserUuid UUID NOT NULL REFERENCES Users(UserUuid),
    sessionToken BYTEA NOT NULL,
    expirationTime TIMESTAMP NOT NULL
);

-- Draft readers (spectators)
CREATE TABLE DraftReaders (
    Id SERIAL PRIMARY KEY,
    UserUuid UUID NOT NULL REFERENCES Users(UserUuid),
    draft INT NOT NULL REFERENCES Drafts(Id)
);

-- TBA API cache
CREATE TABLE TbaCache (
    url TEXT PRIMARY KEY,
    etag VARCHAR(255),
    responseBody BYTEA
);

-- +goose Down
-- Drop tables in reverse dependency order to respect foreign keys

DROP TABLE IF EXISTS TbaCache;
DROP TABLE IF EXISTS Matches_Teams;
DROP TABLE IF EXISTS Picks;
DROP TABLE IF EXISTS DraftInvites;
DROP TABLE IF EXISTS DraftReaders;
DROP TABLE IF EXISTS DraftPlayers;
DROP TABLE IF EXISTS UserSessions;
DROP TABLE IF EXISTS Matches;
DROP TABLE IF EXISTS Drafts;
DROP TABLE IF EXISTS Teams;
DROP TABLE IF EXISTS Users;

-- NOTE: We intentionally do NOT drop extensions here.
-- uuid-ossp and pg_stat_statements are infrastructure-level
-- and may require superuser privileges to remove.
