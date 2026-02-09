package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/beads"
	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

var readyAccount string

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List actionable triage items from beads (open, no blockers)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !beads.Available() {
			return fmt.Errorf("bd (beads) CLI not found on PATH")
		}

		labels := []string{"email", "triage"}
		issues, err := beads.Ready(labels, 20)
		if err != nil {
			return fmt.Errorf("query beads: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(issues)
		}

		if len(issues) == 0 {
			fmt.Println("Nothing actionable right now.")
			return nil
		}

		fmt.Printf("Ready (%d actionable):\n\n", len(issues))
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
	readyCmd.Flags().StringVar(&readyAccount, "account", "", "Filter by account")
	rootCmd.AddCommand(readyCmd)
}
