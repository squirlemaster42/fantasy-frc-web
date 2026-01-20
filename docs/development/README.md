# Development Documentation

Complete guides for Fantasy FRC development environment, testing, and contribution.

## 📚 Documentation

### [🔧 Development Setup](./setup.md)
Complete development environment setup:
- Prerequisites and tool installation
- Database setup and migrations
- Environment configuration
- Docker development environment
- IDE configuration and debugging

### [🧪 Testing Strategy](./testing.md) *(Coming Soon)*
Testing guidelines and best practices:
- Unit testing patterns
- Integration testing approach
- Database testing helpers
- Test data management
- Coverage requirements

### [🤝 Contributing Guidelines](./contributing.md) *(Coming Soon)*
Contribution process and standards:
- Code style guidelines
- Pull request process
- Documentation requirements
- Testing requirements
- Code review criteria

## 🎯 Development Workflow

### 1. Environment Setup
- Install required tools (Go, PostgreSQL, Templ)
- Configure development database
- Set up environment variables
- Verify installation with test run

### 2. Development Process
- Create feature branch from main
- Implement changes with test coverage
- Run tests and ensure quality
- Update documentation as needed
- Submit pull request for review

### 3. Quality Assurance
- Follow project coding standards
- Use provided testing helpers
- Ensure all tests pass
- Verify documentation accuracy
- Check for security considerations

## 🔧 Development Tools

### Required Software
- **Go 1.24+**: Main programming language
- **PostgreSQL 12+**: Database server
- **Templ**: Template engine for HTML
- **Git**: Version control system
- **Docker**: Containerization (optional)

### Recommended Tools
- **VS Code**: IDE with Go extensions
- **Postico**: PostgreSQL database client
- **Postman**: API testing tool
- **Make**: Build automation

### Development Extensions
- **Go extension**: Language support and debugging
- **Templ extension**: Template syntax highlighting
- **PostgreSQL extension**: Database connection and queries
- **GitLens**: Git integration and history

## 📋 Code Standards

### Style Guidelines
- Follow existing code formatting (not go fmt)
- Use descriptive variable names
- Add context to error messages
- Include comments for complex logic
- Use provided assert package for errors

### Testing Requirements
- Write tests for all new functions
- Use testify assertions
- Test error conditions
- Use provided test helpers
- Maintain good test coverage

### Documentation Requirements
- Update relevant documentation
- Include examples in docstrings
- Add TODO comments for future work
- Document any breaking changes

## 🔄 Build Process

### Development Build
```bash
# Standard development build
make
```

### Testing Build
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Production Build
```bash
# Build for production
GOOS=linux GOARCH=amd64 go build -o fantasy-frc-linux ./server

# Build Docker image
docker build -t fantasy-frc:latest .
```

## 📊 Development Metrics

### Code Quality Targets
- **Test Coverage**: >80% for new code
- **Build Time**: <30 seconds for clean build
- **Test Execution**: <2 minutes for full suite
- **Lint Pass**: No critical issues

### Performance Targets
- **Startup Time**: <5 seconds
- **Memory Usage**: <256MB for development
- **Database Connections**: <10 for development
- **API Response**: <100ms for local calls

## 🔗 Related Documentation

- [Architecture Overview](../architecture/system-overview.md) - System design
- [Database Schema](../database/schema.md) - Data structure
- [Web Endpoints](../api/web-endpoints.md) - HTTP endpoints and forms
- [WebSocket API](../api/websocket-api.md) - Real-time notifications
- [Deployment Guide](../deployment/configuration.md) - Environment setup

## 🚀 Quick Start

1. **Clone Repository**: `git clone https://github.com/your-org/fantasy-frc-web.git`
2. **Install Dependencies**: `go mod download && make setup`
3. **Setup Database**: Follow database setup instructions
4. **Configure Environment**: Copy and edit `.env` file
5. **Run Application**: `make` and visit http://localhost:8080
6. **Run Tests**: `go test ./...` to verify setup

## 🤝 Contribution Types

### Bug Fixes
- Identify issue in existing code
- Create reproduction case
- Implement fix with tests
- Update documentation if needed
- Submit pull request

### New Features
- Design feature with documentation
- Implement with test coverage
- Update API documentation
- Consider backward compatibility
- Submit for review

### Documentation
- Improve existing documentation
- Add missing examples
- Fix broken links
- Update diagrams
- Enhance clarity

### Infrastructure
- Improve build processes
- Add development tooling
- Enhance testing setup
- Update deployment scripts
- Optimize performance

---

*Development documentation focuses on enabling efficient, high-quality contribution to the Fantasy FRC project.*
