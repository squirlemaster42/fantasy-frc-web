# Development Setup Guide

Complete guide for setting up Fantasy FRC development environment.

## 🎯 Overview

This guide covers setting up a complete development environment for Fantasy FRC, including database setup, dependencies, and local development workflow.

## 📋 Prerequisites

### Required Software
- **Go**: Version 1.24+ with toolchain go1.24.0
- **PostgreSQL**: Version 14+
- **Templ**: Template engine for Go
- **Make**: Build tool
- **Git**: Version control

## 🚀 Quick Start

### 1. Clone Repository
```bash
git clone https://github.com/your-org/fantasy-frc-web.git
cd fantasy-frc-web
```

### 2. Install Dependencies
```bash
# Install Go dependencies
go mod download

# Install Templ (if not already installed)
go install github.com/a-h/templ/cmd/templ@latest

# Verify installation
templ version
go version
```

### 3. Database Setup
```bash
# Start PostgreSQL (using Homebrew on macOS)
brew services start postgresql

# Create database
createdb fantasy_frc

# Run schema setup
psql -d fantasy_frc -f database/fantasyFrcDb.sql

# Run migrations
psql -d fantasy_frc -f database/changeUserIdToGuid.sql
psql -d fantasy_frc -f database/etagUpgrade.sql
psql -d fantasy_frc -f database/optInSkip.sql
```

### 4. Environment Configuration
```bash
# Create development environment file
cp server/.env.example server/.env

# Edit with your configuration
nano server/.env
```

Add your development configuration:
```env
DB_PASSWORD=your_dev_password
DB_USERNAME=dev_user
DB_IP=localhost
DB_NAME=fantasy_frc
SERVER_PORT=8080
SESSION_SECRET=dev_session_secret_minimum_32_characters_longer_is_better
TBA_TOKEN=your_tba_dev_token
```

### 5. Build and Run
```bash
# Build and run the application
make
```

## 🔌 Development Workflow

### Project Structure
```
fantasy-frc-web/
├── server/              # Main application code
│   ├── main.go         # Application entry point
│   ├── server.go        # HTTP server setup
│   ├── model/           # Data models
│   ├── handler/         # HTTP handlers
│   ├── draft/           # Draft management
│   ├── scorer/          # Scoring system
│   ├── picking/         # Pick management
│   ├── authentication/   # Auth middleware
│   ├── background/       # Background services
│   ├── tbaHandler/      # TBA API integration
│   ├── utils/           # Utility functions
│   ├── view/            # HTML templates
│   └── assets/          # Static assets
├── database/            # Database schema and migrations
├── docs/               # Documentation
├── draftTester/         # Testing tools
├── fuzzer/             # Fuzzing tools
└── Makefile            # Build configuration
```

### Build Commands
```bash
# Standard build and run
make
```

### Testing
```bash
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

**Solution**: Ensure PostgreSQL is running and credentials match

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

**Solution**: Clean and re-download Go modules

### Port Conflicts

**Problem**: `address already in use`
```bash
# Find process using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>

# Or use different port
SERVER_PORT=8081 make
```

**Solution**: Change port or stop conflicting service

### TBA API Issues

**Problem**: `TBA API validation failed`
```bash
# Test TBA token
curl -H "X-TBA-Auth-Key: your_token" \
     https://www.thebluealliance.com/api/v3/team/frc254

# Check token permissions
# Visit: https://www.thebluealliance.com/account
```

**Solution**: Verify token is valid and has required permissions

## 📊 Development Tools

### Database Management
```bash
# Connect to database
psql -h localhost -U dev_user -d fantasy_frc

# Common queries
\l                    # List databases
\dt                   # List tables
\d table_name          # Describe table
\du                   # List users
```

### API Testing
```bash
# Test authentication
curl -X POST http://localhost:8080/login \
     -H "Content-Type: application/x-www-form-urlencoded" \
     -d "username=test&password=test"

# Test API endpoints
curl -X GET http://localhost:8080/u/home \
     -H "Cookie: session_token=your_session"
```

### Log Monitoring
```bash
# View application logs
tail -f /var/log/fantasy-frc/app.log

# Or if running directly
./server | tee app.log
```

## 🔄 Development Workflow

### 1. Feature Development
```bash
# Create feature branch
git checkout -b feature/new-feature

# Make changes
# ... develop feature ...

# Run tests
go test ./...

# Build and test locally
make

# Commit changes
git add .
git commit -m "Add new feature"
```

### 2. Code Quality
```bash
# Run static analysis
go vet ./...

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

*TODO: Add troubleshooting guide, performance profiling setup, and CI/CD integration instructions*
