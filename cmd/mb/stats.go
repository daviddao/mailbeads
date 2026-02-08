package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

type statsOutput struct {
	Emails     map[string]accountStats `json:"emails"`
	Triage     map[string]int          `json:"triage"`
	Priority   map[string]int          `json:"priority"`
	Untriaged  int                     `json:"untriaged"`
	TotalEmail int                     `json:"total_emails"`
	Threads    int                     `json:"threads"`
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

		if jsonOutput {
			out := statsOutput{
				Emails:     emailStats,
				Triage:     triageCounts,
				Priority:   priorityCounts,
				Untriaged:  untriaged,
				TotalEmail: totalEmails,
				Threads:    threads,
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
		pending := triageCounts["pending"]
		detail := ""
		if pending > 0 {
			parts := []string{}
			for _, p := range []string{"high", "medium", "low"} {
				if c := priorityCounts[p]; c > 0 {
					parts = append(parts, fmt.Sprintf("%d %s", c, p))
				}
			}
			if len(parts) > 0 {
				detail = " ("
				for i, p := range parts {
					if i > 0 {
						detail += ", "
					}
					detail += p
				}
				detail += ")"
			}
		}
		fmt.Printf("    Pending    %3d%s\n", pending, detail)
		fmt.Printf("    Done       %3d\n", triageCounts["done"])
		fmt.Printf("    Dismissed  %3d\n", triageCounts["dismissed"])
		fmt.Printf("    Untriaged  %3d threads\n", untriaged)
		fmt.Println()

		fmt.Printf("  Total: %d emails across %d threads\n", totalEmails, threads)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
