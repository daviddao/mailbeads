// Package types defines core data structures for mailbeads.
package types

// Email represents a cached Gmail message.
type Email struct {
	ID        string `json:"id"`
	Account   string `json:"account"`
	ThreadID  string `json:"thread_id"`
	MessageID string `json:"message_id,omitempty"`
	From      string `json:"from"`
	To        string `json:"to,omitempty"`
	CC        string `json:"cc,omitempty"`
	Subject   string `json:"subject"`
	Snippet   string `json:"snippet,omitempty"`
	Body      string `json:"body,omitempty"`
	Date      string `json:"date"`
	Labels    string `json:"labels,omitempty"`
	IsRead    int    `json:"is_read"`
	FetchedAt string `json:"fetched_at"`
}

// TriageRef is a thin cross-reference mapping an email thread to a beads issue.
// All triage state (priority, status, action, dependencies) lives in beads.
type TriageRef struct {
	ThreadID  string `json:"thread_id"`
	Account   string `json:"account"`
	BeadID    string `json:"bead_id"`
	CreatedAt string `json:"created_at"`
}

// Thread groups emails by thread_id + account with optional triage reference.
type Thread struct {
	ThreadID   string     `json:"thread_id"`
	Account    string     `json:"account"`
	Subject    string     `json:"subject"`
	From       string     `json:"from"`
	EmailCount int        `json:"email_count"`
	LatestDate string     `json:"latest_date"`
	TriageRef  *TriageRef `json:"triage_ref,omitempty"`
}

// Priority constants (used for mb triage CLI flags, mapped to beads priorities).
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
	PrioritySpam   = "spam"
)

// ValidPriorities is the set of allowed priority values.
var ValidPriorities = []string{PriorityHigh, PriorityMedium, PriorityLow, PrioritySpam}

// IsValidPriority checks if a priority string is valid.
func IsValidPriority(p string) bool {
	for _, v := range ValidPriorities {
		if v == p {
			return true
		}
	}
	return false
}

// SyncResult holds the result of syncing a single account.
type SyncResult struct {
	Account   string `json:"account"`
	Fetched   int    `json:"fetched"`
	Skipped   int    `json:"skipped"`
	Commented int    `json:"commented,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SyncSummary holds the result of syncing all accounts.
type SyncSummary struct {
	Accounts  []SyncResult `json:"accounts"`
	TotalNew  int          `json:"total_new"`
	TotalInDB int          `json:"total_in_db"`
}
