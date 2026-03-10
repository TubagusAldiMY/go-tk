// Package ui provides Bubbletea / Lipgloss terminal UI components.
package ui

import "github.com/charmbracelet/lipgloss"

// Colour palette used across all go-tk TUI components.
var (
	ColorPrimary  = lipgloss.Color("#7C3AED") // violet-600
	ColorSuccess  = lipgloss.Color("#16A34A") // green-600
	ColorWarning  = lipgloss.Color("#D97706") // amber-600
	ColorError    = lipgloss.Color("#DC2626") // red-600
	ColorMuted    = lipgloss.Color("#6B7280") // gray-500
	ColorSelected = lipgloss.Color("#DDD6FE") // violet-200

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StyleError = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleSelectedItem = lipgloss.NewStyle().
				Foreground(ColorSelected).
				Bold(true)

	StyleBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2)
)

// Banner renders the go-tk ASCII banner.
func Banner() string {
	return StyleBanner.Render("go-tk · Go Backend Toolkit")
}

// SuccessMsg formats a success message with a checkmark.
func SuccessMsg(msg string) string {
	return StyleSuccess.Render("✓ ") + msg
}

// ErrorMsg formats an error message with a cross.
func ErrorMsg(msg string) string {
	return StyleError.Render("✗ ") + msg
}

// InfoMsg formats an informational message.
func InfoMsg(msg string) string {
	return StyleMuted.Render("→ ") + msg
}
