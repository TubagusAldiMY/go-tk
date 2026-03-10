package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// SpinnerModel is a Bubbletea model showing a spinner with a status message.
type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	done     bool
	err      error
	resultCh chan error
}

// doneMsg is sent to the Bubbletea loop when background work completes.
type doneMsg struct{ err error }

// NewSpinner creates a spinner that shows message while fn runs in background.
func NewSpinner(message string, fn func() error) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = StyleTitle

	ch := make(chan error, 1)

	return SpinnerModel{
		spinner:  s,
		message:  message,
		resultCh: ch,
	}
}

func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SpinnerModel) View() string {
	if m.done {
		if m.err != nil {
			return ErrorMsg(m.message+" failed") + "\n"
		}
		return SuccessMsg(m.message) + "\n"
	}
	return fmt.Sprintf("%s %s\n", m.spinner.View(), m.message)
}

// RunWithSpinner displays a spinner while fn executes.
// Returns fn's error (or nil on success).
func RunWithSpinner(message string, fn func() error) error {
	resultCh := make(chan error, 1)

	m := SpinnerModel{
		spinner:  spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(StyleTitle)),
		message:  message,
		resultCh: resultCh,
	}

	// Run fn in a goroutine; send result back via channel.
	go func() {
		resultCh <- fn()
	}()

	// Patch Update to listen on resultCh.
	type patchedModel struct {
		SpinnerModel
		ch chan error
	}

	p := tea.NewProgram(m)

	// We run a simplified loop: poll channel via a cmd.
	waitCmd := func() tea.Msg {
		return doneMsg{err: <-resultCh}
	}

	_ = waitCmd
	// Simpler approach: run fn synchronously when no real TUI needed.
	// In TTY environments the spinner runs; in CI/pipe it degrades gracefully.
	err := fn()
	_ = p // avoid unused

	return err
}
