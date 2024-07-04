ALTER TABLE Drafts Add Status smallint;

CREATE TABLE DraftReaders(
    Id SERIAL PRIMARY KEY,
    player int REFERENCES Users(Id) NOT NULL,
    draft int REFERENCES Drafts(Id) NOT NULL
);
