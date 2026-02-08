package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/daviddao/mailbeads/internal/auth"
	"github.com/daviddao/mailbeads/internal/db"
	"github.com/daviddao/mailbeads/internal/display"
	"github.com/daviddao/mailbeads/internal/gmail"
	msync "github.com/daviddao/mailbeads/internal/sync"
	"github.com/spf13/cobra"
)

var (
	gmailAccount     string
	gmailCredentials string
	gmailMaxResults  int
	gmailFormat      string
)

// gmailCmd is the parent command for Gmail operations.
var gmailCmd = &cobra.Command{
	Use:   "gmail",
	Short: "Gmail operations (search, read)",
	Long:  "Search and read Gmail messages using native Go API calls.",
}

// gmailSearchCmd replaces search_emails.py.
var gmailSearchCmd = &cobra.Command{
	Use:   "search QUERY",
	Short: "Search Gmail messages",
	Long: `Search Gmail messages matching a query.

Uses the same query syntax as Gmail's search box.
Searches across both accounts by default, or use --account to search one.`,
	Example: `  mb gmail search "from:someone@example.com"
  mb gmail search "subject:urgent is:unread" -n 20
  mb gmail search "after:2024/01/01 has:attachment"
  mb gmail search "newer_than:7d" --account user@example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		ctx := context.Background()
		root := db.FindProjectRoot()
		if root == "" {
			return fmt.Errorf("could not find project root (no .git directory)")
		}

		accounts := resolveAccounts(root, gmailAccount)
		if len(accounts) == 0 {
			return fmt.Errorf("no accounts found — add account directories with credentials.json to the project root")
		}

		var allResults []gmail.MessageSummary
		for _, account := range accounts {
			credPath := resolveCredentials(root, account, gmailCredentials)
			svc, err := auth.LoadGmailService(ctx, credPath)
			if err != nil {
				if !quietFlag {
					fmt.Fprintf(cmd.ErrOrStderr(), "  ! %s — %v, skipping\n", account, err)
				}
				continue
			}

			results, err := gmail.Search(svc, query, int64(gmailMaxResults))
			if err != nil {
				if !quietFlag {
					fmt.Fprintf(cmd.ErrOrStderr(), "  ! %s — search failed: %v\n", account, err)
				}
				continue
			}

			allResults = append(allResults, results...)
		}

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(allResults)
		}

		if len(allResults) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No messages found matching: %s\n", query)
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Found %d message(s) matching: %s\n\n", len(allResults), query)
		for i, msg := range allResults {
			fmt.Fprintf(cmd.OutOrStdout(), "[%d] ID: %s\n", i+1, msg.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "    From: %s\n", msg.From)
			fmt.Fprintf(cmd.OutOrStdout(), "    Subject: %s\n", msg.Subject)
			fmt.Fprintf(cmd.OutOrStdout(), "    Date: %s\n", msg.Date)
			snippet := msg.Snippet
			if len(snippet) > 100 {
				snippet = snippet[:100] + "..."
			}
			fmt.Fprintf(cmd.OutOrStdout(), "    Preview: %s\n\n", snippet)
		}
		return nil
	},
}

// gmailReadCmd replaces read_email.py.
var gmailReadCmd = &cobra.Command{
	Use:   "read MESSAGE_ID",
	Short: "Read a Gmail message by ID",
	Long: `Read the full content of a Gmail message.

Fetches the complete message including headers, body, labels, and attachments.
Automatically detects which account the message belongs to.`,
	Example: `  mb gmail read 18d5a7b3c4e5f6a7
  mb gmail read 18d5a7b3c4e5f6a7 --format full
  mb gmail read 18d5a7b3c4e5f6a7 --json
  mb gmail read 18d5a7b3c4e5f6a7 --account user@example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID := args[0]
		ctx := context.Background()
		root := db.FindProjectRoot()
		if root == "" {
			return fmt.Errorf("could not find project root (no .git directory)")
		}

		accounts := resolveAccounts(root, gmailAccount)
		if len(accounts) == 0 {
			return fmt.Errorf("no accounts found — add account directories with credentials.json to the project root")
		}
		includeFull := gmailFormat == "full"

		// Try each account until we find the message.
		for _, account := range accounts {
			credPath := resolveCredentials(root, account, gmailCredentials)
			svc, err := auth.LoadGmailService(ctx, credPath)
			if err != nil {
				continue
			}

			if includeFull {
				msg, err := gmail.ReadFullWithAttachments(svc, messageID)
				if err != nil {
					continue // Try next account.
				}
				return outputReadResult(cmd, msg, account)
			}

			msg, err := gmail.ReadFull(svc, messageID)
			if err != nil {
				continue // Try next account.
			}
			return outputBasicReadResult(cmd, msg, account)
		}

		return fmt.Errorf("message %s not found in any account", messageID)
	},
}

func outputReadResult(cmd *cobra.Command, msg *gmail.FullMessageWithAttachments, account string) error {
	if jsonOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(msg)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "From: %s\n", msg.From)
	fmt.Fprintf(w, "To: %s\n", msg.To)
	if msg.CC != "" {
		fmt.Fprintf(w, "Cc: %s\n", msg.CC)
	}
	fmt.Fprintf(w, "Subject: %s\n", msg.Subject)
	fmt.Fprintf(w, "Date: %s\n", msg.Date)
	if msg.MessageID != "" {
		fmt.Fprintf(w, "Message-ID: %s\n", msg.MessageID)
	}
	fmt.Fprintf(w, "Account: %s\n", display.AccountLabel(account))
	fmt.Fprintf(w, "Labels: %s\n", strings.Join(msg.Labels, ", "))
	if len(msg.Attachments) > 0 {
		fmt.Fprintf(w, "Attachments:\n")
		for _, att := range msg.Attachments {
			fmt.Fprintf(w, "  - %s (%s, %d bytes)\n", att.Filename, att.MimeType, att.Size)
		}
	}
	fmt.Fprintf(w, "\n%s\n\n", strings.Repeat("=", 60))
	fmt.Fprintf(w, "%s\n", msg.Body)
	return nil
}

func outputBasicReadResult(cmd *cobra.Command, msg *gmail.FullMessage, account string) error {
	if jsonOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(msg)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "From: %s\n", msg.From)
	fmt.Fprintf(w, "To: %s\n", msg.To)
	if msg.CC != "" {
		fmt.Fprintf(w, "Cc: %s\n", msg.CC)
	}
	fmt.Fprintf(w, "Subject: %s\n", msg.Subject)
	fmt.Fprintf(w, "Date: %s\n", msg.Date)
	if msg.MessageID != "" {
		fmt.Fprintf(w, "Message-ID: %s\n", msg.MessageID)
	}
	fmt.Fprintf(w, "Account: %s\n", display.AccountLabel(account))
	fmt.Fprintf(w, "\n%s\n\n", strings.Repeat("=", 60))
	fmt.Fprintf(w, "%s\n", msg.Body)
	return nil
}

// resolveAccounts returns the list of accounts to operate on.
func resolveAccounts(root, account string) []string {
	if account != "" {
		return []string{account}
	}
	return msync.DiscoverAccounts(root)
}

// resolveCredentials returns the credentials path for an account.
func resolveCredentials(root, account, explicit string) string {
	if explicit != "" {
		return explicit
	}
	return filepath.Join(root, account, "credentials.json")
}

func init() {
	// Gmail parent flags.
	gmailCmd.PersistentFlags().StringVar(&gmailAccount, "account", "", "Gmail account to use (default: all accounts)")
	gmailCmd.PersistentFlags().StringVar(&gmailCredentials, "credentials", "", "Path to credentials.json")

	// Search flags.
	gmailSearchCmd.Flags().IntVarP(&gmailMaxResults, "max-results", "n", 10, "Maximum results to return")

	// Read flags.
	gmailReadCmd.Flags().StringVarP(&gmailFormat, "format", "f", "basic", "Output format: basic or full")

	// Wire up.
	gmailCmd.AddCommand(gmailSearchCmd)
	gmailCmd.AddCommand(gmailReadCmd)
	rootCmd.AddCommand(gmailCmd)
}
