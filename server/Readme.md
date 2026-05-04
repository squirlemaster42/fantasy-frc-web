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

   Required variables: `DB_PASSWORD`, `DB_USERNAME`, `DB_IP`, `DB_NAME`, `SERVER_PORT`, `TBA_TOKEN`, `TBA_WEBHOOK_SECRET`, `METRIC_SECRET`, `SECURE_HTTP_COOKIE`

3. **Run database migrations:**
   ```bash
   make migrate
   ```

## Development

**Run development server with hot reload:**
```bash
make run-verbose
```

This starts the Go server with verbose logging and watches for changes to templ files and CSS.

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

**Install binary to deploy location:**
```bash
make install
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

## Notes

- Tailwind CSS CLI is automatically downloaded if not present
- Production build creates a static binary with embedded assets
