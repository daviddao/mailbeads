# AGENTS.md — Mailbeads Coding Agent Instructions

## Project Overview

`mb` (mailbeads) is a Go CLI tool for email inbox triage, designed for AI agents.
It syncs Gmail into SQLite, lets agents create triage entries, and surfaces
actionable items. Inspired by [beads](https://github.com/steveyegge/beads).

## Repository Structure

```
cmd/mb/           # Cobra CLI commands (one file per command)
internal/
  auth/           # Google OAuth2 authentication (reads Python-format tokens)
  gmail/          # Native Go Gmail API client (search, read)
  db/             # SQLite database layer (schema.go, db.go)
  display/        # Lipgloss terminal formatting
  sync/           # Gmail sync using native Go API calls
  types/          # Core data types (Email, Triage, Thread)
scripts/          # Install script
```

## Build & Run

```bash
# Build
go build -o mb ./cmd/mb

# Run
./mb init                           # Initialize .mailbeads/
./mb sync                           # Fetch emails
./mb untriaged --json               # List untriaged threads
./mb triage THREAD_ID --action "..." --priority high
./mb inbox                          # View inbox
./mb stats                          # Statistics

# All commands
./mb --help
```

**Requirements:** Go 1.21+. No CGo needed (uses `modernc.org/sqlite`).

## Code Style

### File Structure
Every command file in `cmd/mb/` follows this pattern:
```go
package main

import (
    "fmt"
    "github.com/daviddao/mailbeads/internal/display"
    "github.com/spf13/cobra"
)

var someFlag string

var someCmd = &cobra.Command{
    Use:   "command",
    Short: "One-line description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Use global `store` for DB access
        // Use global `jsonOutput` for --json flag
        // Use global `quietFlag` for --quiet flag
        return nil
    },
}

func init() {
    someCmd.Flags().StringVar(&someFlag, "flag", "", "Description")
    rootCmd.AddCommand(someCmd)
}
```

### Conventions
- **Module path:** `github.com/daviddao/mailbeads`
- **Naming:** Go standard — `camelCase` locals, `PascalCase` exports
- **DB access:** Global `store *db.DB` set by `PersistentPreRunE` in `main.go`
- **JSON output:** Check `jsonOutput` flag, use `json.NewEncoder(cmd.OutOrStdout())`
- **Styling:** Use `internal/display` for terminal formatting (lipgloss)
- **Errors:** Return `fmt.Errorf(...)` from `RunE`, cobra handles display

### Adding a New Command
1. Create `cmd/mb/newcomm.go`
2. Define `var newCmd = &cobra.Command{...}` with `RunE`
3. In `func init()`, add flags and `rootCmd.AddCommand(newCmd)`
4. If the command doesn't need the DB, add its name to the switch in `main.go`'s `PersistentPreRunE`

### Database
- Schema in `internal/db/schema.go` — tables: `emails`, `triage`, `triage_deps`
- All CRUD in `internal/db/db.go`
- IDs are random 16-char hex strings
- Triage supports partial ID matching (prefix search)
- Null strings use `sql.NullString` for scanning, `nullStr()` helper for writing

### Gmail Integration
- `internal/auth/auth.go` — OAuth2 layer that reads Python-format `token.json` files (compatible with both Go and Python tools)
- `internal/gmail/gmail.go` — Native Go Gmail API client (search messages, read full content with MIME decode)
- `internal/sync/sync.go` — Uses native Go Gmail API calls (no Python dependency)
- `cmd/mb/gmail.go` — CLI commands: `mb gmail search QUERY`, `mb gmail read MESSAGE_ID`
- Credentials expected at `./ACCOUNT_EMAIL/credentials.json` relative to project root
- Accounts are auto-discovered from `*/credentials.json` directories in the project root
- Incremental sync by default (since last known email date)

## Testing

No formal test suite yet. Verify manually:

```bash
go build -o mb ./cmd/mb
./mb version
./mb quickstart
./mb prime
./mb stats
```

## Security

**NEVER commit:**
- `*/credentials.json` — OAuth client secrets
- `*/token.json` — OAuth access/refresh tokens
- `.mailbeads/` — Contains synced email data

These are all in `.gitignore`.
