package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

var (
	untriagedAccount string
	untriagedLimit   int
)

var untriagedCmd = &cobra.Command{
	Use:   "untriaged",
	Short: "List threads without triage entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		threads, err := store.UntriagedThreads(untriagedAccount, untriagedLimit)
		if err != nil {
			return fmt.Errorf("query untriaged: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(threads)
		}

		if len(threads) == 0 {
			fmt.Println("All threads triaged.")
			return nil
		}

		fmt.Printf("Untriaged threads (%d):\n\n", len(threads))
		fmt.Printf("  %-16s %-12s %-40s %6s %s\n",
			display.Dim.Render("THREAD"),
			display.Dim.Render("ACCOUNT"),
			display.Dim.Render("SUBJECT"),
			display.Dim.Render("EMAILS"),
			display.Dim.Render("LATEST"),
		)
		for _, t := range threads {
			fmt.Printf("  %-16s %-12s %-40s %6d %s\n",
				display.Truncate(t.ThreadID, 16),
				display.AccountLabel(t.Account),
				display.Truncate(t.Subject, 40),
				t.EmailCount,
				display.TimeAgo(t.LatestDate),
			)
		}
		return nil
	},
}

func init() {
	untriagedCmd.Flags().StringVar(&untriagedAccount, "account", "", "Filter by account")
	untriagedCmd.Flags().IntVarP(&untriagedLimit, "limit", "n", 50, "Max results")
	rootCmd.AddCommand(untriagedCmd)
}
