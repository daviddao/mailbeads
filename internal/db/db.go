// Package db provides SQLite storage for mailbeads.
package db

import (
	"database/sql"
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
// Automatically migrates from old schema if needed.
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", dir, err)
	}

	conn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	d := &DB{conn: conn, path: dbPath}

	// Check if we need to migrate from old schema.
	if d.needsMigration() {
		if err := d.migrate(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("migrate schema: %w", err)
		}
	}

	// Apply current schema (creates tables if they don't exist).
	if _, err := conn.Exec(Schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return d, nil
}

// needsMigration checks if the old triage schema (with 'action' column) exists.
func (d *DB) needsMigration() bool {
	var name string
	err := d.conn.QueryRow(
		"SELECT name FROM pragma_table_info('triage') WHERE name = 'action'",
	).Scan(&name)
	return err == nil && name == "action"
}

// migrate runs the V2 schema migration.
func (d *DB) migrate() error {
	_, err := d.conn.Exec(MigrationV2)
	return err
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

// --- Triage cross-reference operations ---

// GetTriageRef returns the triage cross-reference for a thread, or nil if untriaged.
func (d *DB) GetTriageRef(threadID, account string) (*types.TriageRef, error) {
	t := &types.TriageRef{}
	err := d.conn.QueryRow(`
		SELECT thread_id, account, bead_id, created_at
		FROM triage
		WHERE thread_id = ? AND account = ?`, threadID, account).Scan(
		&t.ThreadID, &t.Account, &t.BeadID, &t.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetTriageRefByBead returns the triage cross-reference for a bead ID.
func (d *DB) GetTriageRefByBead(beadID string) (*types.TriageRef, error) {
	t := &types.TriageRef{}
	err := d.conn.QueryRow(`
		SELECT thread_id, account, bead_id, created_at
		FROM triage
		WHERE bead_id = ?`, beadID).Scan(
		&t.ThreadID, &t.Account, &t.BeadID, &t.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// UpsertTriageRef creates or updates a triage cross-reference.
func (d *DB) UpsertTriageRef(threadID, account, beadID string) (created bool, err error) {
	existing, err := d.GetTriageRef(threadID, account)
	if err != nil {
		return false, err
	}

	now := Now()
	if existing != nil {
		_, err = d.conn.Exec(`
			UPDATE triage SET bead_id = ? WHERE thread_id = ? AND account = ?`,
			beadID, threadID, account,
		)
		return false, err
	}

	_, err = d.conn.Exec(`
		INSERT INTO triage (thread_id, account, bead_id, created_at)
		VALUES (?, ?, ?, ?)`,
		threadID, account, beadID, now,
	)
	return true, err
}

// DeleteTriageRef removes a triage cross-reference by bead ID.
func (d *DB) DeleteTriageRef(beadID string) error {
	_, err := d.conn.Exec("DELETE FROM triage WHERE bead_id = ?", beadID)
	return err
}

// AllTriageRefs returns all triage cross-references.
func (d *DB) AllTriageRefs() ([]*types.TriageRef, error) {
	rows, err := d.conn.Query(`
		SELECT thread_id, account, bead_id, created_at
		FROM triage
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*types.TriageRef
	for rows.Next() {
		t := &types.TriageRef{}
		if err := rows.Scan(&t.ThreadID, &t.Account, &t.BeadID, &t.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// LegacyTriageRefs returns triage refs with "legacy-" prefixed bead IDs
// (created during schema migration, not yet migrated to real beads issues).
func (d *DB) LegacyTriageRefs() ([]*types.TriageRef, error) {
	rows, err := d.conn.Query(`
		SELECT thread_id, account, bead_id, created_at
		FROM triage
		WHERE bead_id LIKE 'legacy-%'
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*types.TriageRef
	for rows.Next() {
		t := &types.TriageRef{}
		if err := rows.Scan(&t.ThreadID, &t.Account, &t.BeadID, &t.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// --- Thread queries ---

// UntriagedThreads returns threads without a triage entry.
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
		HAVING t.bead_id IS NULL
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

// ThreadsWithNewEmails returns threads that have a triage ref but received
// new emails since triage (by checking if the thread's latest email date is
// newer than the triage created_at). Returns the thread info and the triage ref.
func (d *DB) ThreadsWithNewEmails() ([]*types.Thread, error) {
	rows, err := d.conn.Query(`
		SELECT e.thread_id, e.account,
		       MAX(e.subject) as subject,
		       MAX(e.from_addr) as from_addr,
		       COUNT(e.id) as email_count,
		       MAX(e.date) as latest_date,
		       t.bead_id, t.created_at
		FROM emails e
		JOIN triage t ON e.thread_id = t.thread_id AND e.account = t.account
		GROUP BY e.thread_id, e.account
		HAVING MAX(e.fetched_at) > t.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []*types.Thread
	for rows.Next() {
		t := &types.Thread{}
		ref := &types.TriageRef{}
		if err := rows.Scan(&t.ThreadID, &t.Account, &t.Subject, &t.From,
			&t.EmailCount, &t.LatestDate, &ref.BeadID, &ref.CreatedAt); err != nil {
			return nil, err
		}
		ref.ThreadID = t.ThreadID
		ref.Account = t.Account
		t.TriageRef = ref
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

// --- Aggregate queries ---

// UntriagedCount returns the number of untriaged threads.
func (d *DB) UntriagedCount() int {
	var n int
	d.conn.QueryRow(`
		SELECT COUNT(DISTINCT e.thread_id || '|' || e.account)
		FROM emails e
		LEFT JOIN triage t ON e.thread_id = t.thread_id AND e.account = t.account
		WHERE t.bead_id IS NULL`).Scan(&n)
	return n
}

// TriagedCount returns the number of triaged threads.
func (d *DB) TriagedCount() int {
	var n int
	d.conn.QueryRow("SELECT COUNT(*) FROM triage").Scan(&n)
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
