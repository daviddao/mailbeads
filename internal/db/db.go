// Package db provides SQLite storage for mailbeads.
package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daviddao/mailbeads/internal/types"
	_ "modernc.org/sqlite"
)

// DB wraps a SQLite connection for mailbeads operations.
type DB struct {
	conn *sql.DB
	path string
}

// Open opens (or creates) a mailbeads database at the given path.
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", dir, err)
	}

	conn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := conn.Exec(Schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return &DB{conn: conn, path: dbPath}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}

// GenID generates a random 16-character hex ID.
func GenID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}

// Now returns the current time as an ISO 8601 string.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// DiscoverDB finds the mailbeads database by walking up from cwd.
// Returns the path to .mailbeads/mail.db or empty string if not found.
func DiscoverDB() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".mailbeads", "mail.db")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// FindProjectRoot walks up from cwd looking for a .git directory.
func FindProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// --- Email operations ---

// InsertEmail inserts an email, ignoring duplicates.
func (d *DB) InsertEmail(e *types.Email) error {
	_, err := d.conn.Exec(`
		INSERT OR IGNORE INTO emails
			(id, account, thread_id, message_id, from_addr, to_addr, cc, subject, snippet, body, date, labels, is_read, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Account, e.ThreadID, e.MessageID, e.From, e.To, e.CC,
		e.Subject, e.Snippet, e.Body, e.Date, e.Labels, e.IsRead, e.FetchedAt,
	)
	return err
}

// EmailExists checks if an email ID already exists.
func (d *DB) EmailExists(id string) bool {
	var n int
	d.conn.QueryRow("SELECT 1 FROM emails WHERE id = ?", id).Scan(&n)
	return n == 1
}

// LatestEmailDate returns the most recent email date for an account.
func (d *DB) LatestEmailDate(account string) string {
	var date sql.NullString
	d.conn.QueryRow("SELECT MAX(date) FROM emails WHERE account = ?", account).Scan(&date)
	if date.Valid {
		return date.String
	}
	return ""
}

// EmailCount returns the total number of emails.
func (d *DB) EmailCount() int {
	var n int
	d.conn.QueryRow("SELECT COUNT(*) FROM emails").Scan(&n)
	return n
}

// EmailCountByAccount returns email count for a specific account.
func (d *DB) EmailCountByAccount(account string) int {
	var n int
	d.conn.QueryRow("SELECT COUNT(*) FROM emails WHERE account = ?", account).Scan(&n)
	return n
}

// LatestFetchedAt returns the most recent fetched_at for an account.
func (d *DB) LatestFetchedAt(account string) string {
	var t sql.NullString
	d.conn.QueryRow("SELECT MAX(fetched_at) FROM emails WHERE account = ?", account).Scan(&t)
	if t.Valid {
		return t.String
	}
	return ""
}

// ThreadEmails returns all emails in a thread, ordered by date.
func (d *DB) ThreadEmails(threadID, account string) ([]*types.Email, error) {
	rows, err := d.conn.Query(`
		SELECT id, account, thread_id, message_id, from_addr, to_addr, cc,
		       subject, snippet, body, date, labels, is_read, fetched_at
		FROM emails
		WHERE thread_id = ? AND account = ?
		ORDER BY date ASC`, threadID, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEmails(rows)
}

// ThreadAccounts returns which accounts a thread_id appears in.
func (d *DB) ThreadAccounts(threadID string) ([]string, error) {
	rows, err := d.conn.Query(
		"SELECT DISTINCT account FROM emails WHERE thread_id = ?", threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accounts []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func scanEmails(rows *sql.Rows) ([]*types.Email, error) {
	var result []*types.Email
	for rows.Next() {
		e := &types.Email{}
		var msgID, to, cc, snippet, body, labels sql.NullString
		if err := rows.Scan(
			&e.ID, &e.Account, &e.ThreadID, &msgID, &e.From, &to, &cc,
			&e.Subject, &snippet, &body, &e.Date, &labels, &e.IsRead, &e.FetchedAt,
		); err != nil {
			return nil, err
		}
		e.MessageID = msgID.String
		e.To = to.String
		e.CC = cc.String
		e.Snippet = snippet.String
		e.Body = body.String
		e.Labels = labels.String
		result = append(result, e)
	}
	return result, rows.Err()
}

// --- Triage operations ---

// UntriageThreads returns threads without a triage entry (or with stale triage).
func (d *DB) UntriagedThreads(account string, limit int) ([]*types.Thread, error) {
	query := `
		SELECT e.thread_id, e.account,
		       MAX(e.subject) as subject,
		       MAX(e.from_addr) as from_addr,
		       COUNT(e.id) as email_count,
		       MAX(e.date) as latest_date
		FROM emails e
		LEFT JOIN triage t ON e.thread_id = t.thread_id AND e.account = t.account`

	var conditions []string
	args := []any{}

	if account != "" {
		conditions = append(conditions, `e.account = ?`)
		args = append(args, account)
	}

	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	query += ` GROUP BY e.thread_id, e.account
		HAVING t.id IS NULL OR t.latest_date < MAX(e.date)
		ORDER BY latest_date DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []*types.Thread
	for rows.Next() {
		t := &types.Thread{}
		if err := rows.Scan(&t.ThreadID, &t.Account, &t.Subject, &t.From, &t.EmailCount, &t.LatestDate); err != nil {
			return nil, err
		}
		threads = append(threads, t)
	}
	return threads, rows.Err()
}

// ThreadInfo returns aggregated info about a thread from the emails table.
func (d *DB) ThreadInfo(threadID, account string) (*types.Thread, error) {
	t := &types.Thread{}
	err := d.conn.QueryRow(`
		SELECT thread_id, account, MAX(subject), MAX(from_addr), COUNT(id), MAX(date)
		FROM emails
		WHERE thread_id = ? AND account = ?
		GROUP BY thread_id, account`, threadID, account).Scan(
		&t.ThreadID, &t.Account, &t.Subject, &t.From, &t.EmailCount, &t.LatestDate,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetTriage returns the triage entry for a thread.
func (d *DB) GetTriage(threadID, account string) (*types.Triage, error) {
	t := &types.Triage{}
	var from, suggestion, agentNotes, category, snoozed, latestDate, updatedAt sql.NullString
	err := d.conn.QueryRow(`
		SELECT id, thread_id, account, subject, from_addr, priority, action,
		       suggestion, agent_notes, category, status, snoozed_until,
		       email_count, latest_date, created_at, updated_at
		FROM triage
		WHERE thread_id = ? AND account = ?`, threadID, account).Scan(
		&t.ID, &t.ThreadID, &t.Account, &t.Subject, &from, &t.Priority, &t.Action,
		&suggestion, &agentNotes, &category, &t.Status, &snoozed,
		&t.EmailCount, &latestDate, &t.CreatedAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.From = from.String
	t.Suggestion = suggestion.String
	t.AgentNotes = agentNotes.String
	t.Category = category.String
	t.SnoozedUntil = snoozed.String
	t.LatestDate = latestDate.String
	t.UpdatedAt = updatedAt.String
	return t, nil
}

// GetTriageByID returns a triage entry by its ID (supports partial match).
func (d *DB) GetTriageByID(id string) (*types.Triage, error) {
	// Try exact match first
	t, err := d.getTriageByExactID(id)
	if err != nil {
		return nil, err
	}
	if t != nil {
		return t, nil
	}
	// Try prefix match
	rows, err := d.conn.Query(`
		SELECT id, thread_id, account, subject, from_addr, priority, action,
		       suggestion, agent_notes, category, status, snoozed_until,
		       email_count, latest_date, created_at, updated_at
		FROM triage
		WHERE id LIKE ?`, id+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*types.Triage
	for rows.Next() {
		t := &types.Triage{}
		var from, suggestion, agentNotes, category, snoozed, latestDate, updatedAt sql.NullString
		if err := rows.Scan(
			&t.ID, &t.ThreadID, &t.Account, &t.Subject, &from, &t.Priority, &t.Action,
			&suggestion, &agentNotes, &category, &t.Status, &snoozed,
			&t.EmailCount, &latestDate, &t.CreatedAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		t.From = from.String
		t.Suggestion = suggestion.String
		t.AgentNotes = agentNotes.String
		t.Category = category.String
		t.SnoozedUntil = snoozed.String
		t.LatestDate = latestDate.String
		t.UpdatedAt = updatedAt.String
		matches = append(matches, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("triage entry %q not found", id)
	case 1:
		return matches[0], nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.ID
		}
		return nil, fmt.Errorf("ambiguous ID %q, matches: %s", id, strings.Join(ids, ", "))
	}
}

func (d *DB) getTriageByExactID(id string) (*types.Triage, error) {
	t := &types.Triage{}
	var from, suggestion, agentNotes, category, snoozed, latestDate, updatedAt sql.NullString
	err := d.conn.QueryRow(`
		SELECT id, thread_id, account, subject, from_addr, priority, action,
		       suggestion, agent_notes, category, status, snoozed_until,
		       email_count, latest_date, created_at, updated_at
		FROM triage
		WHERE id = ?`, id).Scan(
		&t.ID, &t.ThreadID, &t.Account, &t.Subject, &from, &t.Priority, &t.Action,
		&suggestion, &agentNotes, &category, &t.Status, &snoozed,
		&t.EmailCount, &latestDate, &t.CreatedAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.From = from.String
	t.Suggestion = suggestion.String
	t.AgentNotes = agentNotes.String
	t.Category = category.String
	t.SnoozedUntil = snoozed.String
	t.LatestDate = latestDate.String
	t.UpdatedAt = updatedAt.String
	return t, nil
}

// UpsertTriage creates or updates a triage entry.
func (d *DB) UpsertTriage(t *types.Triage) (created bool, err error) {
	existing, err := d.GetTriage(t.ThreadID, t.Account)
	if err != nil {
		return false, err
	}

	now := Now()
	if existing != nil {
		_, err = d.conn.Exec(`
			UPDATE triage SET
				subject = ?, from_addr = ?, priority = ?, action = ?,
				suggestion = ?, agent_notes = ?, category = ?,
				email_count = ?, latest_date = ?, updated_at = ?, status = 'pending'
			WHERE id = ?`,
			t.Subject, nullStr(t.From), t.Priority, t.Action,
			nullStr(t.Suggestion), nullStr(t.AgentNotes), nullStr(t.Category),
			t.EmailCount, nullStr(t.LatestDate), now, existing.ID,
		)
		t.ID = existing.ID
		return false, err
	}

	t.ID = GenID()
	t.CreatedAt = now
	_, err = d.conn.Exec(`
		INSERT INTO triage
			(id, thread_id, account, subject, from_addr, priority, action,
			 suggestion, agent_notes, category, status, email_count, latest_date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?)`,
		t.ID, t.ThreadID, t.Account, t.Subject, nullStr(t.From), t.Priority, t.Action,
		nullStr(t.Suggestion), nullStr(t.AgentNotes), nullStr(t.Category),
		t.EmailCount, nullStr(t.LatestDate), now,
	)
	return true, err
}

// UpdateTriageStatus updates the status of a triage entry.
func (d *DB) UpdateTriageStatus(id, status string) error {
	res, err := d.conn.Exec(
		"UPDATE triage SET status = ?, updated_at = ? WHERE id = ?",
		status, Now(), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("triage entry %q not found", id)
	}
	return nil
}

// ListTriage returns triage entries filtered by status and optional account.
func (d *DB) ListTriage(status, account string, includeAll bool) ([]*types.Triage, error) {
	query := "SELECT id, thread_id, account, subject, from_addr, priority, action, suggestion, agent_notes, category, status, snoozed_until, email_count, latest_date, created_at, updated_at FROM triage"

	var conditions []string
	var args []any

	if !includeAll {
		if status != "" {
			conditions = append(conditions, "status = ?")
			args = append(args, status)
		} else {
			conditions = append(conditions, "status = 'pending'")
		}
	}
	if account != "" {
		conditions = append(conditions, "account LIKE ?")
		args = append(args, "%"+account+"%")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += ` ORDER BY
		CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 WHEN 'low' THEN 2 WHEN 'spam' THEN 3 END,
		latest_date DESC`

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*types.Triage
	for rows.Next() {
		t := &types.Triage{}
		var from, suggestion, agentNotes, category, snoozed, latestDate, updatedAt sql.NullString
		if err := rows.Scan(
			&t.ID, &t.ThreadID, &t.Account, &t.Subject, &from, &t.Priority, &t.Action,
			&suggestion, &agentNotes, &category, &t.Status, &snoozed,
			&t.EmailCount, &latestDate, &t.CreatedAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		t.From = from.String
		t.Suggestion = suggestion.String
		t.AgentNotes = agentNotes.String
		t.Category = category.String
		t.SnoozedUntil = snoozed.String
		t.LatestDate = latestDate.String
		t.UpdatedAt = updatedAt.String
		result = append(result, t)
	}
	return result, rows.Err()
}

// ReadyTriage returns actionable triage items (pending, not snoozed, not blocked).
func (d *DB) ReadyTriage(account string) ([]*types.Triage, error) {
	query := `
		SELECT t.id, t.thread_id, t.account, t.subject, t.from_addr, t.priority, t.action,
		       t.suggestion, t.agent_notes, t.category, t.status, t.snoozed_until,
		       t.email_count, t.latest_date, t.created_at, t.updated_at
		FROM triage t
		WHERE t.status = 'pending'
		  AND (t.snoozed_until IS NULL OR t.snoozed_until <= datetime('now'))
		  AND NOT EXISTS (
		      SELECT 1 FROM triage_deps d
		      JOIN triage blocker ON d.depends_on_id = blocker.id
		      WHERE d.triage_id = t.id AND blocker.status = 'pending'
		  )`

	var args []any
	if account != "" {
		query += " AND t.account LIKE ?"
		args = append(args, "%"+account+"%")
	}

	query += ` ORDER BY
		CASE t.priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 WHEN 'low' THEN 2 WHEN 'spam' THEN 3 END,
		t.latest_date DESC`

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*types.Triage
	for rows.Next() {
		t := &types.Triage{}
		var from, suggestion, agentNotes, category, snoozed, latestDate, updatedAt sql.NullString
		if err := rows.Scan(
			&t.ID, &t.ThreadID, &t.Account, &t.Subject, &from, &t.Priority, &t.Action,
			&suggestion, &agentNotes, &category, &t.Status, &snoozed,
			&t.EmailCount, &latestDate, &t.CreatedAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		t.From = from.String
		t.Suggestion = suggestion.String
		t.AgentNotes = agentNotes.String
		t.Category = category.String
		t.SnoozedUntil = snoozed.String
		t.LatestDate = latestDate.String
		t.UpdatedAt = updatedAt.String
		result = append(result, t)
	}
	return result, rows.Err()
}

// TriageCountByStatus returns counts grouped by status.
func (d *DB) TriageCountByStatus() (map[string]int, error) {
	rows, err := d.conn.Query("SELECT status, COUNT(*) FROM triage GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{"pending": 0, "done": 0, "dismissed": 0, "snoozed": 0}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

// TriageCountByPriority returns pending triage counts grouped by priority.
func (d *DB) TriageCountByPriority() (map[string]int, error) {
	rows, err := d.conn.Query("SELECT priority, COUNT(*) FROM triage WHERE status = 'pending' GROUP BY priority")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{"high": 0, "medium": 0, "low": 0, "spam": 0}
	for rows.Next() {
		var priority string
		var count int
		if err := rows.Scan(&priority, &count); err != nil {
			return nil, err
		}
		counts[priority] = count
	}
	return counts, rows.Err()
}

// UntriagedCount returns the number of untriaged threads.
func (d *DB) UntriagedCount() int {
	var n int
	d.conn.QueryRow(`
		SELECT COUNT(DISTINCT e.thread_id || '|' || e.account)
		FROM emails e
		LEFT JOIN triage t ON e.thread_id = t.thread_id AND e.account = t.account
		WHERE t.id IS NULL`).Scan(&n)
	return n
}

// ThreadCount returns total distinct threads.
func (d *DB) ThreadCount() int {
	var n int
	d.conn.QueryRow("SELECT COUNT(DISTINCT thread_id || '|' || account) FROM emails").Scan(&n)
	return n
}

// Accounts returns distinct email accounts.
func (d *DB) Accounts() []string {
	rows, err := d.conn.Query("SELECT DISTINCT account FROM emails ORDER BY account")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var accs []string
	for rows.Next() {
		var a string
		rows.Scan(&a)
		accs = append(accs, a)
	}
	return accs
}

// Underlying returns the raw sql.DB connection (for frontend readonly access).
func (d *DB) Underlying() *sql.DB {
	return d.conn
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
