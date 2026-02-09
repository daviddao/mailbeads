package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daviddao/mailbeads/internal/beads"
	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

var (
	inboxAccount  string
	inboxPriority string
	inboxAll      bool
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "List pending triage items from beads, sorted by priority",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !beads.Available() {
			return fmt.Errorf("bd (beads) CLI not found on PATH")
		}

		labels := []string{"email", "triage"}
		status := "open"
		if inboxAll {
			status = ""
		}

		issues, err := beads.List(labels, status, 50)
		if err != nil {
			return fmt.Errorf("query beads: %w", err)
		}

		// Filter by priority if specified.
		if inboxPriority != "" {
			filtered := issues[:0]
			for _, issue := range issues {
				if beads.PriorityFromBeads(issue.Priority) == inboxPriority {
					filtered = append(filtered, issue)
				}
			}
			issues = filtered
		}

		// Filter by account if specified (match against notes which contain "account=X").
		if inboxAccount != "" {
			filtered := issues[:0]
			for _, issue := range issues {
				if strings.Contains(issue.Notes, "account="+inboxAccount) ||
					strings.Contains(issue.Notes, inboxAccount) {
					filtered = append(filtered, issue)
				}
			}
			issues = filtered
		}

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(issues)
		}

		if len(issues) == 0 {
			fmt.Println("Inbox clear.")
			return nil
		}

		label := "pending"
		if inboxAll {
			label = "total"
		}
		fmt.Printf("Inbox (%d %s):\n\n", len(issues), label)

		for _, issue := range issues {
			pri := beads.PriorityFromBeads(issue.Priority)
			fmt.Printf("  %s %s  %s  %s\n",
				display.PriorityDot(pri),
				display.Dim.Render(issue.ID),
				display.PriorityLabel(pri),
				display.Dim.Render(issue.Title),
			)
		}
		return nil
	},
}

func init() {
	inboxCmd.Flags().StringVar(&inboxAccount, "account", "", "Filter by account (partial match)")
	inboxCmd.Flags().StringVar(&inboxPriority, "priority", "", "Filter by priority")
	inboxCmd.Flags().BoolVar(&inboxAll, "all", false, "Include closed/dismissed")
	rootCmd.AddCommand(inboxCmd)
}
