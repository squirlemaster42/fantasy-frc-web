-- +goose Up
Create Table PickMoves (
    Id Serial Primary Key,
    TeamTBAId Varchar(10) References Teams(tbaid) On Update Cascade On Delete Cascade,
    Status Varchar(10),
    Error Varchar(255),
    PickOrder SmallInt
);

-- +goose Down
SELECT 'down SQL query';
Drop Table PickMoves;
