COMMENT ON DATABASE fantasyfrc IS 'Version: 5';
Alter Table Picks Add Column Skipped Boolean Default False;
Alter Table Picks Add Column AvailableTime Timestamp;
ALTER TABLE Picks COLUMN Pick DROP NOT NULL;
ALTER TABLE Picks COLUMN PickTime DROP NOT NULL;
ALTER TABLE Teams RENAME RankingScore TO AllianceScore;
