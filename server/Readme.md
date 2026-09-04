## Fantasy FRC Server

Written in Go with Templ and HTMX.

For detailed development guidelines, build commands, and code style, see the project [`AGENTS.md`](../AGENTS.md).

## Setup

1. **Install dependencies:**
   ```bash
   go mod download
   ```

2. **Set up environment variables:**
   ```bash
   cp .env.example .env  # or manually create .env with required variables
   ```

   Required variables: `DB_PASSWORD`, `DB_USERNAME`, `DB_IP`, `DB_NAME`, `SERVER_PORT`, `TBA_TOKEN`, `TBA_WEBHOOK_SECRET`, `METRIC_SECRET`, `CSRF_SECRET`.

3. **Run database migrations:**
   ```bash
   cd ../database
   make up
   ```

   Migrations are managed with [goose](https://github.com/pressly/goose). See [`../database/README.md`](../database/README.md) for all migration commands.

## Development

**Run development server with hot reload:**
```bash
make run-verbose
```

This starts the [templ proxy](https://templ.guide/commands-and-tools/proxy/) on `http://127.0.0.1:7331` for hot-reloaded UI updates, plus the Go server on `SERVER_PORT`.

**Watch CSS only:**
```bash
make watch-css
```

## Building

**Build CSS:**
```bash
make build-css
```

**Generate templ files:**
```bash
make generate
```

**Generate repository mocks:**
```bash
make mocks
```

**Run linters:**
```bash
make lint
```

**Production build:**
```bash
make build
```

**Build for Linux deployment:**
```bash
make build-linux
```

**Full production build (CSS + generate + build):**
```bash
make prod
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./model
go test ./scorer
go test ./utils

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...
```

## Using Docker

```bash
# Build the container
docker build -t fantasyfrc .

# Run the container
docker run --env-file <env-file> --add-host=host.docker.internal:host-gateway -p <external-port>:<server-port> fantasyfrc
```

The internal listening port is controlled by `SERVER_PORT` in your env file, not by the `EXPOSE` directive.

## Notes

- Tailwind CSS CLI is automatically downloaded if not present.
- Production build creates a static binary with embedded assets.
- Tests create test accounts. To use these you need to set `MIN_PASSWORD_LENGTH` shorter in your env file.
