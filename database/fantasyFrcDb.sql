CREATE DATABASE fantasyfrc;
COMMENT ON DATABASE fantasyfrc IS 'Version: 1';
Create Table Teams(tbaId varchar(10) PRIMARY KEY,
    name varchar(255),
    rankingScore smallint);
Create Table Users(Id SERIAL PRIMARY KEY,
    username varchar(255),
    password varchar(100),
    isadmin boolean);
Create Table Drafts(Id SERIAL PRIMARY KEY,
    DisplayName varchar(255),
    Owner int REFERENCES Users(Id) NOT NULL);
Create Table Matches(tbaId varchar(20) PRIMARY KEY,
    played boolean,
    redScore smallint,
    blueScore smallint);
Create Table Matches_Teams(team_tbaId varchar(10)
    REFERENCES Teams(tbaId) ON UPDATE CASCADE ON DELETE CASCADE,
    match_tbaId varchar(20) REFERENCES matches(tbaId) ON UPDATE CASCADE,
    CONSTRAINT match_team_pkey PRIMARY KEY (team_tbaId, match_tbaId),
    alliance varchar(4),
    isDqed boolean);
Create Table DraftPlayers(Id SERIAL PRIMARY KEY,
    draftId int REFERENCES drafts(Id) NOT NULL,
    playerOrder smallint,
    player int REFERENCES Users(Id) NOT NULL);
CREATE TABLE DraftInvites(Id SERIAL PRIMARY KEY,
    draftId int REFERENCES Drafts(Id) NOT NULL,
    invitedPlayer int REFERENCES Users(Id) NOT NULL,
    invitingPlayer int REFERENCES Users(Id) NOT NULL,
    sentTime TIMESTAMP NOT NULL,
    acceptedTime TIMESTAMP,
    accepted boolean,
    canceled boolean);
Create Table Picks(Id SERIAL PRIMARY KEY,
    player int REFERENCES DraftPlayers(Id) NOT NULL,
    pickOrder smallint NOT NULL,
    pick varchar(10) REFERENCES Teams(tbaId) NOT NULL,
    pickTime TIMESTAMP NOT NULL);
Create Table UserSessions (
    Id Serial PRIMARY KEY,
    userId int REFERENCES Users(Id) NOT NULL,
    sessionToken bytea NOT NULL,
    expirationTime TIMESTAMP NOT NULL);

Create Unique Index idx_user_username On Users(username);

ALTER TABLE Drafts Add Status varchar;

CREATE TABLE DraftReaders(
    Id SERIAL PRIMARY KEY,
    player int REFERENCES Users(Id) NOT NULL,
    draft int REFERENCES Drafts(Id) NOT NULL
);

ALTER TABLE Drafts Add Description TEXT;
ALTER TABLE Drafts Add StartTime Timestamp;
ALTER TABLE Drafts Add EndTime Timestamp;
ALTER TABLE Drafts Add Interval Interval;

Alter Table Picks Add Column ExpirationTime Timestamp Not Null;

Alter Table Picks Add Column Skipped Boolean Default False;
Alter Table Picks Add Column AvailableTime Timestamp;
ALTER TABLE Picks ALTER COLUMN Pick DROP NOT NULL;
ALTER TABLE Picks ALTER COLUMN PickTime DROP NOT NULL;
ALTER TABLE Teams RENAME RankingScore TO AllianceScore;
