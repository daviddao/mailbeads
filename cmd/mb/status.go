package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

type statusOutput struct {
	Summary     statusSummary     `json:"summary"`
	HighItems   []statusItem      `json:"high_priority"`
	ActionItems []statusItem      `json:"action_items"`
	SyncState   []statusSyncState `json:"sync_state"`
}

type statusSummary struct {
	TotalEmails int            `json:"total_emails"`
	Threads     int            `json:"threads"`
	Untriaged   int            `json:"untriaged"`
	Pending     int            `json:"pending"`
	Done        int            `json:"done"`
	Dismissed   int            `json:"dismissed"`
	Priority    map[string]int `json:"priority"`
	Ready       int            `json:"ready"`
}

type statusItem struct {
	ID       string `json:"id"`
	ThreadID string `json:"thread_id"`
	Priority string `json:"priority"`
	Account  string `json:"account"`
	Subject  string `json:"subject"`
	Action   string `json:"action"`
	From     string `json:"from_addr,omitempty"`
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

Combines sync state, triage statistics, and high-priority items into a
single view — similar to how 'bd status' shows project issue state or
'git status' shows working tree state.

Use cases:
  - Quick morning inbox check
  - Agent onboarding context
  - Integration with shell prompts

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

		// Triage counts
		triageCounts, err := store.TriageCountByStatus()
		if err != nil {
			return fmt.Errorf("triage counts: %w", err)
		}
		priorityCounts, err := store.TriageCountByPriority()
		if err != nil {
			return fmt.Errorf("priority counts: %w", err)
		}
		untriaged := store.UntriagedCount()
		threads := store.ThreadCount()

		// High-priority items
		highItems, err := store.ListTriage("pending", "", false)
		if err != nil {
			return fmt.Errorf("list high: %w", err)
		}
		var highList []statusItem
		for _, t := range highItems {
			if t.Priority == "high" {
				highList = append(highList, statusItem{
					ID:       t.ID,
					ThreadID: t.ThreadID,
					Priority: t.Priority,
					Account:  t.Account,
					Subject:  t.Subject,
					Action:   t.Action,
					From:     t.From,
				})
			}
		}

		// Ready (actionable) items
		var readyCount int
		var actionItems []statusItem
		if !statusNoReady {
			readyItems, err := store.ReadyTriage("")
			if err == nil {
				readyCount = len(readyItems)
				// Show up to 5 top ready items (non-spam)
				for _, t := range readyItems {
					if t.Priority == "spam" {
						continue
					}
					if len(actionItems) >= 5 {
						break
					}
					actionItems = append(actionItems, statusItem{
						ID:       t.ID,
						ThreadID: t.ThreadID,
						Priority: t.Priority,
						Account:  t.Account,
						Subject:  t.Subject,
						Action:   t.Action,
						From:     t.From,
					})
				}
			}
		}

		summary := statusSummary{
			TotalEmails: totalEmails,
			Threads:     threads,
			Untriaged:   untriaged,
			Pending:     triageCounts["pending"],
			Done:        triageCounts["done"],
			Dismissed:   triageCounts["dismissed"],
			Priority:    priorityCounts,
			Ready:       readyCount,
		}

		// --- Output ---
		if jsonOutput {
			out := statusOutput{
				Summary:     summary,
				HighItems:   highList,
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
		pending := triageCounts["pending"]

		// Build pending breakdown
		detail := ""
		if pending > 0 {
			var parts []string
			for _, p := range []string{"high", "medium", "low"} {
				if c := priorityCounts[p]; c > 0 {
					parts = append(parts, fmt.Sprintf("%d %s", c, p))
				}
			}
			if len(parts) > 0 {
				detail = " (" + strings.Join(parts, ", ") + ")"
			}
		}
		fmt.Printf("    Pending:     %3d%s\n", pending, detail)
		fmt.Printf("    Done:        %3d\n", triageCounts["done"])
		fmt.Printf("    Dismissed:   %3d\n", triageCounts["dismissed"])
		if untriaged > 0 {
			fmt.Printf("    Untriaged:   %s\n", display.ErrStyle.Render(fmt.Sprintf("%3d threads", untriaged)))
		} else {
			fmt.Printf("    Untriaged:     0 %s\n", display.Success.Render("(all triaged)"))
		}
		if !statusNoReady {
			fmt.Printf("    Ready:       %3d actionable\n", readyCount)
		}
		fmt.Println()

		// High-priority items
		if len(highList) > 0 {
			fmt.Printf("  High Priority (%d)\n", len(highList))
			for _, item := range highList {
				id := display.Truncate(item.ID, 8)
				fmt.Printf("    %s %s  %-12s  %s\n",
					display.PriorityDot("high"),
					display.Dim.Render(id),
					display.AccountLabel(item.Account),
					display.Truncate(item.Subject, 45),
				)
				fmt.Printf("      %s  %s\n",
					display.Dim.Render("action:"),
					item.Action,
				)
			}
			fmt.Println()
		}

		// Top actionable items (if different from high)
		if !statusNoReady && len(actionItems) > 0 {
			// Only show this section if there are non-high items to display
			hasNonHigh := false
			for _, item := range actionItems {
				if item.Priority != "high" {
					hasNonHigh = true
					break
				}
			}
			if hasNonHigh || len(highList) == 0 {
				shown := 0
				fmt.Printf("  Next Up\n")
				for _, item := range actionItems {
					// Skip high-priority items already shown above
					if item.Priority == "high" {
						continue
					}
					id := display.Truncate(item.ID, 8)
					fmt.Printf("    %s %s  %-12s  %s\n",
						display.PriorityDot(item.Priority),
						display.Dim.Render(id),
						display.AccountLabel(item.Account),
						display.Truncate(item.Subject, 45),
					)
					fmt.Printf("      %s  %s\n",
						display.Dim.Render("action:"),
						display.Truncate(item.Action, 55),
					)
					shown++
				}
				if readyCount > len(highList)+shown {
					fmt.Printf("    %s\n", display.Dim.Render(fmt.Sprintf("... and %d more (run 'mb ready' to see all)", readyCount-len(highList)-shown)))
				}
				fmt.Println()
			}
		}

		// Hint
		fmt.Printf("  %s\n", display.Dim.Render("Use 'mb inbox' to browse, 'mb show THREAD' to read, 'mb done ID' to clear."))

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusNoReady, "no-ready", false, "Skip ready-item listing (faster)")
	rootCmd.AddCommand(statusCmd)
}
