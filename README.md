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
- [Testing](#testing)
- [Building and Running](#building-and-running)
- [Deployment](#deployment)
- [Optional Dependencies](#optional-dependencies)
- [License](#license)

## Installation

### Prerequisites

- [Go](https://go.dev/doc/install) 1.26+
- [Templ](https://templ.guide/quick-start/installation/)
- [PostgreSQL](https://www.postgresql.org/download/) 16+
- [Redis](https://redis.io/download) 7+ (required for avatars and rate limiting)
- [Make](https://www.gnu.org/software/make/)

### Install Go

Fantasy FRC is built using Go 1.26+. The server module declares `go 1.26.5`.

### Install Templ

A guide to install Templ can be found [here](https://templ.guide/quick-start/installation/).
Make sure you install the Templ Go Tool with `go get -tool github.com/a-h/templ/cmd/templ`.

### Install PostgreSQL and Set Up Database

1. Install PostgreSQL 16+.
2. Create a new database:
   ```sql
   CREATE DATABASE fantasy_frc;
   ```
3. Install the [goose](https://github.com/pressly/goose) migration CLI:
   ```bash
   go install github.com/pressly/goose/v3/cmd/goose@latest
   ```
4. Run migrations from the `database/` directory:
   ```bash
   cd database
   make up
   ```

The `database/` directory contains all goose migrations. See [database/README.md](database/README.md) for details.

## Configuration

Create a `.env` file in the `server/` directory with at least the required variables:

```env
DB_PASSWORD=your_db_password
DB_USERNAME=your_db_username
DB_IP=your_db_host
DB_NAME=fantasy_frc
SERVER_PORT=8080
TBA_TOKEN=your_tba_token
TBA_WEBHOOK_SECRET=your_webhook_secret
METRIC_SECRET=your_metric_secret
CSRF_SECRET=your_csrf_secret
```

- `DB_*`: Database connection details.
- `SERVER_PORT`: Port for the web server (required, no default).
- `TBA_TOKEN`: API token for The Blue Alliance.
- `TBA_WEBHOOK_SECRET`: Secret for validating TBA webhook HMAC signatures.
- `METRIC_SECRET`: Bearer token required to access `/metrics`.
- `CSRF_SECRET`: Secret used to sign CSRF tokens (required).
- `SECURE_HTTP_COOKIE`: Set to `false` for local development, `true` for production (default: `true`).
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OpenTelemetry collector endpoint (optional).
- `OTEL_RESOURCE_ATTRIBUTES`: OpenTelemetry resource attributes (optional).

Many additional optional tuning variables are documented in [AGENTS.md](AGENTS.md#environment-variables).

## Testing

The project uses repository interfaces with auto-generated mocks for unit testing. See the full [Testing Guide](docs/development/README.md#testing-with-repository-mocks) for details on writing handler tests without a database.

```bash
cd server

# Run all tests
go test ./...

# Generate mocks after interface changes
make mocks

# Run specific handler test
go test ./handler -run TestHandleViewDraftProfile -v
```

## Building and Running

Fantasy FRC uses `make` (run from the `server/` directory) for running the app. The binary supports a few command-line flags:

- `-skipScoring=true`: Disables match and team scoring to avoid most TBA API calls during development.
- `-v`: Enables verbose (debug) logging.
- `-log-format=text`: Emits logs in text instead of JSON.

### Build and Run

Running for development with verbose logging and live UI updates:

```bash
# Navigate to server directory
cd server

# Run development server with hot reload
make run-verbose
```

`make run-verbose` starts the [templ proxy](https://templ.guide/commands-and-tools/proxy/) on `http://127.0.0.1:7331` for hot-reloaded UI updates. The raw Go app also runs on the `SERVER_PORT` you configured.

Other useful commands:

```bash
# Build CSS only
make build-css

# Watch CSS for changes
make watch-css

# Generate templ files
make generate

# Run linters
make lint

# Production build
make build
```

## Deployment

For deployment to a Kubernetes cluster, see [infra/ansible/README.md](infra/ansible/README.md).

## Optional Dependencies

- **Redis**: Used for caching team avatars and distributed HTTP rate limiting. The server fails fast on startup if Redis is unreachable, so it is effectively required.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for more details.
