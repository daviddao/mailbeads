package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/display"
	"github.com/daviddao/mailbeads/internal/types"
	"github.com/spf13/cobra"
)

var (
	showAccount string
	showNoBody  bool
)

type showOutput struct {
	ThreadID string         `json:"thread_id"`
	Account  string         `json:"account"`
	Subject  string         `json:"subject"`
	Emails   []*types.Email `json:"emails"`
	Triage   *types.Triage  `json:"triage,omitempty"`
}

var showCmd = &cobra.Command{
	Use:   "show THREAD_ID",
	Short: "Display thread detail with emails and triage",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		threadID := args[0]

		account := showAccount
		if account == "" {
			accounts, err := store.ThreadAccounts(threadID)
			if err != nil {
				return fmt.Errorf("lookup thread: %w", err)
			}
			switch len(accounts) {
			case 0:
				return fmt.Errorf("thread %q not found", threadID)
			case 1:
				account = accounts[0]
			default:
				return fmt.Errorf("thread exists in multiple accounts (%v), specify --account", accounts)
			}
		}

		emails, err := store.ThreadEmails(threadID, account)
		if err != nil {
			return fmt.Errorf("fetch emails: %w", err)
		}
		if len(emails) == 0 {
			return fmt.Errorf("no emails found for thread %q in %s", threadID, account)
		}

		triage, err := store.GetTriage(threadID, account)
		if err != nil {
			return fmt.Errorf("fetch triage: %w", err)
		}

		if jsonOutput {
			out := showOutput{
				ThreadID: threadID,
				Account:  account,
				Subject:  emails[0].Subject,
				Emails:   emails,
				Triage:   triage,
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		// Pretty output
		fmt.Printf("Thread: %s (%s)\n", threadID, display.AccountLabel(account))
		fmt.Printf("Subject: %s\n", display.Bold.Render(emails[0].Subject))
		fmt.Printf("Emails: %d messages\n\n", len(emails))

		for i, e := range emails {
			var connector string
			switch {
			case len(emails) == 1:
				connector = "──"
			case i == 0:
				connector = "┌─"
			case i == len(emails)-1:
				connector = "└─"
			default:
				connector = "├─"
			}

			body := ""
			if !showNoBody {
				body = e.Body
				if body == "" {
					body = e.Snippet
				}
			}

			display.EmailTree(connector, e.From, e.Date, body)
			if i < len(emails)-1 {
				fmt.Println(display.Muted.Render("  │"))
			}
		}

		fmt.Println()
		if triage != nil {
			fmt.Printf("  Triage: %s\n", display.TriageBadge(triage.Priority, triage.Action))
			if triage.Suggestion != "" {
				fmt.Printf("  Suggestion: %s\n", triage.Suggestion)
			}
			if triage.AgentNotes != "" {
				fmt.Printf("  Notes: %s\n", display.Dim.Render(triage.AgentNotes))
			}
			fmt.Printf("  Status: %s", triage.Status)
			if triage.Category != "" {
				fmt.Printf("  Category: %s", triage.Category)
			}
			fmt.Println()
		} else {
			fmt.Println(display.Dim.Render("  Triage: (not yet triaged)"))
		}

		return nil
	},
}

func init() {
	showCmd.Flags().StringVar(&showAccount, "account", "", "Specify account")
	showCmd.Flags().BoolVar(&showNoBody, "no-body", false, "Hide email bodies")
	rootCmd.AddCommand(showCmd)
}
