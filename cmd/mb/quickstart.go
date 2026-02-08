package main

import (
	"fmt"

	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Quick start guide for mb",
	Long:  "Display a quick start guide showing common mb workflows and patterns.",
	Run: func(cmd *cobra.Command, args []string) {
		b := display.Bold.Render
		a := display.Success.Render
		d := display.Dim.Render

		fmt.Printf("\n%s\n\n", b("mb — Email Inbox Triage for AI Agents"))
		fmt.Println("Sync Gmail, triage threads, track what needs attention.")
		fmt.Println()

		fmt.Println(b("GETTING STARTED"))
		fmt.Printf("  %s           Initialize .mailbeads/ in your project\n", a("mb init"))
		fmt.Printf("                   Creates .mailbeads/mail.db next to your .git root\n\n")
		fmt.Printf("  %s           Fetch emails from both Gmail accounts\n", a("mb sync"))
		fmt.Printf("  %s  Fetch from a single account\n", a("mb sync --account user@example.com"))
		fmt.Printf("  %s    Force full 72h re-scan\n\n", a("mb sync --full"))

		fmt.Println(b("TRIAGING EMAILS"))
		fmt.Printf("  %s      List threads needing triage\n", a("mb untriaged"))
		fmt.Printf("  %s  View thread with all emails\n", a("mb show THREAD_ID"))
		fmt.Println()
		fmt.Printf("  %s\n", a(`mb triage THREAD_ID --action "Reply with agenda" --priority high`))
		fmt.Printf("  %s\n\n", d("  Create or update a triage entry for a thread"))

		fmt.Println(b("VIEWING YOUR INBOX"))
		fmt.Printf("  %s          Pending triage items sorted by priority\n", a("mb inbox"))
		fmt.Printf("  %s  Include done/dismissed items\n", a("mb inbox --all"))
		fmt.Printf("  %s          Actionable items (not snoozed, not blocked)\n", a("mb ready"))
		fmt.Printf("  %s  Full inbox statistics\n\n", a("mb stats"))

		fmt.Println(b("MANAGING TRIAGE"))
		fmt.Printf("  %s    Mark triage entry as done\n", a("mb done ID"))
		fmt.Printf("  %s Mark multiple at once\n", a("mb done ID1 ID2"))
		fmt.Printf("  %s Dismiss spam/irrelevant\n", a("mb dismiss ID"))
		fmt.Println()
		fmt.Printf("  %s\n\n", d("IDs support partial matching — 'mb done abc' matches IDs starting with 'abc'"))

		fmt.Println(b("PRIORITY LEVELS"))
		fmt.Printf("  %s   Direct questions, time-sensitive, approval requests\n", display.HighStyle.Render("high"))
		fmt.Printf("  %s FYI threads, project updates, relevant newsletters\n", display.MediumStyle.Render("medium"))
		fmt.Printf("  %s    Receipts, automated confirmations, CI notifications\n", display.LowStyle.Render("low"))
		fmt.Printf("  %s   Marketing, cold outreach, unsolicited sales\n\n", display.SpamStyle.Render("spam"))

		fmt.Println(b("JSON OUTPUT"))
		fmt.Printf("  All commands support %s for machine-readable output:\n", a("--json"))
		fmt.Printf("  %s\n", a("mb inbox --json"))
		fmt.Printf("  %s\n", a("mb untriaged --json"))
		fmt.Printf("  %s\n\n", a("mb stats --json"))

		fmt.Println(b("ACCOUNTS"))
		fmt.Println("  Accounts are auto-discovered from */credentials.json in the project root.")
		fmt.Printf("  Use %s to target a specific account.\n\n", a("--account user@example.com"))

		fmt.Println(b("AGENT INTEGRATION"))
		fmt.Println("  mb is designed for AI agent workflows:")
		fmt.Printf("    • Agents run %s to fetch latest emails\n", a("mb sync"))
		fmt.Printf("    • %s lists threads needing analysis\n", a("mb untriaged --json"))
		fmt.Printf("    • %s reads full thread detail\n", a("mb show THREAD_ID --json"))
		fmt.Printf("    • Agents write triage entries via %s\n", a("mb triage"))
		fmt.Printf("    • %s shows what's actionable\n\n", a("mb ready --json"))

		fmt.Printf("%s Run %s to get AGENTS.md integration snippet.\n",
			display.Success.Render("Ready!"), a("mb onboard"))
		fmt.Printf("Run %s to get AI-optimized workflow context.\n\n", a("mb prime"))
	},
}

func init() {
	rootCmd.AddCommand(quickstartCmd)
}
