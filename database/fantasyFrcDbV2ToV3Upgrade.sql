COMMENT ON DATABASE fantasyfrc IS 'Version: 3';
ALTER TABLE Drafts Add Status varchar;

CREATE TABLE DraftReaders(
    Id SERIAL PRIMARY KEY,
    player int REFERENCES Users(Id) NOT NULL,
    draft int REFERENCES Drafts(Id) NOT NULL
);
