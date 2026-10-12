-- +goose Up
SELECT 'up SQL query';

Create Type timingType As Enum ('per_pick', 'increment');

Alter Table Drafts Add Column TimingType timingType;
Alter Table Drafts Add Column IncrementTimeSec smallint;
Alter Table Drafts Add Column PerPickExpTimeSec smallint;
Alter Table Drafts Drop Column Interval;

Alter Table DraftPlayers Add Column RemainingPickTimeSec int;

-- +goose Down
SELECT 'down SQL query';

Alter Table Drafts Drop Column TimingType;
Alter Table Drafts Drop Column IncrementTimeSec;
Alter Table Drafts Drop Column PerPickExpTimeSec;
Alter Table Drafts Add Column Interval interval;

Alter Table DraftPlayers Drop Column RemainingPickTimeSec;

Drop Type timingType;
