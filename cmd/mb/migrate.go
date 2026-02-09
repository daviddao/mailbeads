package main

import (
	"encoding/json"
	"fmt"

	"github.com/daviddao/mailbeads/internal/beads"
	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

var migrateDryRun bool

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate pending triage entries from old schema to beads issues",
	Long: `Migrate legacy triage entries (from the old fat triage table) to beads issues.

During schema migration, pending entries were preserved with bead_id prefixed
with "legacy-". This command creates actual beads issues for each one and
updates the cross-reference.

Use --dry-run to preview what would be created without making changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !beads.Available() {
			return fmt.Errorf("bd (beads) CLI not found on PATH — install from https://beads.sh")
		}

		legacyRefs, err := store.LegacyTriageRefs()
		if err != nil {
			return fmt.Errorf("query legacy refs: %w", err)
		}

		if len(legacyRefs) == 0 {
			fmt.Println("No legacy triage entries to migrate.")
			return nil
		}

		fmt.Printf("Found %d legacy triage entries to migrate.\n\n", len(legacyRefs))

		for _, ref := range legacyRefs {
			// Get thread info for the beads issue.
			info, err := store.ThreadInfo(ref.ThreadID, ref.Account)
			if err != nil {
				display.ErrorMsg("thread %s not found in %s, skipping", ref.ThreadID, ref.Account)
				continue
			}

			if migrateDryRun {
				fmt.Printf("  [dry-run] Would create beads issue for: %s (%s)\n",
					display.Truncate(info.Subject, 50), ref.Account)
				continue
			}

			// Create beads issue with thread metadata.
			notes := fmt.Sprintf("from=%s account=%s thread=%s emails=%d\n\nMigrated from legacy mailbeads triage.",
				info.From, ref.Account, ref.ThreadID, info.EmailCount)

			issue, err := beads.Create(
				info.Subject, // Use subject as action since we don't have the old action.
				"",           // No suggestion.
				notes,
				beads.PriorityToBeads("medium"), // Default to medium.
				"",                              // No category.
				"",                              // No epic.
				nil,                             // No extra labels.
				ref.ThreadID,
			)
			if err != nil {
				display.ErrorMsg("create beads issue for %s: %v", ref.ThreadID, err)
				continue
			}

			// Update local cross-reference.
			if _, err := store.UpsertTriageRef(ref.ThreadID, ref.Account, issue.ID); err != nil {
				display.ErrorMsg("update ref for %s: %v", ref.ThreadID, err)
				continue
			}

			display.SuccessMsg("Migrated %s -> %s %q", ref.ThreadID, issue.ID, info.Subject)
		}

		if migrateDryRun {
			fmt.Println()
			fmt.Println("Run without --dry-run to execute migration.")
		}

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]int{"migrated": len(legacyRefs)})
		}

		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Preview what would be migrated")
	rootCmd.AddCommand(migrateCmd)
}
