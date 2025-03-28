COMMENT ON DATABASE fantasyfrc IS 'Version: 4';
ALTER TABLE Drafts Add Description TEXT;
ALTER TABLE Drafts Add StartTime Timestamp;
ALTER TABLE Drafts Add EndTime Timestamp;
ALTER TABLE Drafts Add Interval Interval;
