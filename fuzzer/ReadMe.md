# Draft Fuzzer

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)

A fuzzing tool for Fantasy FRC Web that generates randomized match JSON data. Used to test webhook functionality, match scoring accuracy, and system performance under load.

## Features

- Generate fuzzy match JSON payloads
- Test webhook integration with The Blue Alliance
- Validate match scoring logic
- Stress test system with high-volume data
- Create realistic test scenarios

## Installation

### Prerequisites

- [Go](https://go.dev/doc/install) 1.24+

### Build

```bash
cd fuzzer
go mod tidy
go build
```

## Usage

### Run Directly

```bash
cd fuzzer
go run .
```

### Build and Run

```bash
cd fuzzer
go build
./fuzzer
```

### Configuration

The fuzzer can be configured to generate different types of match data. Check the `example.json` file for sample output format.

## Output

Generates JSON files containing randomized match data that can be used to:

- Test webhook endpoints
- Validate scoring calculations
- Simulate high-volume match updates
- Debug scoring edge cases

## Development

This tool helps ensure the reliability of the Fantasy FRC scoring system by providing comprehensive test data for webhook processing and match scoring. 
