# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Before Committing

After making code changes, run these checks before committing:

```bash
gofmt -w .         # Format code
go vet ./...       # Check for issues
go build .         # Verify it compiles
```

## Build & Run Commands

```bash
# Build binary
go build -o dist/dinero-backup .

# Run from source
go run .

# Run specific command
go run . run --invoices --dry-run
go run . test-connection
go run . state
```

## Project Overview

CLI tool for backing up data from [Dinero](https://dinero.dk) ERP. Uses OAuth2 for authentication against Dinero's API.

## Architecture

### Package Structure

- **main.go** - CLI entry point using Cobra. Defines commands (`run`, `state`, `test-connection`) and flags.
- **dinero/** - API client for Dinero. Handles OAuth2 authentication and HTTP requests.
- **backup/** - Backup logic for each data type. Each file exports a `Backup<Type>()` function.
- **state/** - Persists sync state to `state.json` for incremental backups.

### Key Patterns

**Backup functions** follow a consistent signature:
```go
func Backup<Type>(client *dinero.Client, stateManager *state.Manager, outDir string, dryRun bool) error
```

**Incremental sync**: Each backup type tracks its last sync timestamp in `state.json`. On subsequent runs, only changes since `lastSync` are fetched using the API's `changesSince` parameter.

**Entries special case**: The entries backup uses two different API endpoints:
- First run per year: `/entries` (includes primo/opening balance)
- Subsequent runs: `/entries/changes` (incremental updates, chunked in 31-day windows)
- Changes are merged into existing year files by `EntryGuid`

### Configuration

Requires environment variables (via `.env` or exported):
- `CLIENT_ID`, `CLIENT_SECRET` - OAuth credentials from Dinero
- `API_KEY` - Personal integration key
- `ORG_ID` - Dinero organization ID
- `OUT_DIR` - Output directory (optional, defaults to "output")
