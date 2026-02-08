package main

import (
	"fmt"

	"github.com/daviddao/mailbeads/internal/display"
	"github.com/daviddao/mailbeads/internal/types"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done ID [ID...]",
	Short: "Mark triage entries as done",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, id := range args {
			t, err := store.GetTriageByID(id)
			if err != nil {
				display.ErrorMsg("%v", err)
				continue
			}
			if err := store.UpdateTriageStatus(t.ID, types.StatusDone); err != nil {
				display.ErrorMsg("failed to update %s: %v", t.ID, err)
				continue
			}
			display.SuccessMsg("Done: %s %q", t.ID, t.Subject)
		}
		return nil
	},
}

var dismissCmd = &cobra.Command{
	Use:   "dismiss ID [ID...]",
	Short: "Dismiss triage entries (spam/irrelevant)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, id := range args {
			t, err := store.GetTriageByID(id)
			if err != nil {
				display.ErrorMsg("%v", err)
				continue
			}
			if err := store.UpdateTriageStatus(t.ID, types.StatusDismissed); err != nil {
				display.ErrorMsg("failed to update %s: %v", t.ID, err)
				continue
			}
			fmt.Printf("%s Dismissed: %s %q\n", display.Dim.Render("✗"), t.ID, t.Subject)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(dismissCmd)
}
