package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daviddao/mailbeads/internal/db"
	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time.
var Version = "dev"

var (
	dbPath     string
	jsonOutput bool
	quietFlag  bool
	store      *db.DB
)

var rootCmd = &cobra.Command{
	Use:   "mb",
	Short: "mb - Email inbox triage for AI agents",
	Long:  "Mailbeads: sync Gmail, triage threads, track dependencies. Inspired by beads.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip DB for commands that don't need it
		name := cmd.Name()
		switch name {
		case "init", "help", "version", "quickstart", "onboard":
			return nil
		case "search", "read":
			// Gmail subcommands don't need the DB
			if cmd.Parent() != nil && cmd.Parent().Name() == "gmail" {
				return nil
			}
		case "gmail":
			// Parent command (shows help)
			return nil
		case "prime":
			// Prime works without DB (just no live stats)
			path := dbPath
			if path == "" {
				path = db.DiscoverDB()
			}
			if path != "" {
				var err error
				store, err = db.Open(path)
				if err != nil {
					store = nil // continue without DB
				}
			}
			return nil
		}

		// Discover database
		path := dbPath
		if path == "" {
			path = db.DiscoverDB()
		}
		if path == "" {
			return fmt.Errorf("no mailbeads database found — run 'mb init' first")
		}

		var err error
		store, err = db.Open(path)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if store != nil {
			store.Close()
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mb version %s\n", Version)
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .mailbeads/ in the project root",
	RunE: func(cmd *cobra.Command, args []string) error {
		root := db.FindProjectRoot()
		if root == "" {
			return fmt.Errorf("could not find project root (no .git directory found)")
		}

		dbPath := root + "/.mailbeads/mail.db"
		s, err := db.Open(dbPath)
		if err != nil {
			return err
		}
		s.Close()

		// Add .mailbeads/ to .gitignore if not already present
		ensureGitignore(root)

		if !quietFlag {
			fmt.Printf("Initialized mailbeads at %s\n", dbPath)
		}
		return nil
	},
}

// ensureGitignore adds .mailbeads/ to .gitignore if not already present.
func ensureGitignore(root string) {
	gitignorePath := filepath.Join(root, ".gitignore")
	entry := ".mailbeads/"

	// Check if entry already exists
	if f, err := os.Open(gitignorePath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == entry || line == ".mailbeads" {
				f.Close()
				return // already present
			}
		}
		f.Close()
	}

	// Append to .gitignore
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return // silently skip if can't write
	}
	defer f.Close()

	// Check if file ends with newline
	info, err := f.Stat()
	if err == nil && info.Size() > 0 {
		// Read last byte to check for trailing newline
		rf, err := os.Open(gitignorePath)
		if err == nil {
			buf := make([]byte, 1)
			rf.Seek(-1, 2)
			rf.Read(buf)
			rf.Close()
			if buf[0] != '\n' {
				f.WriteString("\n")
			}
		}
	}

	fmt.Fprintf(f, "\n# Mailbeads database (local email triage)\n%s\n", entry)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database path (default: auto-discover .mailbeads/mail.db)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress non-essential output")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
