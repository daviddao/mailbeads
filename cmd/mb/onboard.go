package main

import (
	"fmt"

	"github.com/daviddao/mailbeads/internal/display"
	"github.com/spf13/cobra"
)

const agentsMDSnippet = `## Email Inbox Triage

This project uses **mb (mailbeads)** for email inbox triage.
Run ` + "`mb prime`" + ` for workflow context.

**Quick reference:**
- ` + "`mb sync`" + ` - Fetch latest emails from Gmail
- ` + "`mb untriaged --json`" + ` - List threads needing triage
- ` + "`mb show THREAD_ID --json`" + ` - Read thread detail
- ` + "`mb triage THREAD_ID --action \"...\" --priority high`" + ` - Create triage entry
- ` + "`mb ready --json`" + ` - Actionable inbox items
- ` + "`mb done ID`" + ` / ` + "`mb dismiss ID`" + ` - Manage triage status
- ` + "`mb stats --json`" + ` - Inbox statistics

For full workflow details: ` + "`mb prime`"

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Display minimal snippet for AGENTS.md",
	Long: `Display a minimal snippet to add to AGENTS.md for mb integration.

This outputs a small snippet that points to 'mb prime' for full
workflow context. This approach keeps AGENTS.md lean while mb prime
provides dynamic, always-current workflow details.`,
	Run: func(cmd *cobra.Command, args []string) {
		b := display.Bold.Render
		a := display.Success.Render

		fmt.Printf("\n%s\n\n", b("mb Onboarding"))
		fmt.Println("Add this snippet to AGENTS.md (or your agent instructions file):")
		fmt.Println()
		fmt.Println(display.Dim.Render("--- BEGIN AGENTS.MD CONTENT ---"))
		fmt.Println(agentsMDSnippet)
		fmt.Println(display.Dim.Render("--- END AGENTS.MD CONTENT ---"))
		fmt.Println()

		fmt.Println(b("How it works:"))
		fmt.Printf("   • %s provides dynamic workflow context for AI agents\n", a("mb prime"))
		fmt.Printf("   • %s provides full triage instructions with --json examples\n", a("mb prime --full"))
		fmt.Println("   • AGENTS.md only needs this minimal pointer, not full instructions")
		fmt.Println()
		fmt.Printf("%s\n\n", display.Success.Render("This keeps AGENTS.md lean while mb prime provides up-to-date workflow details."))
	},
}

func init() {
	rootCmd.AddCommand(onboardCmd)
}
