# mb - Mailbeads

**Email inbox triage for AI agents.** Sync Gmail, analyze threads, surface what needs attention.

**Platforms:** macOS, Linux

[![License](https://img.shields.io/github/license/daviddao/mailbeads)](LICENSE)

Mailbeads provides a persistent, structured email triage system for coding agents. It syncs Gmail into a local SQLite database, lets agents analyze threads and create triage entries with priority and suggested actions, and surfaces actionable items in a dashboard.

Inspired by [beads](https://github.com/steveyegge/beads).

## Quick Start

```bash
# Install (requires Go 1.21+)
curl -fsSL https://raw.githubusercontent.com/daviddao/mailbeads/main/scripts/install.sh | bash

# Initialize in your project
cd your-project
mb init

# Sync emails and check inbox
mb sync
mb untriaged
mb stats
```

## Features

- **Gmail Sync:** Fetches emails from multiple Gmail accounts into a local SQLite database.
- **Agent-Optimized:** All commands support `--json` for machine-readable output.
- **Triage Workflow:** Analyze threads, assign priority, suggest actions, track status.
- **Partial ID Matching:** `mb done abc` matches triage IDs starting with "abc".
- **Live Stats:** `mb prime` outputs workflow context with live inbox statistics.
- **Pure Go:** Uses `modernc.org/sqlite` (no CGo), builds as a single static binary.

## Essential Commands

| Command | Action |
| --- | --- |
| `mb sync` | Fetch latest emails from Gmail |
| `mb untriaged` | List threads needing triage |
| `mb show THREAD_ID` | View thread detail with emails |
| `mb triage THREAD_ID --action "..." --priority high` | Create triage entry |
| `mb inbox` | List pending triage items by priority |
| `mb ready` | Show actionable items (not snoozed/blocked) |
| `mb done ID` | Mark triage entry as done |
| `mb dismiss ID` | Dismiss spam/irrelevant |
| `mb stats` | Show inbox statistics |

## Agent Integration

```bash
# Get AGENTS.md snippet for your project
mb onboard

# Get AI-optimized workflow context (with live stats)
mb prime

# Full workflow reference
mb prime --full
```

## Triage Workflow

```bash
# 1. Sync latest emails
mb sync

# 2. Find threads needing triage
mb untriaged --json

# 3. Read a thread
mb show THREAD_ID --json

# 4. Create triage entry
mb triage THREAD_ID \
  --action "Reply with agenda" \
  --priority high \
  --suggestion "Time-sensitive meeting request"

# 5. Manage inbox
mb inbox                    # View pending items
mb done TRIAGE_ID           # Mark done
mb dismiss TRIAGE_ID        # Dismiss spam
```

## Priority Levels

| Priority | Criteria |
|----------|----------|
| **high** | Direct questions, time-sensitive, approval requests |
| **medium** | FYI threads, project updates, relevant newsletters |
| **low** | Receipts, automated confirmations, CI notifications |
| **spam** | Marketing, cold outreach, unsolicited sales |

## Installation

### One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/daviddao/mailbeads/main/scripts/install.sh | bash
```

### Go install

```bash
go install github.com/daviddao/mailbeads/cmd/mb@latest
```

### Build from source

```bash
git clone https://github.com/daviddao/mailbeads.git
cd mailbeads
go build -o mb ./cmd/mb
sudo mv mb /usr/local/bin/
```

## Prerequisites

1. **Go 1.21+** for building
2. **Gmail OAuth credentials** at `./ACCOUNT_EMAIL/credentials.json` relative to your project root

Accounts are auto-discovered from `*/credentials.json` directories in the project root.

## How It Works

```
Gmail Accounts ──> mb sync ──> .mailbeads/mail.db ──> mb triage ──> Dashboard
                   (Go API)       (SQLite)              (Go CLI)     (Next.js)
```

- **`.mailbeads/mail.db`** — SQLite database storing emails and triage entries
- **`mb init`** creates the database and adds `.mailbeads/` to `.gitignore`
- **`mb sync`** fetches emails using native Go Gmail API calls
- **`mb gmail search/read`** — search and read emails directly via Gmail API
- **Triage entries** are created by AI agents analyzing thread content
- **Frontend** reads from the database for the Inbox sidebar

## License

MIT
