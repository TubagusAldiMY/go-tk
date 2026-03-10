package ui

import (
	"fmt"
	"strings"
)

// PrintStep prints a numbered step indicator to stdout.
func PrintStep(n, total int, msg string) {
	badge := StyleMuted.Render(fmt.Sprintf("[%d/%d]", n, total))
	fmt.Printf("%s %s\n", badge, msg)
}

// PrintFileCreated prints a "created file" line.
func PrintFileCreated(path string) {
	fmt.Printf("  %s %s\n", StyleSuccess.Render("+"), path)
}

// PrintFileSkipped prints a "skipped (already exists)" line.
func PrintFileSkipped(path string) {
	fmt.Printf("  %s %s %s\n", StyleMuted.Render("~"), path, StyleMuted.Render("(skipped, already exists)"))
}

// PrintDryRun prints what would be created in dry-run mode.
func PrintDryRun(path string) {
	fmt.Printf("  %s %s\n", StyleWarning.Render("→"), path)
}

// PrintSection prints a bold section header.
func PrintSection(title string) {
	fmt.Printf("\n%s\n%s\n", StyleTitle.Render(title), strings.Repeat("─", len(title)+2))
}

// PrintDone prints a final success summary.
func PrintDone(msg string) {
	fmt.Printf("\n%s\n", StyleSuccess.Render("✓ "+msg))
}

// PrintError prints an error line.
func PrintError(msg string) {
	fmt.Printf("\n%s\n", StyleError.Render("✗ "+msg))
}

// PrintHint prints a subtle hint/next-step line.
func PrintHint(msg string) {
	fmt.Printf("  %s\n", StyleMuted.Render("→ "+msg))
}
