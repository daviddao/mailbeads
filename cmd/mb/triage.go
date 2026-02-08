package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/display"
	"github.com/daviddao/mailbeads/internal/types"
	"github.com/spf13/cobra"
)

var (
	triageAccount    string
	triagePriority   string
	triageAction     string
	triageSuggestion string
	triageAgentNotes string
	triageCategory   string
	triageFrom       string
)

var triageCmd = &cobra.Command{
	Use:   "triage THREAD_ID",
	Short: "Create or update a triage entry for a thread",
	Long: `Create or update a triage entry for an email thread.

Examples:
  mb triage 19abc123 --account user@example.com --action "Reply with agenda" --priority high
  mb triage 19abc123 --account user@example.com --action "FYI" --suggestion "No response needed"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		threadID := args[0]

		if triageAccount == "" {
			// Auto-detect account if thread only exists in one
			accounts, err := store.ThreadAccounts(threadID)
			if err != nil {
				return fmt.Errorf("lookup thread: %w", err)
			}
			switch len(accounts) {
			case 0:
				return fmt.Errorf("thread %q not found in emails", threadID)
			case 1:
				triageAccount = accounts[0]
			default:
				return fmt.Errorf("thread exists in multiple accounts (%v), specify --account", accounts)
			}
		}

		if triageAction == "" {
			return fmt.Errorf("--action is required")
		}

		if triagePriority != "" && !types.IsValidPriority(triagePriority) {
			return fmt.Errorf("invalid priority %q (must be: high, medium, low, spam)", triagePriority)
		}
		if triagePriority == "" {
			triagePriority = types.PriorityMedium
		}

		// Get thread info from emails table
		info, err := store.ThreadInfo(threadID, triageAccount)
		if err != nil {
			return fmt.Errorf("thread %q not found in %s", threadID, triageAccount)
		}

		from := triageFrom
		if from == "" {
			from = info.From
		}

		t := &types.Triage{
			ThreadID:   threadID,
			Account:    triageAccount,
			Subject:    info.Subject,
			From:       from,
			Priority:   triagePriority,
			Action:     triageAction,
			Suggestion: triageSuggestion,
			AgentNotes: triageAgentNotes,
			Category:   triageCategory,
			EmailCount: info.EmailCount,
			LatestDate: info.LatestDate,
		}

		created, err := store.UpsertTriage(t)
		if err != nil {
			return fmt.Errorf("upsert triage: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(t)
		}

		verb := "Updated"
		if created {
			verb = "Triaged"
		}
		display.SuccessMsg("%s %s [%s] %q", verb, t.ID, t.Priority, t.Action)
		return nil
	},
}

func init() {
	triageCmd.Flags().StringVar(&triageAccount, "account", "", "Gmail account")
	triageCmd.Flags().StringVar(&triagePriority, "priority", "", "Priority: high, medium, low, spam (default: medium)")
	triageCmd.Flags().StringVar(&triageAction, "action", "", "Short action phrase (required)")
	triageCmd.Flags().StringVar(&triageSuggestion, "suggestion", "", "Detailed suggestion")
	triageCmd.Flags().StringVar(&triageAgentNotes, "agent-notes", "", "Agent reasoning notes")
	triageCmd.Flags().StringVar(&triageCategory, "category", "", "Category label")
	triageCmd.Flags().StringVar(&triageFrom, "from", "", "Sender (auto-detected if omitted)")
	rootCmd.AddCommand(triageCmd)
}
