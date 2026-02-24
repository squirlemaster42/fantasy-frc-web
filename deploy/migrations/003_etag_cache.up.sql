-- TBA Cache table (from etagUpgrade.sql)
Create Table TbaCache (
    url text Primary Key,
    etag varchar(255),
    responseBody bytea
);
