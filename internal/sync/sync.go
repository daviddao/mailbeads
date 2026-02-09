// Package sync fetches emails from Gmail using native Go API calls.
package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/daviddao/mailbeads/internal/auth"
	"github.com/daviddao/mailbeads/internal/beads"
	"github.com/daviddao/mailbeads/internal/db"
	"github.com/daviddao/mailbeads/internal/gmail"
	"github.com/daviddao/mailbeads/internal/types"
)

// DiscoverAccounts finds accounts by scanning for */credentials.json
// directories in the project root. Returns email addresses (directory names).
func DiscoverAccounts(projectRoot string) []string {
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		return nil
	}

	var accounts []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Look for directories that contain credentials.json and look like email addresses.
		if !strings.Contains(name, "@") {
			continue
		}
		credPath := filepath.Join(projectRoot, name, "credentials.json")
		if _, err := os.Stat(credPath); err == nil {
			accounts = append(accounts, name)
		}
	}

	sort.Strings(accounts)
	return accounts
}

// toGmailDate converts an ISO date to Gmail after: format.
func toGmailDate(isoDate string) string {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, isoDate); err == nil {
			return t.Format("2006/01/02")
		}
	}
	return ""
}

// SyncAccount fetches emails for a single account using native Go Gmail API.
func SyncAccount(store *db.DB, projectRoot, account string, forceFull bool, includeSpam bool, quiet bool) (*types.SyncResult, error) {
	result := &types.SyncResult{Account: account}
	ctx := context.Background()

	credPath := filepath.Join(projectRoot, account, "credentials.json")
	if _, err := os.Stat(credPath); err != nil {
		result.Error = "credentials not found"
		if !quiet {
			fmt.Fprintf(os.Stderr, "  ! %s — credentials not found, skipping\n", account)
		}
		return result, nil
	}

	// Authenticate and get Gmail service.
	svc, err := auth.LoadGmailService(ctx, credPath)
	if err != nil {
		result.Error = fmt.Sprintf("auth failed: %v", err)
		if !quiet {
			fmt.Fprintf(os.Stderr, "  ! %s — auth failed: %v\n", account, err)
		}
		return result, nil
	}

	// Determine search window.
	var query string
	latestDate := store.LatestEmailDate(account)

	if !forceFull && latestDate != "" {
		gmailDate := toGmailDate(latestDate)
		if gmailDate != "" {
			query = "after:" + gmailDate
			if !quiet {
				fmt.Printf("\n  %s — incremental (after %s)\n", account, gmailDate)
			}
		}
	}
	if query == "" {
		query = "newer_than:3d"
		if !quiet {
			fmt.Printf("\n  %s — full sync (last 72h)\n", account)
		}
	}

	// Only sync inbox by default (excludes drafts, sent-only, spam, trash).
	if !includeSpam {
		query += " in:inbox"
	}

	// Search Gmail natively.
	results, err := gmail.Search(svc, query, 100)
	if err != nil {
		result.Error = fmt.Sprintf("search failed: %v", err)
		if !quiet {
			fmt.Fprintf(os.Stderr, "  ! search failed: %v\n", err)
		}
		return result, nil
	}

	// Filter already-synced.
	var newEmails []gmail.MessageSummary
	for _, r := range results {
		if !store.EmailExists(r.ID) {
			newEmails = append(newEmails, r)
		}
	}

	if !quiet {
		fmt.Printf("  Found %d results, %d new\n", len(results), len(newEmails))
	}

	if len(newEmails) == 0 {
		result.Skipped = len(results)
		if !quiet {
			fmt.Printf("  ✓ 0 new, %d already synced\n", len(results))
		}
		return result, nil
	}

	// Fetch full content for new emails.
	now := time.Now().UTC().Format(time.RFC3339)

	for i, email := range newEmails {
		full, err := gmail.ReadFull(svc, email.ID)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "  ! failed to read %s: %v\n", email.ID, err)
			}
			continue
		}

		labels := strings.Join(full.Labels, ",")
		isRead := 1
		for _, l := range full.Labels {
			if l == "UNREAD" {
				isRead = 0
				break
			}
		}

		fromAddr := full.From
		if fromAddr == "" {
			fromAddr = email.From
		}
		subject := full.Subject
		if subject == "" {
			subject = email.Subject
		}
		date := full.Date
		if date == "" {
			date = email.Date
		}

		e := &types.Email{
			ID:        full.ID,
			Account:   account,
			ThreadID:  full.ThreadID,
			MessageID: full.MessageID,
			From:      fromAddr,
			To:        full.To,
			CC:        full.CC,
			Subject:   subject,
			Snippet:   email.Snippet,
			Body:      full.Body,
			Date:      date,
			Labels:    labels,
			IsRead:    isRead,
			FetchedAt: now,
		}

		if err := store.InsertEmail(e); err == nil {
			result.Fetched++
		}

		if !quiet {
			fmt.Fprintf(os.Stdout, "  Fetching %d/%d...\r", i+1, len(newEmails))
		}
	}

	result.Skipped = len(results) - len(newEmails)
	if !quiet {
		fmt.Printf("  ✓ %d new, %d already synced              \n", result.Fetched, result.Skipped)
	}

	// Auto-comment on beads issues for triaged threads that received new emails.
	if result.Fetched > 0 && beads.Available() {
		commented := notifyNewEmails(store, quiet)
		result.Commented = commented
	}

	return result, nil
}

// notifyNewEmails checks for triaged threads that have new emails and adds
// a comment to the corresponding beads issue.
func notifyNewEmails(store *db.DB, quiet bool) int {
	threads, err := store.ThreadsWithNewEmails()
	if err != nil || len(threads) == 0 {
		return 0
	}

	commented := 0
	for _, t := range threads {
		if t.TriageRef == nil {
			continue
		}
		// Skip legacy refs that haven't been migrated yet.
		if strings.HasPrefix(t.TriageRef.BeadID, "legacy-") {
			continue
		}

		comment := fmt.Sprintf("New email activity on thread: %s (%d emails, latest from %s)",
			t.Subject, t.EmailCount, t.From)

		if err := beads.Comment(t.TriageRef.BeadID, comment); err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "  ! comment on %s: %v\n", t.TriageRef.BeadID, err)
			}
			continue
		}

		// Update the triage ref created_at to mark the notification as sent.
		store.UpsertTriageRef(t.ThreadID, t.Account, t.TriageRef.BeadID)

		commented++
		if !quiet {
			fmt.Printf("  → Notified %s of new email on %q\n", t.TriageRef.BeadID, t.Subject)
		}
	}
	return commented
}
