# mb - Mailbeads

**Email inbox triage for AI agents.** Sync Gmail, analyze threads, surface what needs attention.

**Platforms:** macOS, Linux

[![License](https://img.shields.io/github/license/daviddao/mailbeads)](LICENSE)

Mailbeads syncs Gmail into a local SQLite database and provides a triage workflow for AI agents. When an agent triages a thread, mailbeads creates an issue in [beads](https://github.com/steveyegge/beads) with priority and suggested actions, then stores a slim cross-reference so it can efficiently track which threads have been handled.

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
mb status
```

## Features

- **Gmail Sync:** Fetches emails from multiple Gmail accounts into a local SQLite database. Spam and trash are excluded by default (`--include-spam` to override).
- **Beads Integration:** Triage decisions are stored as [beads](https://github.com/steveyegge/beads) issues with priority, labels, and dependencies. Mailbeads only keeps a slim cross-reference mapping threads to bead IDs.
- **Agent-Optimized:** All commands support `--json` for machine-readable output.
- **Triage Workflow:** Analyze threads, assign priority, suggest actions, track status via beads.
- **Auto-Comments:** When syncing, mailbeads detects threads with new emails since triage and auto-comments on the linked beads issue.
- **Live Stats:** `mb prime` outputs workflow context with live inbox statistics.
- **Pure Go:** Uses `modernc.org/sqlite` (no CGo), builds as a single static binary.

## Essential Commands

| Command | Action |
| --- | --- |
| `mb sync` | Fetch latest emails from Gmail (excludes spam/trash) |
| `mb untriaged` | List threads needing triage |
| `mb show THREAD_ID` | View thread detail with emails and linked bead |
| `mb triage THREAD_ID --action "..." --priority high` | Create triage entry (beads issue + cross-reference) |
| `mb inbox` | List pending triage items from beads, sorted by priority |
| `mb ready` | Show actionable items (open, no blockers) |
| `mb done BEAD_ID` | Close beads issue as done, remove triage cross-reference |
| `mb dismiss BEAD_ID` | Close beads issue as dismissed, remove triage cross-reference |
| `mb status` | Full inbox overview: sync state, triage summary, high-priority items |
| `mb stats` | Show inbox statistics |
| `mb migrate` | Migrate legacy triage entries to real beads issues |

## Agent Integration

```bash
# Get AGENTS.md snippet for your project
mb onboard

# Get AI-optimized workflow context (with live stats)
mb prime

# Full workflow reference
mb prime --full
```

## Architecture

```
Gmail Accounts ──> mb sync ──> .mailbeads/mail.db ──> mb triage ──> .beads/beads.db
                   (Go API)       (emails +             (bd CLI)     (issues with
                                   triage xrefs)                      priority, status,
                                                                      labels, deps)
```

Mailbeads uses a **two-database architecture**:

- **`.mailbeads/mail.db`** (SQLite) — Owned by `mb`. Stores synced emails and thin triage cross-references (`thread_id`, `account`, `bead_id`, `created_at`).
- **`.beads/beads.db`** (SQLite) — Owned by `bd` ([beads](https://github.com/steveyegge/beads)). Stores all triage decisions: priority, status, action, dependencies, comments.

When `mb triage` runs, it creates a beads issue via the `bd` CLI with `email,triage` labels, then stores a cross-reference in `.mailbeads/mail.db`. When `mb done` or `mb dismiss` runs, it closes the beads issue and removes the local cross-reference.

### Triage Cross-Reference Schema

```sql
CREATE TABLE triage (
    thread_id   TEXT NOT NULL,
    account     TEXT NOT NULL,
    bead_id     TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    UNIQUE(thread_id, account)
);
```

### Priority Mapping

| mb priority | Beads numeric | Criteria |
|-------------|---------------|----------|
| **high** | P1 | Direct questions, time-sensitive, approval requests |
| **medium** | P2 (default) | FYI threads, project updates, relevant newsletters |
| **low** | P3 | Receipts, automated confirmations, CI notifications |
| **spam** | P4 | Marketing, cold outreach, unsolicited sales |

## Triage Workflow

```bash
# 1. Sync latest emails (spam/trash excluded by default)
mb sync
# mb sync --include-spam    # include spam and trash

# 2. Find threads needing triage
mb untriaged --json

# 3. Read a thread
mb show THREAD_ID --json

# 4. Create triage entry (creates a beads issue)
mb triage THREAD_ID \
  --action "Reply with agenda" \
  --priority high \
  --suggestion "Time-sensitive meeting request" \
  --category meetings

# 5. Manage inbox (reads from beads)
mb inbox                    # View pending items by priority
mb ready                    # Actionable items (no blockers)
mb done BEAD_ID             # Close as done
mb dismiss BEAD_ID          # Close as dismissed

# 6. Full status overview
mb status
```

### Triage Flags

| Flag | Description |
|------|-------------|
| `--action` | Short action phrase (required) — becomes the beads issue title |
| `--priority` | `high`, `medium`, `low`, `spam` (default: `medium`) |
| `--suggestion` | Detailed suggestion — becomes the beads issue description |
| `--agent-notes` | Agent reasoning notes — appended to beads issue notes |
| `--category` | Category label — added alongside `email,triage` labels |
| `--from` | Sender (auto-detected if omitted) |
| `--epic` | Link to a beads epic (parent dependency) |

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

### Dependencies

| Dependency | Required | Notes |
|------------|----------|-------|
| Go 1.21+ | Yes | No CGo needed (`modernc.org/sqlite`) |
| `bd` (beads) | Yes | [beads.sh](https://beads.sh) — issue tracking CLI for triage state |

## Prerequisites

Mailbeads syncs Gmail using native Go API calls via OAuth 2.0. You need a `credentials.json` file from Google Cloud.

### Getting `credentials.json`

#### 1. Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or select an existing one)
3. Navigate to **APIs & Services** > **Library**
4. Search for **Gmail API** and click **Enable**

#### 2. Configure the OAuth Consent Screen

1. Go to **APIs & Services** > **OAuth consent screen**
2. Select **External** user type, click **Create**
3. Fill in the app name and your email
4. Add scopes:
   - `https://www.googleapis.com/auth/gmail.readonly`
   - `https://www.googleapis.com/auth/gmail.modify`
5. Under **Test users**, add the Gmail address you want to sync

#### 3. Create OAuth Credentials

1. Go to **APIs & Services** > **Credentials**
2. Click **Create Credentials** > **OAuth client ID**
3. Select **Desktop app** as the application type
4. Click **Create**, then **Download JSON**
5. Rename the downloaded file to `credentials.json`

#### 4. Set Up the Account Directory

Place the credentials in a directory named after your email address, at the project root:

```
your-project/
  user@gmail.com/
    credentials.json      <- put it here
  another@example.com/    <- add more accounts the same way
    credentials.json
  .mailbeads/
    mail.db               <- emails + triage cross-references
  .beads/
    beads.db              <- triage state (priority, status, actions)
```

Mailbeads auto-discovers accounts by scanning for `*/credentials.json` directories.

#### 5. First Sync

Run `mb sync` — it will open your browser for OAuth consent on first use. A `token.json` file is saved next to `credentials.json` for future use.

```bash
mb sync
# Browser opens -> sign in -> grant access -> done
```

> **Security:** Never commit `credentials.json` or `token.json`. They are in `.gitignore` by default.

### Migrating from Legacy Schema

If you have an older mailbeads installation with the fat triage table (inline priority, action, status columns), run:

```bash
mb migrate              # Migrate legacy triage entries to beads issues
mb migrate --dry-run    # Preview without making changes
```

The migration creates real beads issues for each legacy entry and updates the cross-references.

## License

MIT
