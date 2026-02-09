// Package beads provides a shell-out wrapper for the bd (beads) CLI.
//
// Mailbeads delegates all work-tracking state (triage decisions, priorities,
// dependencies) to beads. This package shells out to the bd binary and parses
// its JSON output.
package beads

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Issue is the subset of beads issue fields that mailbeads cares about.
type Issue struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Notes       string `json:"notes,omitempty"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	IssueType   string `json:"issue_type"`
	ExternalRef string `json:"external_ref,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	CloseReason string `json:"close_reason,omitempty"`
}

// Available checks if the bd binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("bd")
	return err == nil
}

// PriorityToBeads maps mailbeads priority strings to beads numeric priorities.
func PriorityToBeads(mbPriority string) string {
	switch mbPriority {
	case "high":
		return "1"
	case "medium":
		return "2"
	case "low":
		return "3"
	case "spam":
		return "4"
	default:
		return "2"
	}
}

// PriorityFromBeads maps beads numeric priorities to mailbeads priority strings.
func PriorityFromBeads(bdPriority int) string {
	switch bdPriority {
	case 0:
		return "high" // P0 critical -> high
	case 1:
		return "high"
	case 2:
		return "medium"
	case 3:
		return "low"
	case 4:
		return "spam"
	default:
		return "medium"
	}
}

// ExternalRef builds the external_ref string for a mailbeads thread.
func ExternalRef(threadID string) string {
	return "mb:" + threadID
}

// Create creates a new beads issue and returns the created issue.
func Create(title, description, notes, priority, category, parent string, labels []string, threadID string) (*Issue, error) {
	args := []string{"create", title,
		"-p", priority,
		"-t", "task",
		"--external-ref", ExternalRef(threadID),
		"--json", "--silent",
	}

	if description != "" {
		args = append(args, "-d", description)
	}
	if notes != "" {
		args = append(args, "--notes", notes)
	}

	// Always include email and triage labels.
	allLabels := []string{"email", "triage"}
	if category != "" {
		allLabels = append(allLabels, category)
	}
	allLabels = append(allLabels, labels...)
	args = append(args, "-l", strings.Join(allLabels, ","))

	if parent != "" {
		args = append(args, "--parent", parent)
	}

	out, err := run(args...)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(out, &issue); err != nil {
		return nil, fmt.Errorf("parse bd create output: %w", err)
	}
	return &issue, nil
}

// Close closes a beads issue with a reason.
func Close(beadID, reason string) error {
	args := []string{"close", beadID, "-q"}
	if reason != "" {
		args = append(args, "-r", reason)
	}
	_, err := run(args...)
	return err
}

// Show returns a beads issue by ID.
func Show(beadID string) (*Issue, error) {
	out, err := run("show", beadID, "--json")
	if err != nil {
		return nil, err
	}

	// bd show returns an array of issues.
	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		// Try single object too, in case bd changes format.
		var issue Issue
		if err2 := json.Unmarshal(out, &issue); err2 != nil {
			return nil, fmt.Errorf("parse bd show output: %w", err)
		}
		return &issue, nil
	}
	if len(issues) == 0 {
		return nil, fmt.Errorf("bead %q not found", beadID)
	}
	return &issues[0], nil
}

// List returns beads issues matching filters.
func List(labels []string, status string, limit int) ([]Issue, error) {
	args := []string{"list", "--json"}

	if len(labels) > 0 {
		args = append(args, "-l", strings.Join(labels, ","))
	}
	if status != "" {
		args = append(args, "-s", status)
	}
	if limit > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", limit))
	}

	out, err := run(args...)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parse bd list output: %w", err)
	}
	return issues, nil
}

// Ready returns actionable beads issues (open, no blockers).
func Ready(labels []string, limit int) ([]Issue, error) {
	args := []string{"ready", "--json"}

	if len(labels) > 0 {
		args = append(args, "-l", strings.Join(labels, ","))
	}
	if limit > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", limit))
	}

	out, err := run(args...)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parse bd ready output: %w", err)
	}
	return issues, nil
}

// AddDep adds a parent-child dependency.
func AddDep(childID, parentID string) error {
	_, err := run("dep", "add", childID, parentID, "-q")
	return err
}

// Comment adds a comment to a beads issue.
func Comment(beadID, text string) error {
	_, err := run("comments", "add", beadID, text, "-q")
	return err
}

// Update updates fields on a beads issue.
func Update(beadID string, fields map[string]string) error {
	args := []string{"update", beadID, "-q"}
	for k, v := range fields {
		args = append(args, "--"+k, v)
	}
	_, err := run(args...)
	return err
}

// discoverBeadsDB walks up from cwd looking for a .beads/ directory
// and returns the path to .beads/beads.db, or empty string if not found.
func discoverBeadsDB() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".beads", "beads.db")
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

// run executes the bd CLI and returns stdout.
// It auto-discovers the beads database path so bd works even when called
// from a nested git submodule directory.
func run(args ...string) ([]byte, error) {
	// If bd can't find its DB (e.g., we're in a nested git repo), help it out.
	dbPath := discoverBeadsDB()
	if dbPath != "" {
		args = append([]string{"--db", dbPath}, args...)
	}

	cmd := exec.Command("bd", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			return nil, fmt.Errorf("bd %s: %s", args[0], stderr)
		}
		return nil, fmt.Errorf("bd %s: %w", args[0], err)
	}
	return out, nil
}
