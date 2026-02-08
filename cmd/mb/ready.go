package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

var readyAccount string

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List actionable triage items (not snoozed, not blocked)",
	RunE: func(cmd *cobra.Command, args []string) error {
		items, err := store.ReadyTriage(readyAccount)
		if err != nil {
			return fmt.Errorf("query ready: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(items)
		}

		if len(items) == 0 {
			fmt.Println("Nothing actionable right now.")
			return nil
		}

		fmt.Printf("Ready (%d actionable):\n\n", len(items))
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
	readyCmd.Flags().StringVar(&readyAccount, "account", "", "Filter by account")
	rootCmd.AddCommand(readyCmd)
}
