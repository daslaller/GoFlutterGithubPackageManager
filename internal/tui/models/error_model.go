// Package models/error_model.go - Error Screen Model
//
// This file implements a generic error screen that can display errors
// and provide recovery options.

package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ErrorModel handles error display and recovery
type ErrorModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// Error details
	title        string
	message      string
	err          error
	returnScreen AppScreen

	// Styles
	headerStyle lipgloss.Style
	errorStyle  lipgloss.Style
	normalStyle lipgloss.Style
}

// ErrorData contains information about the error to display
type ErrorData struct {
	Title        string
	Message      string
	Error        error
	ReturnScreen AppScreen
}

// NewErrorModel creates a new error model
func NewErrorModel(cfg core.Config, logger *core.Logger, shared *AppState) *ErrorModel {
	return &ErrorModel{
		cfg:          cfg,
		logger:       logger,
		shared:       shared,
		returnScreen: ScreenMainMenu, // Default return to main menu

		// Styles
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("52")).
			Padding(1, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("196")),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
}

// Init initializes the error screen
func (m *ErrorModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the error screen
func (m *ErrorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "enter", "esc":
			return m, TransitionToScreen(m.returnScreen)
		}
	}
	return m, nil
}

// View renders the error screen
func (m *ErrorModel) View() string {
	var b strings.Builder

	// Header
	title := m.title
	if title == "" {
		title = "❌ An Error Occurred"
	}
	b.WriteString(m.headerStyle.Render(title) + "\n\n")

	// Error box
	errorContent := ""
	if m.message != "" {
		errorContent = m.message + "\n"
	}
	if m.err != nil {
		errorContent += fmt.Sprintf("Technical details: %s", m.err.Error())
	}
	if errorContent == "" {
		errorContent = "An unexpected error occurred."
	}

	b.WriteString(m.errorStyle.Render(errorContent) + "\n\n")

	// Recovery instructions
	b.WriteString(m.normalStyle.Render("What you can do:") + "\n")
	b.WriteString("• Press Enter or Q to return to the main menu\n")
	b.WriteString("• Check your internet connection\n")
	b.WriteString("• Verify your GitHub CLI authentication (gh auth status)\n")
	b.WriteString("• Try the operation again\n\n")

	b.WriteString(m.normalStyle.Render("Press Enter or Q to continue"))

	return b.String()
}

// SetError sets the error details for display
func (m *ErrorModel) SetError(data ErrorData) {
	m.title = data.Title
	m.message = data.Message
	m.err = data.Error
	if data.ReturnScreen != 0 {
		m.returnScreen = data.ReturnScreen
	}

	// Log the error
	if m.err != nil {
		m.logger.Error("error_screen", m.err)
	}
}
