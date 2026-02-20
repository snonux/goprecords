# AGENTS.md - AI Agent Instructions

## Project Overview

`goprecords` is a Go command-line program that generates uptime reports for hosts based on input record files from `uptimed`. It supports importing records into SQLite and querying for reports, or reporting directly from a stats directory.

## Build & Test Commands

```bash
# Build
go build -o goprecords ./cmd/goprecords
# or: mage

# Run tests
go test ./...
# or: mage test

# Run integration tests
./goprecords test

# Install to GOPATH/bin
mage install
```

## Project Structure

```
goprecords/
├── cmd/goprecords/main.go    # Entry point, CLI handling
├── internal/
│   ├── goprecords/           # Core logic (db, aggregate, report, parse, order, types)
│   └── version/version.go    # Version constant
├── Magefile.go               # Build automation
├── fixtures/                 # Test fixtures and expected outputs
└── go.mod                    # Go 1.21, modernc.org/sqlite
```

## Code Style

- Go 1.21
- No comments in code unless explicitly requested
- Functions should be ~30 lines; refactor when approaching 50 lines
- Move code from main.go to ./internal package when main.go grows too large
- Follow existing patterns in the codebase

## Categories & Metrics

- Categories: `Host`, `Kernel`, `KernelMajor`, `KernelName`
- Metrics: `Boots`, `Uptime`, `Score`, `Downtime`, `Lifespan`
- Output formats: `Plaintext`, `Markdown`, `Gemtext`
- `Downtime` and `Lifespan` metrics only apply to `Host` category

## Key Types

- `Category`, `Metric`, `OutputFormat` - enums for report configuration
- `Aggregate` - per-entity stats (host, kernel, etc.)
- `HostAggregate` - extends Aggregate with LastKernel, Downtime, Lifespan
