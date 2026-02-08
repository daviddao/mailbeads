package main

import (
	"encoding/json"
	"fmt"

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
	Short: "List pending triage items sorted by priority",
	RunE: func(cmd *cobra.Command, args []string) error {
		status := "pending"
		if inboxAll {
			status = ""
		}
		items, err := store.ListTriage(status, inboxAccount, inboxAll)
		if err != nil {
			return fmt.Errorf("query inbox: %w", err)
		}

		// Filter by priority if specified
		if inboxPriority != "" {
			var filtered = items[:0]
			for _, t := range items {
				if t.Priority == inboxPriority {
					filtered = append(filtered, t)
				}
			}
			items = filtered
		}

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(items)
		}

		if len(items) == 0 {
			fmt.Println("Inbox clear.")
			return nil
		}

		label := "pending"
		if inboxAll {
			label = "total"
		}
		fmt.Printf("Inbox (%d %s):\n\n", len(items), label)

		for _, t := range items {
			id := display.Truncate(t.ID, 8)
			fmt.Printf("  %s %s  %s  %-12s  %-35s  %s\n",
				display.PriorityDot(t.Priority),
				display.Dim.Render(id),
				display.PriorityLabel(t.Priority),
				display.AccountLabel(t.Account),
				display.Truncate(t.Subject, 35),
				display.Dim.Render(t.Action),
			)
		}
		return nil
	},
}

func init() {
	inboxCmd.Flags().StringVar(&inboxAccount, "account", "", "Filter by account (partial match)")
	inboxCmd.Flags().StringVar(&inboxPriority, "priority", "", "Filter by priority")
	inboxCmd.Flags().BoolVar(&inboxAll, "all", false, "Include done/dismissed")
	rootCmd.AddCommand(inboxCmd)
}
