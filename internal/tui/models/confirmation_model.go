// Package models/confirmation_model.go - Confirmation Screen Model

package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ConfirmationModel handles change confirmation
type ConfirmationModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// State
	choice int // 0=confirm, 1=cancel

	// Styles
	headerStyle   lipgloss.Style
	packageStyle  lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
	warningStyle  lipgloss.Style
}

// NewConfirmationModel creates a new confirmation model
func NewConfirmationModel(cfg core.Config, logger *core.Logger, shared *AppState) *ConfirmationModel {
	return &ConfirmationModel{
		cfg:    cfg,
		logger: logger,
		shared: shared,
		choice: 0, // Default to confirm

		// Styles
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).
			Bold(true),

		packageStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Padding(0, 2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("10")),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#8B5CF6")).
			Padding(0, 1).
			Bold(true),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("202")).
			Bold(true),
	}
}

// Init initializes the confirmation screen
func (m *ConfirmationModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for confirmation
func (m *ConfirmationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeys(msg)

	case tea.WindowSizeMsg:
		// Handle window resize gracefully
		return m, nil
	}
	return m, nil
}

// View renders the confirmation screen
func (m *ConfirmationModel) View() string {
	if len(m.shared.PackageSpecs) == 0 {
		return m.warningStyle.Render("❌ No Package Specifications") + "\n\nNo packages have been configured yet.\n\nPress Q to return to main menu"
	}

	var b strings.Builder

	// Header
	b.WriteString(m.headerStyle.Render("✅ Confirm Package Installation") + "\n\n")
	b.WriteString(fmt.Sprintf("Review the %d packages that will be added:\n\n", len(m.shared.PackageSpecs)))

	// Package list
	for i, spec := range m.shared.PackageSpecs {
		packageInfo := fmt.Sprintf("📦 %s\n", spec.Name)
		packageInfo += fmt.Sprintf("   URL: %s\n", spec.URL)
		packageInfo += fmt.Sprintf("   Ref: %s\n", spec.Ref)
		if spec.Subdir != "" {
			packageInfo += fmt.Sprintf("   Subdir: %s\n", spec.Subdir)
		}

		if i < len(m.shared.PackageSpecs)-1 {
			packageInfo += "\n"
		}

		b.WriteString(m.packageStyle.Render(packageInfo))
		if i < len(m.shared.PackageSpecs)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")

	// Backup warning
	b.WriteString(m.warningStyle.Render("⚠️  A backup of pubspec.yaml will be created") + "\n\n")

	// Choice buttons
	choices := []string{"✅ Confirm Installation", "❌ Cancel"}
	for i, choice := range choices {
		if i == m.choice {
			b.WriteString(m.selectedStyle.Render(choice))
		} else {
			b.WriteString(m.normalStyle.Render(choice))
		}
		if i < len(choices)-1 {
			b.WriteString("    ")
		}
	}

	b.WriteString("\n\n")

	// Help
	b.WriteString(m.normalStyle.Render("left/right: select • enter: confirm choice • q: back to menu"))

	return b.String()
}

// handleKeys handles keyboard input
func (m *ConfirmationModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, TransitionToScreen(ScreenMainMenu)

	case "left", "h":
		m.choice = 0 // Confirm
		return m, nil

	case "right", "l":
		m.choice = 1 // Cancel
		return m, nil

	case "y":
		m.choice = 0
		return m.confirm()

	case "n":
		m.choice = 1
		return m.confirm()

	case "enter":
		return m.confirm()
	}

	return m, nil
}

// confirm executes the user's choice
func (m *ConfirmationModel) confirm() (tea.Model, tea.Cmd) {
	if m.choice == 0 {
		// Confirm installation
		m.logger.Info("confirmation", "User confirmed package installation")
		return m, TransitionToScreen(ScreenExecution)
	} else {
		// Cancel
		m.logger.Info("confirmation", "User cancelled package installation")
		return m, TransitionToScreen(ScreenMainMenu)
	}
}
