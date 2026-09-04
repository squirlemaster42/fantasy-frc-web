# AGENTS.md - Development Guidelines for Fantasy FRC Web

This file contains build/lint/test commands and code style guidelines for agentic coding assistants working on this Go, templ, Htmx, Postgres web application

## Build/Lint/Test Commands

### Building and Running
```bash
# Build the application
make build

# Run the application (after building)
./server

# Run the development server with hot reload
make run-verbose
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./model
go test ./scorer
go test ./utils

# Run a specific test
go test ./model -run TestGetDraftsForUser
go test ./scorer -run TestSortMatchOrder

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...
```

### Linting and Formatting
```bash
# Run all linters (go vet and golangci-lint)
make lint

# Vet code for potential issues
go vet ./...

# Run golangci-lint
go tool golangci-lint run ./...

For formatting, follow the format of the rest of the code, do not use Go's built in formatter
```

### Dependencies
```bash
# Download dependencies
go mod download

# Tidy up dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Database Migrations

Migrations are managed with [goose](https://github.com/pressly/goose) in the `database/` directory.

```bash
# Install goose CLI
go install github.com/pressly/goose/v3/cmd/goose@latest

# Create a new migration
cd database && make create name=description_here

# Run pending migrations locally
cd database && make up

# Check migration status
cd database && make status

# Rollback one migration
cd database && make down

# Test full up/down cycle in Docker
cd database && make test
```

Migrations are **not** tied to the server application. They run manually or as a K8s Job.
See `database/README.md` for full details

## Code Style Guidelines

### General Conventions

- **Go Version**: Go 1.26+ (the server module declares `go 1.26.5`)
- **Logging**: Use the custom `server/log` package for structured logging with context
- **Error Handling**: Use custom `assert` package for context-aware error handling. Only to be used for behavior which should theoretically never happen. Other errors should be logged using the logging pattern used throughout the entire project with the appropriate log level.
- **Testing**: Use `github.com/stretchr/testify/assert` for assertions

### Import Organization

```go
import (
    // Standard library imports
    "database/sql"
    "errors"
    "fmt"
    "log/slog"
    "strings"
    "time"

    // Third-party imports
    "github.com/google/uuid"
    "github.com/joho/godotenv"
    "github.com/labstack/echo/v4"
    "github.com/stretchr/testify/assert"

    // Local imports
    "server/assert"
    "server/database"
    "server/model"
)
```

### Error Handling

- Provide context when creating assertions: `assert := assert.CreateAssertWithContext("Function Name")`
- Add context to assertions: `assert.AddContext("User ID", userId)`
- Use the `server/log` package (which wraps `slog`) for non-critical errors and informational logging
- Return errors from functions when appropriate, especially for model operations

#### Custom `assert` Package

The `server/assert` package is for *invariant* conditions — states that should be impossible if the code, schema, and data are correct (e.g., a SQL update affecting zero rows when exactly one was expected, an internal map being nil, a schema/statement preparation failure). It is **not** for user input errors, authentication failures, missing resources, or transient database/network issues.

`RunAssert`, `NoError`, `AssertCF`, and `NoErrorCF` call `log.Fatal` and crash the process on failure. This is intentional fail-fast behavior. The project follows principles aligned with NASA software safety guidance (see [NASA-STD-8719.13C — Software Safety Standard](https://standards.nasa.gov/standard/nasa/nasa-std-871913)): detect unsafe or inconsistent state as early as possible, prevent propagation of bad state, and return to a known-good state on restart.

Because the application runs in Kubernetes, a momentary container crash is acceptable. Kubernetes will restart the pod and the system will resume from a clean state. A transient request may fail, but the server will not continue operating on corrupt or inconsistent data.

Guidelines:

- Use the custom `assert` package only for conditions that should theoretically never happen.
- **Never use `RunAssert`, `NoError`, `AssertCF`, or `NoErrorCF` in authentication hot paths or user-facing handlers** — always return errors gracefully for invalid user input.
- For database operations, `database.Prepare` already classifies errors and only crashes on schema/syntax/statement SQLSTATE classes (`42xxx`, `22xxx`, `26xxx`). Transient failures are returned as errors.
- If an `assert` call can be triggered by user action, bad input, or recoverable data state, convert it to a returned error instead.

### Database Operations

- Always use prepared statements for SQL queries
- Use `sql.NullString`, `sql.NullInt16`, etc. for nullable fields
- Return pointers to structs for optional results (e.g., `*[]DraftModel`)
- Use transactions for multi-step database operations
- Log database errors with appropriate context

### Struct Definitions

```go
type DraftModel struct {
    Id          int           // PascalCase field names
    DisplayName string        // Use descriptive names
    Description string        // Include comments for complex fields
    Owner       User
    Status      DraftState
    Players     []DraftPlayer
    NextPick    DraftPlayer
}
```

### Constants and Enums

```go
type DraftState string

const (
    FILLING           DraftState = "Filling"
    WAITING_TO_START  DraftState = "Waiting to Start"
    PICKING           DraftState = "Picking"
    TEAMS_PLAYING     DraftState = "Teams Playing"
    COMPLETE          DraftState = "Complete"
)
```

### Function Signatures

- Return errors as the last return value for functions that can fail
- Use pointer receivers for methods that modify the struct
- Return pointers for large structs to avoid copying
- Use descriptive parameter names

```go
func GetDraft(database *sql.DB, draftId int) (DraftModel, error)
func (d *DraftModel) String() string
func CreateDraft(database *sql.DB, draft *DraftModel) int
```

### Testing Patterns

- Use descriptive test names: `TestGetDraftsForUser`, `TestSortMatchOrder`
- Create helper functions for test setup: `CreateDBConnection`, `GetOrCreateUser`
- Use testify assertions: `assert.Equal`, `assert.True`, `assert.NoError`
- Load environment variables in test setup
- Clean up test data appropriately

### Comments and Documentation

- Add comments for exported functions and complex logic
- Use TODO comments for future improvements: `// TODO: Add validation`
- Document struct fields when the purpose isn't obvious
- Add context to error messages for better debugging

### Security Considerations

- Never log sensitive information (passwords, tokens, etc.)
- Use environment variables for configuration
- Validate user input before database operations
- Use prepared statements to prevent SQL injection
- Implement proper authentication and authorization

### Environment Variables

The server fails fast on startup if a required variable is missing or if an optional variable is set to a malformed value (e.g., a non-bool value for a bool setting or a non-numeric `SERVER_PORT`).

#### Required

- `TBA_TOKEN` (string): API token for The Blue Alliance.
- `DB_PASSWORD` (string): PostgreSQL password.
- `DB_USERNAME` (string): PostgreSQL username.
- `DB_IP` (string): PostgreSQL host.
- `DB_NAME` (string): PostgreSQL database name.
- `SERVER_PORT` (int): Port the HTTP server listens on (must be 1–65535).
- `TBA_WEBHOOK_SECRET` (string): Secret used to verify TBA webhook HMAC signatures.
- `METRIC_SECRET` (string): Secret required to access the metrics endpoint.
- `CSRF_SECRET` (string): Secret used to sign CSRF tokens.

#### Optional

- `TRUST_PROXY` (bool, default `false`): When `true`, configures the Echo server to extract the client IP from the `X-Forwarded-For` header (required when running behind a reverse proxy such as nginx or a Kubernetes ingress). When `false` (default), the server uses the direct connection IP. **Never set to `true` unless the application is behind a trusted proxy**, otherwise clients can spoof their IP address.

- `ALLOWED_ORIGIN` (string, required when `TRUST_PROXY=true`): The exact origin allowed for WebSocket connections (e.g., `https://fantasy-frc.example.com`). When set, the server validates the `Origin` header on WebSocket upgrade requests against this value. **Must be set in production when `TRUST_PROXY=true`** to prevent cross-origin WebSocket abuse. When `TRUST_PROXY` is `false` and `ALLOWED_ORIGIN` is unset, the server falls back to allowing same-origin and `localhost` requests for local development.

- `SECURE_HTTP_COOKIE` (bool, default `true`): Whether session and CSRF cookies are sent with the `Secure` flag.

- `RATE_LIMIT_ENABLED` (bool, default `true`): Whether HTTP rate limiting is enabled.

- `RATE_LIMIT_POSTS_PER_MINUTE` (int64, default `100`): Per-minute POST rate limit when rate limiting is enabled.

- `MIN_PASSWORD_LENGTH` (int, default `12`): Minimum password length for new registrations.

- `MIN_USERNAME_LENGTH` (int, default `3`): Minimum username length.

- `MAX_USERNAME_LENGTH` (int, default `32`): Maximum username length.

- `USERNAME_ALLOWED_SPECIAL_CHARS` (string, default `_-`): Characters allowed in usernames in addition to letters and digits.

- `BCRYPT_COST` (int, default `14`): bcrypt cost factor for password hashing. Values outside the bcrypt range are replaced with the default.

- `REDIS_ADDR` (string, default `localhost:6379`): Redis server address. If unset, the server falls back to `localhost:6379`.

- `REDIS_PASSWORD` (string, default empty): Redis password.

- `REDIS_RATE_LIMIT_DB` (int, default `1`): Redis database number for rate-limit data.

- `REDIS_AVATAR_DB` (int, default `2`): Redis database number for avatar cache data.

- `DRAFT_ACTOR_CACHE_SIZE` (int, default `128`): Maximum number of draft actors to keep in memory. When the cache is full, the least-recently-used actor is shut down and evicted.

- `PICK_WINDOWS_CONFIG_FILE` (string, default `../config/pick-windows.json` relative to the `server/` working directory): Path to a JSON file that configures how long each pick lasts and the allowed pick windows per weekday. If the file is missing, built-in defaults are used. If the file is present but malformed, the server fails fast on startup. Example format:
  ```json
  {
    "pick_time": "1h",
    "windows": {
      "Sunday":    {"start_hour": 8,  "end_hour": 22},
      "Monday":    {"start_hour": 17, "end_hour": 22},
      "Tuesday":   {"start_hour": 17, "end_hour": 22},
      "Wednesday": {"start_hour": 17, "end_hour": 22},
      "Thursday":  {"start_hour": 17, "end_hour": 22},
      "Friday":    {"start_hour": 17, "end_hour": 22},
      "Saturday":  {"start_hour": 8,  "end_hour": 22}
    }
  }
  ```

- `DB_MAX_OPEN_CONNS` (int, default `90`): Maximum open database connections.
- `DB_MAX_IDLE_CONNS` (int, default `25`): Maximum idle database connections.
- `DB_CONN_MAX_LIFETIME` (duration, default `30m`): Maximum lifetime of a database connection.
- `SESSION_CLEANUP_LEEWAY_HOURS` (int, default `2`): How far before a session's expiration time the cleanup service removes it.
- `CLEANUP_INTERVAL_MINUTES` (int, default `60`): Minutes between cleanup service runs.
- `DRAFT_DAEMON_TICK_INTERVAL` (duration, default `1m`): How often the draft daemon checks for expired/skipped picks.
- `DRAFT_DAEMON_TICK_TIMEOUT` (duration, default `55s`): Per-tick context timeout for the draft daemon.
- `RATE_LIMIT_LOGIN_ATTEMPTS` (int64, default `5`): Login attempts allowed per `RATE_LIMIT_AUTH_WINDOW`.
- `RATE_LIMIT_REGISTER_ATTEMPTS` (int64, default `3`): Registration attempts allowed per `RATE_LIMIT_AUTH_WINDOW`.
- `RATE_LIMIT_AUTH_WINDOW` (duration, default `15m`): Sliding window for login/register rate limits.
- `RATE_LIMIT_REDIS_PING_TIMEOUT` (duration, default `2s`): Redis availability check timeout when initializing rate limiting.
- `HSTS_MAX_AGE_SECONDS` (int, default `63072000`): `Strict-Transport-Security` max-age in seconds.
- `SESSION_TOKEN_BYTES` (int, default `16`): Number of random bytes in a session token.
- `SESSION_EXPIRATION_DAYS` (int, default `10`): Number of days until a session token expires.
- `AVATAR_CACHE_TTL` (duration, default `672h`): Redis TTL for cached team avatars.
- `AVATAR_HTTP_CACHE_MAX_AGE_SECONDS` (int, default `604800`): `Cache-Control` max-age for HTTP avatar responses.
- `DRAFT_ACTOR_INBOX_BUFFER` (int, default `100`): Capacity of each draft actor's inbox channel.
- `DRAFT_ACTOR_REQUEST_TIMEOUT` (duration, default `5s`): Timeout when posting to or receiving a reply from a draft actor.
- `PICK_NOTIFIER_QUEUE_BUFFER` (int, default `10`): Capacity of each WebSocket watcher notification channel.
- `PICK_NOTIFIER_SEND_TIMEOUT` (duration, default `5s`): Timeout when sending a notification to a watcher.
- `DISCORD_WEBHOOK_TIMEOUT` (duration, default `15s`): HTTP client timeout for Discord webhook calls.
- `DISCORD_PREMATCH_QUEUE_BUFFER` (int, default `100`): Capacity of the Discord pre-match notification channel.
- `DISCORD_MIN_ID_LENGTH` (int, default `17`): Minimum valid Discord snowflake length.
- `TBA_ALLIANCE_MAX_RETRIES` (int, default `5`): Retries when TBA returns an empty elimination alliance list.
- `TBA_ALLIANCE_BACKOFF_BASE` (duration, default `1s`): Base duration for exponential backoff between alliance retries.
- `SCORER_QUAL_WIN_POINTS` (int, default `3`): Points awarded to the winning alliance of a qualification match.
- `SCORER_ENERGIZED_BONUS_POINTS` (int, default `1`): Bonus points for the Energized ranking point.
- `SCORER_SUPERCHARGED_BONUS_POINTS` (int, default `1`): Bonus points for the Supercharged ranking point.
- `SCORER_TRAVERSAL_BONUS_POINTS` (int, default `2`): Bonus points for the Traversal ranking point.
- `SCORER_PLAYOFF_FINALS_POINTS` (int, default `18`): Points for a finals playoff match win.
- `SCORER_PLAYOFF_UPPER_BRACKET_POINTS` (int, default `15`): Points for an upper-bracket semifinals playoff match win.
- `SCORER_PLAYOFF_LOWER_BRACKET_POINTS` (int, default `9`): Points for a lower-bracket semifinals playoff match win.
- `SCORER_EINSTEIN_MULTIPLIER` (int, default `2`): Multiplier applied to playoff points at Einstein.
- `SCORER_ALLIANCE_PICK_MULTIPLIER` (int, default `2`): Multiplier applied to alliance selection base scores.
- `LEADERBOARD_PER_PAGE` (int, default `25`): Number of leaderboard entries per page.
- `WS_READ_BUFFER_SIZE` (int, default `1024`): WebSocket upgrader read buffer size.
- `WS_WRITE_BUFFER_SIZE` (int, default `1024`): WebSocket upgrader write buffer size.
- `WS_READ_TIMEOUT` (duration, default `120s`): WebSocket read/pong deadline.
- `WS_PING_INTERVAL` (duration, default `30s`): WebSocket ping ticker interval.
- `WS_WRITE_TIMEOUT` (duration, default `10s`): WebSocket write/control-frame deadline.
- `TBA_WEBHOOK_MAX_BODY_BYTES` (int64, default `1048576`): Maximum TBA webhook body size.
- `TBA_UPCOMING_MATCH_TEAM_COUNT` (int, default `6`): Expected number of teams in an upcoming match notification.
- `TBA_EVENT_CODES` (string, comma-separated, default `arc,cur,dal,gal,hop,joh,mil,new,cmptx`): Championship event codes for the current season.
- `TBA_WEBHOOK_SECRET_FILE` (string, default `./webhookSecret.txt`): Path to the TBA webhook verification file.
- `STATIC_ASSET_MAX_AGE_SECONDS` (int, default `2592000`): `Cache-Control` max-age for static assets.
- `SERVER_SHUTDOWN_TIMEOUT` (duration, default `10s`): Graceful shutdown timeout.
- `METRICS_ACTIVE_USER_TICK_INTERVAL` (duration, default `10s`): How often active-user gauges are updated.
- `METRICS_QUERY_ID_MAX_LENGTH` (int, default `100`): Maximum length of a query ID metric label before truncation.
- `DB_QUERY_THRESHOLD_MS` (int, default `50`): Minimum mean query time (ms) for a statement to appear in metrics.
- `DB_QUERY_POLL_INTERVAL` (duration, default `30s`): How often database query stats are collected.
- `DB_QUERY_MAX_COUNT` (int, default `50`): Maximum number of queries returned by the database stats collector.
