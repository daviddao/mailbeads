package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/db"
	"github.com/daviddao/mailbeads/internal/display"
	msync "github.com/daviddao/mailbeads/internal/sync"
	"github.com/daviddao/mailbeads/internal/types"
	"github.com/spf13/cobra"
)

var (
	syncFull    bool
	syncAccount string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch emails from Gmail into the local database",
	Long:  "Sync emails from all discovered Gmail accounts into the mailbeads database.",
	RunE: func(cmd *cobra.Command, args []string) error {
		root := db.FindProjectRoot()
		if root == "" {
			return fmt.Errorf("could not find project root (no .git directory)")
		}

		if !quietFlag {
			mode := ""
			if syncFull {
				mode = " (full 72h)"
			}
			fmt.Printf("Syncing emails%s...\n", mode)
		}

		var accounts []string
		if syncAccount != "" {
			accounts = []string{syncAccount}
		} else {
			accounts = msync.DiscoverAccounts(root)
		}
		if len(accounts) == 0 {
			return fmt.Errorf("no accounts found — add account directories with credentials.json to the project root")
		}

		summary := &types.SyncSummary{}
		for _, account := range accounts {
			result, err := msync.SyncAccount(store, root, account, syncFull, quietFlag)
			if err != nil {
				return err
			}
			summary.Accounts = append(summary.Accounts, *result)
			summary.TotalNew += result.Fetched
		}
		summary.TotalInDB = store.EmailCount()

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(summary)
		}

		if !quietFlag {
			fmt.Println()
			display.SuccessMsg("Done! %d new emails synced. Total in DB: %d", summary.TotalNew, summary.TotalInDB)
		}
		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncFull, "full", false, "Force full 72h re-scan")
	syncCmd.Flags().StringVar(&syncAccount, "account", "", "Sync single account")
	rootCmd.AddCommand(syncCmd)
}
