# Draft Tester

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)

A testing utility for Fantasy FRC Web that creates drafts and simulates picks. Designed for creating drafts for developmentt/tesing and eventually stress testing of the system.

## Features

- Create test drafts programmatically
- Simulate player picks
- Stress test the drafting system
- Generate test data for development

## Installation

### Prerequisites

- [Go](https://go.dev/doc/install) 1.24+

### Build

```bash
cd draftTester
go mod tidy
go build
```

## Usage

### Run Directly

```bash
cd draftTester
go run main.go
```

### Build and Run

```bash
cd draftTester
go build
./draftTester
```

### Configuration

The tool uses the same database configuration as the main Fantasy FRC application. Ensure your `.env` file is set up in the `server/` directory.

## Development

This tool is currently in development and will be expanded to include more comprehensive stress testing features for the Fantasy FRC drafting system.
