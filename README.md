# Fantasy FRC Web

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Fantasy FRC is a web-based fantasy league game for FIRST Robotics Competition
(FRC) teams. Created by then students (now alumni) of FRC Team 1699 (the Robocats)
during the 2018 New England FIRST District Championships, this project automates 
the entire drafting and scoring process for FRC competitions.

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Building and Running](#building-and-running)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Prerequisites

- [Go](https://go.dev/doc/install) 1.24+
- [Templ](https://templ.guide/quick-start/installation/)
- [PostgreSQL](https://www.postgresql.org/download/)
- [Make](https://www.gnu.org/software/make/)

### Install Go

Fantasy FRC is built using the latest version of Go. The current system requires Go 1.23+.

### Install Templ

A guide to install Templ can be found [here](https://templ.guide/quick-start/installation/).
Make sure you install the Templ Go Tool with `go get -tool github.com/a-h/templ/cmd/templ`

### Install PostgreSQL and Set Up Database

1. Install PostgreSQL using your system's package manager or from the official website.
2. Create a new database:
   ```sql
   CREATE DATABASE fantasy_frc;
   ```
3. Connect to the database and run the setup script:
   ```bash
   psql -d fantasy_frc -f database/fantasyFrcDb.sql
   ```
4. Run any additional migration scripts as needed.

**Note**: Database versioning is planned for future releases.

## Configuration
.
Create a `.env` file in the `server/` directory with the following environment variables:

```env
TBA_TOKEN=your_tba_api_token
DB_PASSWORD=your_db_password
DB_USERNAME=your_db_username
DB_IP=your_db_host
DB_NAME=fantasy_frc
SESSION_SECRET=your_session_secret
SERVER_PORT=8080
```

- `TBA_TOKEN`: Your API token from [The Blue Alliance](https://www.thebluealliance.com/account)
- `DB_*`: Database connection details
- `SESSION_SECRET`: Random string for session encryption
- `SERVER_PORT`: Override port for the web server (default: 8080)

## Building and Running

Fantasy FRC uses `make` for building. The Makefile includes options to disable certain features during testing or prepopulate teams:

- `skipScoring=true`: Disables match and team scoring to avoid excessive TBA API calls during development
- `populateTeams=true`: Populates the database with teams from configured events on startup (to be deprecated, will be automated)

### Build and Run

```bash
# Build and run the application
make

# Build with options
make skipScoring=true
make populateTeams=true
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
