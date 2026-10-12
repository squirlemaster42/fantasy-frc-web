# Development Setup Guide

Complete guide for setting up a Fantasy FRC development environment.

## 🎯 Overview

This guide covers setting up a complete development environment for Fantasy FRC, including database setup, dependencies, and local development workflow.

## 📋 Prerequisites

### Required Software
- **Go**: Version 1.26+ (the server module declares `go 1.26.5`)
- **PostgreSQL**: Version 16+
- **Redis**: Version 7+ (required for avatars and rate limiting)
- **Templ**: Template engine for Go
- **Make**: Build tool
- **Git**: Version control
- **goose**: Migration CLI (`go install github.com/pressly/goose/v3/cmd/goose@latest`)

## 🚀 Quick Start

### 1. Clone Repository
```bash
git clone https://github.com/your-org/fantasy-frc-web.git
cd fantasy-frc-web
```

### 2. Install Dependencies
```bash
# Navigate to server directory
cd server

# Install Go dependencies
go mod download

# Install Templ (if not already installed)
go get -tool github.com/a-h/templ/cmd/templ

# Verify installation
go tool templ version
go version
```

### 3. Database Setup
```bash
# Start PostgreSQL (using Homebrew on macOS)
brew services start postgresql

# Create database
createdb fantasy_frc

# Install goose if you haven't already
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations
cd ../database
make up
```

### 4. Environment Configuration
```bash
# From the workspace root
cp server/.env.example server/.env

# Edit with your preferred editor
vim server/.env  # or nano, code, etc.
```

Add your development configuration:
```env
DB_PASSWORD=your_dev_password
DB_USERNAME=dev_user
DB_IP=localhost
DB_NAME=fantasy_frc
SERVER_PORT=8080
TBA_TOKEN=your_tba_dev_token
TBA_WEBHOOK_SECRET=your_webhook_secret
METRIC_SECRET=your_metric_secret
CSRF_SECRET=your_csrf_secret
SECURE_HTTP_COOKIE=false
REDIS_ADDR=localhost:6379
```

`CSRF_SECRET` is required. `SECURE_HTTP_COOKIE` defaults to `true` if omitted; set it to `false` for local development over HTTP.

### 5. Build and Run
```bash
# Navigate to server directory
cd server

# Run development server with hot reload and verbose logging
make run-verbose
```

`make run-verbose` starts the templ proxy on `http://127.0.0.1:7331` for hot-reloaded UI updates. The Go server runs on the `SERVER_PORT` you configured.

## 🔌 Development Workflow

### Project Structure
```
fantasy-frc-web/
├── server/              # Main application code
│   ├── main.go          # Application entry point
│   ├── server.go        # HTTP server setup
│   ├── defaults.go      # Server-level env defaults
│   ├── model/           # Data models and Store interfaces
│   ├── handler/         # HTTP handlers
│   ├── draft/           # Draft actor and state machine
│   ├── scorer/          # Scoring system
│   ├── picking/         # WebSocket pick notifier
│   ├── authentication/  # Auth service and middleware
│   ├── background/      # Cleanup service and draft daemon
│   ├── middleware/      # CSRF, rate limiting, security headers
│   ├── cache/           # Redis avatar store
│   ├── metrics/         # Prometheus metrics
│   ├── tbaHandler/      # TBA API integration
│   ├── utils/           # Utility functions
│   ├── view/            # HTML templates
│   ├── assets/          # Static assets
│   ├── database/        # DB connection helpers
│   ├── log/             # Structured logging
│   ├── otel/            # OpenTelemetry tracer setup
│   ├── discord/         # Discord webhook bus
│   ├── swagger/         # Generated TBA API models
│   └── types/           # Shared view/page data types
├── database/            # Goose migrations and migration tooling
├── docs/                # Documentation
├── draftAgent/          # AI draft automation tool
├── draftTester/         # Testing tools
├── fuzzer/              # Fuzzing tools
└── infra/               # Infrastructure, Ansible, and K8s manifests
```

### Build Commands
```bash
# Navigate to server directory first
cd server

# Run development server with hot reload
make run-verbose

# Watch CSS only
make watch-css

# Generate templ files
make generate

# Generate repository mocks
make mocks

# Run linters
make lint

# Production build
make build

# Build for Linux deployment
make build-linux
```

### Testing
```bash
# Navigate to server directory first
cd server

# Run all tests
go test ./...

# Run tests for specific package
go test ./model
go test ./scorer
go test ./utils

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...
```

## 🐛 Common Development Issues

### Database Connection Issues

**Problem**: `database connection failed`
```bash
# Check PostgreSQL is running
brew services list | grep postgresql

# Check connection details
psql -h localhost -U dev_user -d fantasy_frc

# Verify environment variables
cat server/.env
```

**Solution**: Ensure PostgreSQL is running and credentials match.

### Redis Connection Issues

**Problem**: Server exits on startup with an avatar-store error.
```bash
# Check Redis is running
redis-cli ping
```

**Solution**: Redis is effectively required. Start a local Redis instance or point `REDIS_ADDR` at a reachable server.

### Go Module Issues

**Problem**: `module not found` or version conflicts
```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download

# Verify module
go mod verify
```

**Solution**: Clean and re-download Go modules.

### Port Conflicts

**Problem**: `address already in use`
```bash
# Find process using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>

# Or use a different port (from server directory)
cd server && SERVER_PORT=8081 make run-verbose
```

**Solution**: Change `SERVER_PORT` or stop the conflicting service. Note that `make run-verbose` also uses port `7331` for the templ proxy.

### TBA API Issues

**Problem**: `TBA API validation failed`
```bash
# Test TBA token
curl -H "X-TBA-Auth-Key: your_token" \
     https://www.thebluealliance.com/api/v3/team/frc254

# Check token permissions
# Visit: https://www.thebluealliance.com/account
```

**Solution**: Verify token is valid and has required permissions.

## 📊 Development Tools

### Database Management
```bash
# Connect to database
psql -h localhost -U dev_user -d fantasy_frc

# Common queries
\l                    # List databases
\dt                   # List tables
\d table_name         # Describe table
\du                   # List users
```

### Migration Commands
```bash
cd database

# Apply pending migrations
make up

# Check status
make status

# Rollback one migration
make down

# Test full up/down cycle in Docker
make test

# Create a new migration
make create name=add_my_feature
```

## 🔄 Development Workflow

### 1. Feature Development
```bash
# Create feature branch
git checkout -b feature/new-feature

# Make changes
# ... develop feature ...

# Run tests (from server directory)
cd server
go test ./...

# Build and test locally
make run-verbose

# Commit changes
git add .
git commit -m "Add new feature"
```

### 2. Code Quality
```bash
# Run static analysis
go vet ./...

# Run linters
make lint

# Format code (follow existing style)
# Note: Don't use go fmt, follow project style

# Run tests with coverage
go test -cover ./...
```

### 3. Pre-commit Checklist
- [ ] All tests pass
- [ ] Code follows project style
- [ ] Documentation updated
- [ ] No sensitive data committed
- [ ] Build completes successfully

---

*Last updated: 2026-09-04*

*TODO: Add troubleshooting guide, performance profiling setup, and CI/CD integration instructions*
