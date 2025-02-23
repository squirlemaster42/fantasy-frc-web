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
