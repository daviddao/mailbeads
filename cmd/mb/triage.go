package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/beads"
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
	triageEpic       string
)

type triageOutput struct {
	ThreadID string `json:"thread_id"`
	Account  string `json:"account"`
	BeadID   string `json:"bead_id"`
	Action   string `json:"action"`
	Priority string `json:"priority"`
	Subject  string `json:"subject"`
	Created  bool   `json:"created"`
}

var triageCmd = &cobra.Command{
	Use:   "triage THREAD_ID",
	Short: "Create or update a triage entry for a thread",
	Long: `Create a beads issue for an email thread and link it in the local database.

The triage decision (priority, action, notes) is stored as a beads issue via
the bd CLI. The local mailbeads database only keeps a cross-reference so
mb untriaged / mb show can look up whether a thread has been triaged.

Examples:
  mb triage 19abc123 --action "Reply with agenda" --priority high
  mb triage 19abc123 --action "FYI" --suggestion "No response needed"
  mb triage 19abc123 --action "Review PR" --epic bd-a3f8`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		threadID := args[0]

		if !beads.Available() {
			return fmt.Errorf("bd (beads) CLI not found on PATH — install from https://beads.sh")
		}

		if triageAccount == "" {
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

		// Get thread info from emails table.
		info, err := store.ThreadInfo(threadID, triageAccount)
		if err != nil {
			return fmt.Errorf("thread %q not found in %s", threadID, triageAccount)
		}

		from := triageFrom
		if from == "" {
			from = info.From
		}

		// Check if already triaged.
		existing, err := store.GetTriageRef(threadID, triageAccount)
		if err != nil {
			return fmt.Errorf("check existing triage: %w", err)
		}

		bdPriority := beads.PriorityToBeads(triagePriority)

		// Build notes with email metadata for the beads issue.
		notes := fmt.Sprintf("from=%s account=%s thread=%s emails=%d",
			from, triageAccount, threadID, info.EmailCount)
		if triageAgentNotes != "" {
			notes += "\n\n" + triageAgentNotes
		}

		var beadID string
		var created bool

		if existing != nil {
			// Update the existing beads issue.
			beadID = existing.BeadID
			fields := map[string]string{
				"title":    triageAction,
				"priority": bdPriority,
			}
			if triageSuggestion != "" {
				fields["description"] = triageSuggestion
			}
			if err := beads.Update(beadID, fields); err != nil {
				return fmt.Errorf("update beads issue: %w", err)
			}
		} else {
			// Create a new beads issue.
			var extraLabels []string
			issue, err := beads.Create(
				triageAction,
				triageSuggestion,
				notes,
				bdPriority,
				triageCategory,
				triageEpic,
				extraLabels,
				threadID,
			)
			if err != nil {
				return fmt.Errorf("create beads issue: %w", err)
			}
			beadID = issue.ID
			created = true

			// Store cross-reference in local DB.
			if _, err := store.UpsertTriageRef(threadID, triageAccount, beadID); err != nil {
				return fmt.Errorf("save triage ref: %w", err)
			}
		}

		// If --epic was specified and we just created the issue, link it.
		if triageEpic != "" && !created {
			// For updates, add the dep if epic changed.
			if err := beads.AddDep(beadID, triageEpic); err != nil {
				display.ErrorMsg("link to epic: %v", err)
			}
		}

		if jsonOutput {
			out := triageOutput{
				ThreadID: threadID,
				Account:  triageAccount,
				BeadID:   beadID,
				Action:   triageAction,
				Priority: triagePriority,
				Subject:  info.Subject,
				Created:  created,
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		verb := "Updated"
		if created {
			verb = "Triaged"
		}
		display.SuccessMsg("%s %s [%s] %q", verb, beadID, triagePriority, triageAction)
		if triageEpic != "" {
			fmt.Printf("  Linked to epic: %s\n", triageEpic)
		}
		return nil
	},
}

func init() {
	triageCmd.Flags().StringVar(&triageAccount, "account", "", "Gmail account")
	triageCmd.Flags().StringVar(&triagePriority, "priority", "", "Priority: high, medium, low, spam (default: medium)")
	triageCmd.Flags().StringVar(&triageAction, "action", "", "Short action phrase (required)")
	triageCmd.Flags().StringVar(&triageSuggestion, "suggestion", "", "Detailed suggestion (stored as beads description)")
	triageCmd.Flags().StringVar(&triageAgentNotes, "agent-notes", "", "Agent reasoning notes")
	triageCmd.Flags().StringVar(&triageCategory, "category", "", "Category label")
	triageCmd.Flags().StringVar(&triageFrom, "from", "", "Sender (auto-detected if omitted)")
	triageCmd.Flags().StringVar(&triageEpic, "epic", "", "Link to a beads epic (e.g., bd-a3f8)")
	rootCmd.AddCommand(triageCmd)
}
