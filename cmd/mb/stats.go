package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/beads"
	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

type statsOutput struct {
	Emails     map[string]accountStats `json:"emails"`
	Untriaged  int                     `json:"untriaged"`
	Triaged    int                     `json:"triaged"`
	TotalEmail int                     `json:"total_emails"`
	Threads    int                     `json:"threads"`
	BeadsOpen  int                     `json:"beads_open,omitempty"`
}

type accountStats struct {
	Count    int    `json:"count"`
	LastSync string `json:"last_sync,omitempty"`
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show inbox statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		accounts := store.Accounts()
		emailStats := make(map[string]accountStats)
		totalEmails := 0
		for _, acc := range accounts {
			count := store.EmailCountByAccount(acc)
			lastSync := store.LatestFetchedAt(acc)
			emailStats[acc] = accountStats{Count: count, LastSync: lastSync}
			totalEmails += count
		}

		untriaged := store.UntriagedCount()
		triaged := store.TriagedCount()
		threads := store.ThreadCount()

		// Get beads open count if available.
		beadsOpen := 0
		if beads.Available() {
			issues, err := beads.List([]string{"email", "triage"}, "open", 0)
			if err == nil {
				beadsOpen = len(issues)
			}
		}

		if jsonOutput {
			out := statsOutput{
				Emails:     emailStats,
				Untriaged:  untriaged,
				Triaged:    triaged,
				TotalEmail: totalEmails,
				Threads:    threads,
				BeadsOpen:  beadsOpen,
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		display.Header("Mailbeads Statistics")
		fmt.Println()

		fmt.Println("  Emails")
		for _, acc := range accounts {
			s := emailStats[acc]
			syncInfo := ""
			if s.LastSync != "" {
				syncInfo = fmt.Sprintf("(last sync: %s)", display.TimeAgo(s.LastSync))
			}
			fmt.Printf("    %-28s %4d emails  %s\n",
				display.AccountLabel(acc), s.Count, display.Dim.Render(syncInfo))
		}
		fmt.Println()

		fmt.Println("  Triage")
		fmt.Printf("    Triaged    %3d threads\n", triaged)
		fmt.Printf("    Untriaged  %3d threads\n", untriaged)
		if beadsOpen > 0 {
			fmt.Printf("    Open beads %3d issues\n", beadsOpen)
		}
		fmt.Println()

		fmt.Printf("  Total: %d emails across %d threads\n", totalEmails, threads)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
