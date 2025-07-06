# fantasy-frc-web

Fantasy Frc is a game that was created by students on Frc Team 1699 during district championships in 2018. In the years since, there have
been several rules changes, many different scoring applications developed, but the general principal stayed the same. This project is
what we hope to be the final form of the many appications that have been developed over the past few years. In 2018, we built a scorer
in Java that reached out to The Blue Alliance and just scored Qual matches. Drafts were done using Google Sheets and all scoring outside of
qualification matches was done by hand. Introduced in 2025, Fantasy Frc Web is an effort to atomate the entire drafting and scoring process.
This project allows for users to create drafts, invite players, draft teams, and automatically score all aspects of the competition from
Qual matches to alliance selection to playoffs and Einstein in real time using TBA web hooks. 

# Setup

## Install Go
Fantasy FRC is build to use the latest version of [Go](https://go.dev/doc/install). Current system is built against 23/24.

## Install Templ
A guide to install Templ can be found [here](https://templ.guide/quick-start/installation/).

## Install Postgres and Setting Up your Database
There are many guides on how to install Postgres so I will not detail that here. Once you have Postgres installed create a database.
The name does not necesarily matter but you will need to reference it in the next set setting up the `.env` file so make sure you know the name.
Once your database is created connect to it with `\c <database-name>`. Once you are connected to the database run the database setup script found
at `database\fantasyFrcDb.sql`. This will setup all of the required tables needed to run Fantasy Frc. There will also be several migration scripts fhat should be run. 

TBD - Database versioning

## Setting up your .env file

In order for Fantasy FRC to run correctly, the envirment variables
listed below must be places in a `.env` file which should be located in the `server/` directory.
The file should contain the following contents:
```
TBA_TOKEN=
DB_PASSWORD=
DB_USERNAME=
DB_IP=
DB_NAME=
SESSION_SECRET=
SERVER_PORT=
```

## Building Fantasy FRC

Fantasy FRC is built using `make` so ensure you have this installed.
The make file has several options that can be set to turn off certain features while testing
- skipScoring: When set to true, the application will not score matches or teams. This makes
it so you are not making tons of calls to The Blue Alliance while testing and since the scorer
if not running there will be fewer logs in the terminal.
- populateTeams: When set to true, on startup, the application will reach out to The Blue Alliance
and grab all of the teams who are all the currently configured set of events. It will then add
those teams to the database which allows them to be picked.
