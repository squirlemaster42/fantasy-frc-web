Create Table UserSessions (
    Id Serial PRIMARY KEY,
    userId int REFERENCES Users(Id) NOT NULL,
    sessionToken bytea NOT NULL,
    expirationTime TIMESTAMP NOT NULL);

Create Unique Index idx_user_username On Users(username);
