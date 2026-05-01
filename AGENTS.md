# AGENTS.md - Development Guidelines for Fantasy FRC Web

This file contains build/lint/test commands and code style guidelines for agentic coding assistants working on this Go, templ, Htmx, Postgres web application.

## Build/Lint/Test Commands

### Building and Running
```bash
# Build and run the application
make
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
# Vet code for potential issues
go vet ./...

For formatting, follow the format of the rest of the code, do not use Go's built in formatter.
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
See `database/README.md` for full details.

## Code Style Guidelines

### General Conventions

- **Go Version**: Go 1.24.0 with toolchain go1.24.0
- **Logging**: Use custom logging with context
- **Error Handling**: Use custom `assert` package for context-aware error handling
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

- Use the custom `assert` package for database operations and critical paths
- Provide context when creating assertions: `assert := assert.CreateAssertWithContext("Function Name")`
- Add context to assertions: `assert.AddContext("User ID", userId)`
- Use `slog` for non-critical errors and informational logging
- Return errors from functions when appropriate, especially for model operations

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
    Interval    int          // Number of seconds to pick
    StartTime   time.Time
    EndTime     time.Time
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
