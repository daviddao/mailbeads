package main

import (
	"encoding/json"
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
	// If we have a store (DB found), include live stats
	statsBlock := ""
	if store != nil {
		triageCounts, err := store.TriageCountByStatus()
		if err == nil {
			untriaged := store.UntriagedCount()
			totalEmails := store.EmailCount()
			statsBlock = fmt.Sprintf(`
## Current State
- Emails in DB: %d
- Untriaged threads: %d
- Pending triage: %d | Done: %d | Dismissed: %d
`,
				totalEmails, untriaged,
				triageCounts["pending"], triageCounts["done"], triageCounts["dismissed"])
		}
	}

	context := `# Mailbeads (mb) — Email Triage Active
` + statsBlock + `
## Workflow
1. ` + "`mb sync`" + ` — fetch latest emails
2. ` + "`mb untriaged --json`" + ` — find threads needing triage
3. ` + "`mb show THREAD_ID --json`" + ` — read thread detail
4. ` + "`mb triage THREAD_ID --action \"...\" --priority high`" + ` — write triage
5. ` + "`mb ready --json`" + ` — check actionable items
6. ` + "`mb done ID`" + ` / ` + "`mb dismiss ID`" + ` — manage status

## Rules
- All commands support ` + "`--json`" + ` for machine-readable output
- Priority: high, medium, low, spam
- Triage IDs support partial matching
- Accounts are auto-discovered from */credentials.json in the project root

Run ` + "`mb prime --full`" + ` for complete workflow reference with examples.
`
	_, err := fmt.Fprint(w, context)
	return err
}

// outputFullContext outputs the complete workflow reference.
func outputFullContext(w io.Writer) error {
	// Include live stats if DB available
	statsBlock := ""
	if store != nil {
		triageCounts, err := store.TriageCountByStatus()
		if err == nil {
			priorityCounts, _ := store.TriageCountByPriority()
			untriaged := store.UntriagedCount()
			totalEmails := store.EmailCount()
			threads := store.ThreadCount()

			statsJSON, _ := json.MarshalIndent(map[string]any{
				"total_emails": totalEmails,
				"threads":      threads,
				"untriaged":    untriaged,
				"triage":       triageCounts,
				"priority":     priorityCounts,
			}, "", "  ")

			statsBlock = fmt.Sprintf(`
## Current Inbox State
`+"```json"+`
%s
`+"```"+`
`, string(statsJSON))
		}
	}

	context := `# Mailbeads (mb) — Full Workflow Reference

Email inbox triage tool for AI agents. Syncs Gmail into SQLite, lets agents
create triage entries, surfaces actionable items in the dashboard.
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
mb show THREAD_ID --json             # Full thread with emails + triage
mb show THREAD_ID --no-body          # Headers only (less context)
` + "```" + `

### Step 4: Create triage entries
` + "```bash" + `
mb triage THREAD_ID \
  --action "Reply with agenda" \
  --priority high \
  --suggestion "Time-sensitive meeting request" \
  --agent-notes "Sender is team lead, context: Q1 planning" \
  --category "meetings"
` + "```" + `

Required: THREAD_ID (positional), --action
Auto-detected: --account (if thread in only one account), subject, sender, email count
Optional: --priority (default: medium), --suggestion, --agent-notes, --category, --from

### Step 5: Manage inbox
` + "```bash" + `
mb inbox --json                      # Pending items by priority
mb ready --json                      # Actionable (not snoozed/blocked)
mb done TRIAGE_ID                    # Mark done
mb dismiss TRIAGE_ID                 # Dismiss spam
mb done ID1 ID2 ID3                  # Batch operations
mb stats --json                      # Full statistics
` + "```" + `

## Priority Guidelines

| Priority | Criteria |
|----------|----------|
| **high** | Direct questions, time-sensitive, approval requests, meeting follow-ups |
| **medium** | FYI threads, project updates, relevant newsletters |
| **low** | Receipts, automated confirmations, CI notifications |
| **spam** | Marketing, cold outreach, unsolicited sales |

## Reporting

After triaging, provide a summary:
` + "```" + `
Inbox Triage Summary:
- X threads triaged (Y high, Z medium, W low, V spam)
- High priority items:
  1. [Subject] from [Sender] — [Action]
` + "```" + `

Focus on high-priority items. Use ` + "`mb stats --json`" + ` for machine-readable summary.
`
	_, err := fmt.Fprint(w, context)
	return err
}
