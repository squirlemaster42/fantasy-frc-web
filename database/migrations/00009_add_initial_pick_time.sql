-- +goose Up
SELECT 'up SQL query';

Alter Table Drafts Add Column InitialPickTimeSec smallint;

-- +goose Down
SELECT 'down SQL query';

Alter Table Drafts Drop Column InitialPickTimeSec;
