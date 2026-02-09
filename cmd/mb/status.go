package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/beads"
	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

type statusOutput struct {
	Summary     statusSummary     `json:"summary"`
	HighItems   []beads.Issue     `json:"high_priority"`
	ActionItems []beads.Issue     `json:"action_items"`
	SyncState   []statusSyncState `json:"sync_state"`
}

type statusSummary struct {
	TotalEmails int `json:"total_emails"`
	Threads     int `json:"threads"`
	Untriaged   int `json:"untriaged"`
	Triaged     int `json:"triaged"`
	BeadsOpen   int `json:"beads_open"`
	BeadsReady  int `json:"beads_ready"`
}

type statusSyncState struct {
	Account  string `json:"account"`
	Emails   int    `json:"emails"`
	LastSync string `json:"last_sync,omitempty"`
}

var statusNoReady bool

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"st"},
	Short:   "Show inbox overview: sync state, triage summary, and high-priority items",
	Long: `Show a quick snapshot of your mailbeads inbox state.

Combines sync state, email statistics, and beads triage items into a
single view. Triage state (priority, status, dependencies) is read from
the beads (.beads/) database.

Examples:
  mb status                # Full status overview
  mb status --no-ready     # Skip ready-item listing (faster)
  mb status --json         # Machine-readable output
  mb st                    # Short alias`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --- Gather data ---

		// Sync state per account
		accounts := store.Accounts()
		syncStates := make([]statusSyncState, 0, len(accounts))
		totalEmails := 0
		for _, acc := range accounts {
			count := store.EmailCountByAccount(acc)
			lastSync := store.LatestFetchedAt(acc)
			syncStates = append(syncStates, statusSyncState{
				Account:  acc,
				Emails:   count,
				LastSync: lastSync,
			})
			totalEmails += count
		}

		untriaged := store.UntriagedCount()
		triaged := store.TriagedCount()
		threads := store.ThreadCount()

		// Get beads data.
		var openIssues []beads.Issue
		var readyIssues []beads.Issue
		hasBd := beads.Available()

		if hasBd {
			var err error
			openIssues, err = beads.List([]string{"email", "triage"}, "open", 50)
			if err != nil {
				openIssues = nil
			}

			if !statusNoReady {
				readyIssues, err = beads.Ready([]string{"email", "triage"}, 20)
				if err != nil {
					readyIssues = nil
				}
			}
		}

		// Split into high-priority and others.
		var highItems []beads.Issue
		for _, issue := range openIssues {
			if issue.Priority <= 1 { // P0 or P1
				highItems = append(highItems, issue)
			}
		}

		// Top actionable items (non-high).
		var actionItems []beads.Issue
		for _, issue := range readyIssues {
			if issue.Priority > 1 && len(actionItems) < 5 {
				actionItems = append(actionItems, issue)
			}
		}

		summary := statusSummary{
			TotalEmails: totalEmails,
			Threads:     threads,
			Untriaged:   untriaged,
			Triaged:     triaged,
			BeadsOpen:   len(openIssues),
			BeadsReady:  len(readyIssues),
		}

		// --- Output ---
		if jsonOutput {
			out := statusOutput{
				Summary:     summary,
				HighItems:   highItems,
				ActionItems: actionItems,
				SyncState:   syncStates,
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		// Terminal display
		display.Header("Mailbeads Status")
		fmt.Println()

		// Sync state
		fmt.Println("  Sync")
		for _, s := range syncStates {
			syncInfo := ""
			if s.LastSync != "" {
				syncInfo = fmt.Sprintf("(last sync: %s)", display.TimeAgo(s.LastSync))
			}
			fmt.Printf("    %-28s %4d emails  %s\n",
				display.AccountLabel(s.Account), s.Emails, display.Dim.Render(syncInfo))
		}
		fmt.Printf("    %s\n", display.Dim.Render(fmt.Sprintf("%d emails across %d threads", totalEmails, threads)))
		fmt.Println()

		// Triage overview
		fmt.Println("  Triage")
		if untriaged > 0 {
			fmt.Printf("    Untriaged:   %s\n", display.ErrStyle.Render(fmt.Sprintf("%3d threads", untriaged)))
		} else {
			fmt.Printf("    Untriaged:     0 %s\n", display.Success.Render("(all triaged)"))
		}
		fmt.Printf("    Triaged:     %3d threads\n", triaged)
		if hasBd {
			fmt.Printf("    Open beads:  %3d issues\n", len(openIssues))
			if !statusNoReady {
				fmt.Printf("    Ready:       %3d actionable\n", len(readyIssues))
			}
		} else {
			fmt.Printf("    %s\n", display.Dim.Render("(bd not found — install from https://beads.sh)"))
		}
		fmt.Println()

		// High-priority items
		if len(highItems) > 0 {
			fmt.Printf("  High Priority (%d)\n", len(highItems))
			for _, issue := range highItems {
				fmt.Printf("    %s %s  %s\n",
					display.PriorityDot("high"),
					display.Dim.Render(issue.ID),
					display.Truncate(issue.Title, 55),
				)
			}
			fmt.Println()
		}

		// Top actionable items
		if !statusNoReady && len(actionItems) > 0 {
			fmt.Printf("  Next Up\n")
			for _, issue := range actionItems {
				pri := beads.PriorityFromBeads(issue.Priority)
				fmt.Printf("    %s %s  %s\n",
					display.PriorityDot(pri),
					display.Dim.Render(issue.ID),
					display.Truncate(issue.Title, 55),
				)
			}
			remaining := len(readyIssues) - len(highItems) - len(actionItems)
			if remaining > 0 {
				fmt.Printf("    %s\n", display.Dim.Render(fmt.Sprintf("... and %d more (run 'mb ready' to see all)", remaining)))
			}
			fmt.Println()
		}

		// Hint
		fmt.Printf("  %s\n", display.Dim.Render("Use 'mb inbox' to browse, 'mb show THREAD' to read, 'mb done BEAD_ID' to clear."))

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusNoReady, "no-ready", false, "Skip ready-item listing (faster)")
	rootCmd.AddCommand(statusCmd)
}
