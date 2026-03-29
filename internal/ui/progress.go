package ui

import (
	"fmt"
	"strings"
)

// Quiet suppresses all informational output when true.
// Set via the global --quiet / -q persistent flag.
// PrintError always prints regardless of this flag.
var Quiet bool

// PrintStep prints a numbered step indicator to stdout.
func PrintStep(n, total int, msg string) {
	if Quiet {
		return
	}
	badge := StyleMuted.Render(fmt.Sprintf("[%d/%d]", n, total))
	fmt.Printf("%s %s\n", badge, msg)
}

// PrintFileCreated prints a "created file" line.
func PrintFileCreated(path string) {
	if Quiet {
		return
	}
	fmt.Printf("  %s %s\n", StyleSuccess.Render("+"), path)
}

// PrintFileSkipped prints a "skipped (already exists)" line.
func PrintFileSkipped(path string) {
	if Quiet {
		return
	}
	fmt.Printf("  %s %s %s\n", StyleMuted.Render("~"), path, StyleMuted.Render("(skipped, already exists)"))
}

// PrintDryRun prints what would be created in dry-run mode.
func PrintDryRun(path string) {
	if Quiet {
		return
	}
	fmt.Printf("  %s %s\n", StyleWarning.Render("→"), path)
}

// PrintSection prints a bold section header.
func PrintSection(title string) {
	if Quiet {
		return
	}
	fmt.Printf("\n%s\n%s\n", StyleTitle.Render(title), strings.Repeat("─", len(title)+2))
}

// PrintDone prints a final success summary.
// Always prints — not suppressed by --quiet (it is the "final status" output).
func PrintDone(msg string) {
	fmt.Printf("\n%s\n", StyleSuccess.Render("✓ "+msg))
}

// PrintBanner prints the go-tk ASCII banner.
// Suppressed by --quiet.
func PrintBanner() {
	if Quiet {
		return
	}
	fmt.Println()
	fmt.Println(Banner())
	fmt.Println()
}

// PrintError prints an error line to stdout. Always prints — not suppressed by --quiet.
func PrintError(msg string) {
	fmt.Printf("\n%s\n", StyleError.Render("✗ "+msg))
}

// PrintHint prints a subtle hint/next-step line.
func PrintHint(msg string) {
	if Quiet {
		return
	}
	fmt.Printf("  %s\n", StyleMuted.Render("→ "+msg))
}
