# Fantasy FRC Web

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Fantasy FRC is a web-based, fantasy football style game for FIRST Robotics Competition
(FRC) teams. Created by students (now alumni) of FRC Team 1699 (the Robocats)
during the 2018 New England FIRST District Championships, this project automates
the entire drafting and scoring process for Fantasy FRC.

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Building and Running](#building-and-running)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Prerequisites

- [Go](https://go.dev/doc/install) 1.26+
- [Templ](https://templ.guide/quick-start/installation/)
- [PostgreSQL](https://www.postgresql.org/download/)
- [Make](https://www.gnu.org/software/make/)

### Install Go

Fantasy FRC is built using Go 1.26+. Current testing against Go 1.26.2.

### Install Templ

A guide to install Templ can be found [here](https://templ.guide/quick-start/installation/).
Make sure you install the Templ Go Tool with `go get -tool github.com/a-h/templ/cmd/templ`

### Install PostgreSQL and Set Up Database

1. Install PostgreSQL
2. Create a new database:
   ```sql
   CREATE DATABASE fantasy_frc;
   ```
3. Connect to the database and run the setup script:
   ```bash
   psql -d fantasy_frc -f database/fantasyFrcDb.sql
   ```
4. Run any additional migration scripts as needed. They can be found in the database directory. 

**Note**: Database versioning will be done in future release.

## Configuration

Create a `.env` file in the `server/` directory with the following variables:

```env
DB_PASSWORD=your_db_password
DB_USERNAME=your_db_username
DB_IP=your_db_host
DB_NAME=fantasy_frc
SERVER_PORT=8080
TBA_TOKEN=your_tba_token
TBA_WEBHOOK_SECRET=your_webhook_secret
METRIC_SECRET=your_metric_secret
```

- `TBA_TOKEN`: Your API token from [The Blue Alliance](https://www.thebluealliance.com/account)
- `DB_*`: Database connection details
- `SERVER_PORT`: Port for the web server (default: 3000)
- `TBA_WEBHOOK_SECRET`: Secret for validating TBA webhook requests
- `METRIC_SECRET`: Secret for metrics endpoint authentication (required)
- `SECURE_HTTP_COOKIE`: Set to `false` for development, `true` for production (default: `true`)
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OpenTelemetry collector endpoint (optional)
- `OTEL_RESOURCE_ATTRIBUTES`: OpenTelemetry resource attributes (optional)

## Building and Running

The Makefile is located in the `server/` directory and includes options to disable certain features during testing:

- `skipScoring=true`: Disables match and team scoring to avoid most TBA API calls during development

### Build and Run

Running for development with verbose logging and live UI updates:
```bash
# Navigate to server directory
cd server

# Run development server with hot reload
make run-verbose
```

Other useful commands:
```bash
# Build CSS only
make watch-css

# Generate templ files
make generate

# Production build
make build

# Build for Linux deployment
make build-linux
```

## Deployment

For production deployment to Linux servers, see [deploy/README.md](deploy/README.md).

## Optional Dependencies

- **Redis**: Used for caching team avatars. If not available, avatars are fetched directly from The Blue Alliance API.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
