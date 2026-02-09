package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

var (
	primeFullMode bool
)

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output AI-optimized workflow context",
	Long: `Output essential mailbeads workflow context in AI-optimized markdown format.

Two modes:
- Default: Brief workflow reminders for agents that already know mb (~30 lines)
- --full:  Complete workflow reference with examples (~80 lines)

Designed for Claude Code hooks and agent session start to provide
context about the email triage workflow.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if primeFullMode {
			return outputFullContext(cmd.OutOrStdout())
		}
		return outputBriefContext(cmd.OutOrStdout())
	},
}

func init() {
	primeCmd.Flags().BoolVar(&primeFullMode, "full", false, "Output full workflow reference (for new agents)")
	rootCmd.AddCommand(primeCmd)
}

// outputBriefContext outputs a concise reminder for agents that know mb.
func outputBriefContext(w io.Writer) error {
	// If we have a store (DB found), include live stats.
	statsBlock := ""
	if store != nil {
		untriaged := store.UntriagedCount()
		triaged := store.TriagedCount()
		totalEmails := store.EmailCount()
		statsBlock = fmt.Sprintf(`
## Current State
- Emails in DB: %d
- Triaged threads: %d
- Untriaged threads: %d
`,
			totalEmails, triaged, untriaged)
	}

	context := `# Mailbeads (mb) — Email Triage Active

Triage state is stored in beads (.beads/) — mb creates beads issues via bd.
` + statsBlock + `
## Workflow
1. ` + "`mb sync`" + ` — fetch latest emails
2. ` + "`mb untriaged --json`" + ` — find threads needing triage
3. ` + "`mb show THREAD_ID --json`" + ` — read thread detail
4. ` + "`mb triage THREAD_ID --action \"...\" --priority high`" + ` — create beads issue
5. ` + "`mb triage THREAD_ID --action \"...\" --epic bd-XXXX`" + ` — link to epic
6. ` + "`mb ready --json`" + ` — check actionable items (from beads)
7. ` + "`mb done BEAD_ID`" + ` / ` + "`mb dismiss BEAD_ID`" + ` — close beads issue

## Rules
- All commands support ` + "`--json`" + ` for machine-readable output
- Priority: high, medium, low, spam (mapped to beads P1-P4)
- done/dismiss take bead IDs (e.g., cowork-abc), not triage IDs
- ` + "`--epic`" + ` links triage to a beads epic via parent-child dependency
- Accounts are auto-discovered from */credentials.json in the project root

Run ` + "`mb prime --full`" + ` for complete workflow reference with examples.
`
	_, err := fmt.Fprint(w, context)
	return err
}

// outputFullContext outputs the complete workflow reference.
func outputFullContext(w io.Writer) error {
	// Include live stats if DB available.
	statsBlock := ""
	if store != nil {
		untriaged := store.UntriagedCount()
		triaged := store.TriagedCount()
		totalEmails := store.EmailCount()
		threads := store.ThreadCount()

		statsBlock = fmt.Sprintf(`
## Current Inbox State
- %d emails across %d threads
- %d triaged, %d untriaged
`, totalEmails, threads, triaged, untriaged)
	}

	context := `# Mailbeads (mb) — Full Workflow Reference

Email inbox triage tool for AI agents. Syncs Gmail into SQLite, creates
beads issues for triage decisions, surfaces actionable items via bd ready.

**Architecture:** mb owns email storage (.mailbeads/), beads owns work tracking (.beads/).
When you triage a thread, mb creates a beads issue with label:email,triage and
stores a cross-reference locally.
` + statsBlock + `
## Accounts

Accounts are auto-discovered from ` + "`*/credentials.json`" + ` directories in the project root.
Use ` + "`--account user@example.com`" + ` to target a specific account.

## Complete Workflow

### Step 1: Sync emails
` + "```bash" + `
mb sync                              # All discovered accounts
mb sync --account user@example.com   # Single account
mb sync --full                       # Force 72h re-scan
` + "```" + `

### Step 2: Find untriaged threads
` + "```bash" + `
mb untriaged --json                  # All untriaged
mb untriaged --account gain -n 10    # Filter + limit
` + "```" + `

### Step 3: Read thread detail
` + "```bash" + `
mb show THREAD_ID --json             # Full thread with emails + bead info
mb show THREAD_ID --no-body          # Headers only (less context)
` + "```" + `

### Step 4: Create triage entries (beads issues)
` + "```bash" + `
mb triage THREAD_ID \
  --action "Reply with agenda" \
  --priority high \
  --suggestion "Time-sensitive meeting request" \
  --agent-notes "Sender is team lead, context: Q1 planning" \
  --category "meetings" \
  --epic bd-a3f8                     # Link to beads epic
` + "```" + `

Required: THREAD_ID (positional), --action
Auto-detected: --account (if thread in only one account), subject, sender
Optional: --priority (default: medium), --suggestion, --agent-notes, --category, --epic, --from

### Step 5: Manage inbox
` + "```bash" + `
mb inbox --json                      # Open beads issues with email label
mb ready --json                      # Actionable (not blocked in beads DAG)
mb done BEAD_ID                      # Close beads issue (done)
mb dismiss BEAD_ID                   # Close beads issue (dismissed)
mb done ID1 ID2 ID3                  # Batch operations
mb stats --json                      # Full statistics
` + "```" + `

## Priority Mapping

| mb priority | beads priority | Criteria |
|-------------|---------------|----------|
| **high**    | P1            | Direct questions, time-sensitive, approval requests |
| **medium**  | P2            | FYI threads, project updates, relevant newsletters |
| **low**     | P3            | Receipts, automated confirmations, CI notifications |
| **spam**    | P4            | Marketing, cold outreach, unsolicited sales |

## Beads Integration

- Triage creates beads issues with labels ` + "`email,triage`" + `
- ` + "`--epic`" + ` creates a parent-child dependency in the beads DAG
- ` + "`mb done`" + ` / ` + "`mb dismiss`" + ` closes the beads issue
- Use ` + "`bd dep add`" + ` / ` + "`bd show`" + ` for advanced dependency management
- Email metadata stored in beads issue notes (thread_id, account, from)

## Reporting

After triaging, provide a summary:
` + "```" + `
Inbox Triage Summary:
- X threads triaged (Y high, Z medium, W low, V spam)
- High priority items:
  1. [Subject] from [Sender] — [Action]
` + "```" + `
`
	_, err := fmt.Fprint(w, context)
	return err
}
