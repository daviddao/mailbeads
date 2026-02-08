// Package display provides terminal formatting for mailbeads output.
package display

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Detect if we're in a TTY for color support.
	isTTY = lipgloss.HasDarkBackground()

	// Styles
	Muted    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	Dim      = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ca3af"))
	Bold     = lipgloss.NewStyle().Bold(true)
	Success  = lipgloss.NewStyle().Foreground(lipgloss.Color("#16a34a"))
	ErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#dc2626"))

	HighStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#dc2626"))
	MediumStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#d97706"))
	LowStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	SpamStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ca3af"))
)

// PriorityDot returns a colored dot for a priority level.
func PriorityDot(priority string) string {
	switch priority {
	case "high":
		return HighStyle.Render("●")
	case "medium":
		return MediumStyle.Render("○")
	case "low":
		return LowStyle.Render("○")
	case "spam":
		return SpamStyle.Render("◌")
	default:
		return Dim.Render("·")
	}
}

// PriorityLabel returns a styled priority label.
func PriorityLabel(priority string) string {
	label := strings.ToUpper(priority)
	switch priority {
	case "high":
		return HighStyle.Render(fmt.Sprintf("%-6s", label))
	case "medium":
		return MediumStyle.Render(fmt.Sprintf("%-6s", label))
	case "low":
		return LowStyle.Render(fmt.Sprintf("%-6s", label))
	case "spam":
		return SpamStyle.Render(fmt.Sprintf("%-6s", label))
	default:
		return fmt.Sprintf("%-6s", label)
	}
}

// AccountLabel returns a short label for an account.
// Derives the label from the domain (e.g., "user@example.com" -> "example").
func AccountLabel(account string) string {
	if idx := strings.Index(account, "@"); idx > 0 {
		domain := account[idx+1:]
		// Use the domain name without TLD (e.g., "company.com" -> "company")
		if dotIdx := strings.Index(domain, "."); dotIdx > 0 {
			return domain[:dotIdx]
		}
		return domain
	}
	return account
}

// TimeAgo formats an ISO date string as a relative time.
func TimeAgo(isoDate string) string {
	if isoDate == "" {
		return ""
	}

	// Try multiple formats
	var t time.Time
	var err error
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02 15:04:05", time.RFC3339Nano} {
		t, err = time.Parse(layout, isoDate)
		if err == nil {
			break
		}
	}
	if err != nil {
		return isoDate[:min(10, len(isoDate))]
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}

// Truncate shortens a string to maxLen, adding ellipsis if needed.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// SuccessMsg prints a green checkmark + message.
func SuccessMsg(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(Success.Render("✓") + " " + msg)
}

// ErrorMsg prints a red X + message to stderr.
func ErrorMsg(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, ErrStyle.Render("✗")+" "+msg)
}

// Header prints a section header.
func Header(title string) {
	fmt.Println(Bold.Render(title))
}

// SubHeader prints a dim subsection label.
func SubHeader(title string) {
	fmt.Println(Muted.Render(title))
}

// EmailTree prints an email in a tree-style format.
// connector is one of "┌─", "├─", "└─"
func EmailTree(connector, from, date, body string) {
	fromStr := Bold.Render(from)
	dateStr := Dim.Render(TimeAgo(date))
	fmt.Printf("  %s %s  ·  %s\n", Muted.Render(connector), fromStr, dateStr)
	if body != "" {
		// Indent body lines under the connector
		prefix := "  │  "
		if connector == "└─" {
			prefix = "     "
		}
		lines := strings.Split(strings.TrimSpace(body), "\n")
		maxLines := 4
		for i, line := range lines {
			if i >= maxLines {
				fmt.Printf("%s%s\n", Muted.Render(prefix), Dim.Render(fmt.Sprintf("... (%d more lines)", len(lines)-maxLines)))
				break
			}
			trimmed := Truncate(strings.TrimSpace(line), 80)
			fmt.Printf("%s%s\n", Muted.Render(prefix), trimmed)
		}
	}
}

// TriageBadge prints a styled triage label.
func TriageBadge(priority, action string) string {
	dot := PriorityDot(priority)
	label := PriorityLabel(priority)
	return fmt.Sprintf("%s %s %s", dot, label, action)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
