// Package models/picker.go - File Picker Component
//
// Reusable directory picker component for source configuration.
// Wraps the Charmbracelet bubbles/filepicker with sensible defaults.

package models

// Integrated picker model used by SourceConfig and other screens

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is a reusable Bubble Tea model wrapping the bubbles/filepicker component.
//
// Result fields:
//   - Selected: absolute path of the selected file or directory. Empty if none.
//   - Err: last transient error (e.g., trying to open a disabled file). Cleared automatically.
//
// Behavior:
//   - Press Enter to confirm the highlighted item.
//   - In directory mode (SelectDir=true), pressing Enter on a file selects its parent directory.
//   - Press q, Esc, or Ctrl+C to quit without selection.
//
// This model can be embedded into larger TUI apps or used standalone via tea.NewProgram(&m).Run().
// After Run() returns, cast the returned tea.Model back to picker.Model and read Selected.
// See README for a complete example.
type Model struct {
	Filepicker filepicker.Model
	Selected   string
	Quitting   bool
	Err        error
	SelectDir  bool
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg { return clearErrorMsg{} })
}

// New constructs a Model.
// Params:
//   - selectDir: if true, enable directory selection mode. Enter on a directory
//     selects that directory; Enter on a file selects its parent directory.
//   - allowedTypes: allowed file extensions (e.g., []string{".go", ".md"});
//     nil/empty means any type.
//   - startDir: directory to start in; if empty, defaults to user's home directory.
//
// Note: Confirmation is done with Enter. Quit with q, Esc, or Ctrl+C.
func New(selectDir bool, allowedTypes []string, startDir string) Model {
	fp := filepicker.New()
	fp.Height = 15 // Set a reasonable default height for better visibility

	if len(allowedTypes) > 0 {
		fp.AllowedTypes = allowedTypes
	}

	if startDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			fp.CurrentDirectory = home
		}
	} else {
		fp.CurrentDirectory = startDir
	}

	if selectDir {
		fp.DirAllowed = true
	}

	return Model{
		Filepicker: fp,
		SelectDir:  selectDir,
	}
}

func (m Model) Init() tea.Cmd {
	return m.Filepicker.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		}
	case clearErrorMsg:
		m.Err = nil
	case tea.WindowSizeMsg:
		// Pass window size to underlying filepicker
		m.Filepicker.SetHeight(max(msg.Height-8, 10))
	}

	var cmd tea.Cmd
	m.Filepicker, cmd = m.Filepicker.Update(msg)

	// Did the user select a file?
	if didSelect, path := m.Filepicker.DidSelectFile(msg); didSelect {
		if m.SelectDir {
			m.Selected = filepath.Dir(path)
		} else {
			m.Selected = path
		}
	}

	// Did the user select a disabled file?
	if didSelect, path := m.Filepicker.DidSelectDisabledFile(msg); didSelect {
		m.Err = errors.New(path + " is not valid.")
		m.Selected = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}

func (m Model) View() string {
	if m.Quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n  ")
	if m.Err != nil {
		b.WriteString(m.Filepicker.Styles.DisabledFile.Render(m.Err.Error()))
	} else if m.Selected == "" {
		if m.SelectDir {
			b.WriteString("Pick a directory:")
		} else {
			b.WriteString("Pick a file:")
		}
	} else {
		if m.SelectDir {
			b.WriteString("Selected directory: " + m.Filepicker.Styles.Selected.Render(m.Selected))
		} else {
			b.WriteString("Selected file: " + m.Filepicker.Styles.Selected.Render(m.Selected))
		}
	}
	b.WriteString("\n\n" + m.Filepicker.View() + "\n")
	return b.String()
}
