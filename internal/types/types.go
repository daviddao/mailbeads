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

// Triage represents an AI-generated triage decision for an email thread.
type Triage struct {
	ID           string `json:"id"`
	ThreadID     string `json:"thread_id"`
	Account      string `json:"account"`
	Subject      string `json:"subject"`
	From         string `json:"from,omitempty"`
	Priority     string `json:"priority"`
	Action       string `json:"action"`
	Suggestion   string `json:"suggestion,omitempty"`
	AgentNotes   string `json:"agent_notes,omitempty"`
	Category     string `json:"category,omitempty"`
	Status       string `json:"status"`
	SnoozedUntil string `json:"snoozed_until,omitempty"`
	EmailCount   int    `json:"email_count"`
	LatestDate   string `json:"latest_date,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// TriageDep represents a dependency between triage entries.
type TriageDep struct {
	TriageID    string `json:"triage_id"`
	DependsOnID string `json:"depends_on_id"`
	CreatedAt   string `json:"created_at"`
}

// Thread groups emails by thread_id + account with optional triage.
type Thread struct {
	ThreadID   string  `json:"thread_id"`
	Account    string  `json:"account"`
	Subject    string  `json:"subject"`
	From       string  `json:"from"`
	EmailCount int     `json:"email_count"`
	LatestDate string  `json:"latest_date"`
	Triage     *Triage `json:"triage,omitempty"`
}

// Priority constants.
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

// Status constants.
const (
	StatusPending   = "pending"
	StatusDone      = "done"
	StatusDismissed = "dismissed"
	StatusSnoozed   = "snoozed"
)

// ValidStatuses is the set of allowed status values.
var ValidStatuses = []string{StatusPending, StatusDone, StatusDismissed, StatusSnoozed}

// IsValidStatus checks if a status string is valid.
func IsValidStatus(s string) bool {
	for _, v := range ValidStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// SyncResult holds the result of syncing a single account.
type SyncResult struct {
	Account string `json:"account"`
	Fetched int    `json:"fetched"`
	Skipped int    `json:"skipped"`
	Error   string `json:"error,omitempty"`
}

// SyncSummary holds the result of syncing all accounts.
type SyncSummary struct {
	Accounts  []SyncResult `json:"accounts"`
	TotalNew  int          `json:"total_new"`
	TotalInDB int          `json:"total_in_db"`
}
