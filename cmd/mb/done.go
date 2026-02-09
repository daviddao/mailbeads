package main

import (
	"fmt"

	"github.com/daviddao/mailbeads/internal/beads"
	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done BEAD_ID [BEAD_ID...]",
	Short: "Mark triage entries as done (closes the beads issue)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !beads.Available() {
			return fmt.Errorf("bd (beads) CLI not found on PATH")
		}
		for _, id := range args {
			if err := beads.Close(id, "done"); err != nil {
				display.ErrorMsg("close %s: %v", id, err)
				continue
			}
			// Clean up local cross-reference.
			store.DeleteTriageRef(id)
			display.SuccessMsg("Done: %s", id)
		}
		return nil
	},
}

var dismissCmd = &cobra.Command{
	Use:   "dismiss BEAD_ID [BEAD_ID...]",
	Short: "Dismiss triage entries (closes as spam/irrelevant)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !beads.Available() {
			return fmt.Errorf("bd (beads) CLI not found on PATH")
		}
		for _, id := range args {
			if err := beads.Close(id, "dismissed — spam/irrelevant"); err != nil {
				display.ErrorMsg("dismiss %s: %v", id, err)
				continue
			}
			// Clean up local cross-reference.
			store.DeleteTriageRef(id)
			fmt.Printf("%s Dismissed: %s\n", display.Dim.Render("✗"), id)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(dismissCmd)
}
