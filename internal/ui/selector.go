package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectOption represents a single selectable item.
type SelectOption struct {
	Label string
	Value string
	Desc  string // shown as subtitle in the list
}

func (o SelectOption) Title() string       { return o.Label }
func (o SelectOption) Description() string { return o.Desc }
func (o SelectOption) FilterValue() string { return o.Label }

// SelectorModel is a Bubbletea model for selecting one option from a list.
type SelectorModel struct {
	list     list.Model
	prompt   string
	selected *SelectOption
	quitting bool
}

// NewSelector creates a selector model for the given prompt and options.
func NewSelector(prompt string, options []SelectOption) SelectorModel {
	items := make([]list.Item, len(options))
	for i, o := range options {
		items[i] = o
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorPrimary).
		PaddingLeft(1)

	l := list.New(items, delegate, 0, min(len(items)+4, 12))
	l.Title = prompt
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = StyleTitle
	l.SetShowHelp(false)

	return SelectorModel{list: l, prompt: prompt}
}

func (m SelectorModel) Init() tea.Cmd { return nil }

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(SelectOption); ok {
				m.selected = &item
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SelectorModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

// Selected returns the chosen option or nil if the user quit.
func (m SelectorModel) Selected() *SelectOption { return m.selected }

// RunSelector runs a TUI selector and returns the chosen value.
// Returns empty string if the user cancelled.
func RunSelector(prompt string, options []SelectOption) (string, error) {
	m := NewSelector(prompt, options)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running selector: %w", err)
	}

	sm, ok := finalModel.(SelectorModel)
	if !ok || sm.selected == nil {
		return "", nil
	}

	return sm.selected.Value, nil
}

// RunConfirm shows a yes/no prompt and returns the user's answer.
func RunConfirm(prompt string) (bool, error) {
	options := []SelectOption{
		{Label: "Yes", Value: "yes"},
		{Label: "No", Value: "no"},
	}
	val, err := RunSelector(prompt, options)
	if err != nil {
		return false, err
	}
	return val == "yes", nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BuildSummary formats a multi-line summary of selected options.
func BuildSummary(fields map[string]string) string {
	var sb strings.Builder
	maxKey := 0
	for k := range fields {
		if len(k) > maxKey {
			maxKey = len(k)
		}
	}
	for k, v := range fields {
		pad := strings.Repeat(" ", maxKey-len(k))
		sb.WriteString(StyleMuted.Render(k+pad+" : ") + v + "\n")
	}
	return sb.String()
}
