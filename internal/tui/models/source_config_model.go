// Package models/source_config_model.go - Source Project Configuration Screen
//
// This file implements configuration for the selected source project (save location, name editing).

package models

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// SourceConfigModel handles source project configuration
type SourceConfigModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// UI components
	picker     Model
	pathInput  textinput.Model
	nameInput  textinput.Model
	showPicker bool

	// State
	focusIndex int // 0 = path, 1 = name, 2 = continue

	// Styles
	headerStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
	helpStyle     lipgloss.Style
}

// NewSourceConfigModel creates a new source configuration model
func NewSourceConfigModel(cfg core.Config, logger *core.Logger, shared *AppState) *SourceConfigModel {
	// Initialize reusable picker in directory mode
	var startDir string
	if cwd, err := os.Getwd(); err == nil {
		startDir = cwd
	} else {
		startDir = "."
	}
	pm := New(true, nil, startDir)

	pathInput := textinput.New()
	pathInput.Placeholder = "./projects"
	pathInput.SetValue("./projects")
	pathInput.Width = 50

	nameInput := textinput.New()
	nameInput.Placeholder = "project-name"
	if shared.SourceProject != nil {
		nameInput.SetValue(shared.SourceProject.Name)
	}
	nameInput.Width = 50

	return &SourceConfigModel{
		cfg:        cfg,
		logger:     logger,
		shared:     shared,
		picker:     pm,
		pathInput:  pathInput,
		nameInput:  nameInput,
		showPicker: false,
		focusIndex: 0,

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0EA5E9")).
			Bold(true),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0EA5E9")).
			Padding(0, 1),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151")),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			Italic(true),
	}
}

// Init initializes the source config screen
func (m *SourceConfigModel) Init() tea.Cmd {
	m.focusIndex = 0
	m.showPicker = false
	m.pathInput.Focus()
	return textinput.Blink
}

// Update handles messages for source configuration
func (m *SourceConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If filepicker is shown, handle its updates
	if m.showPicker {
		// Intercept quit keys to close picker without quitting the app
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "esc", "q", "ctrl+c":
				m.showPicker = false
				m.updateFocus()
				return m, nil
			}
		}

		var cmd tea.Cmd
		var tm tea.Model
		tm, cmd = m.picker.Update(msg)
		m.picker = tm.(Model)

		// If a selection was made, capture and close
		if m.picker.Selected != "" {
			m.pathInput.SetValue(m.picker.Selected)
			m.showPicker = false
			m.logger.Info("source_config", fmt.Sprintf("Selected directory: %s", m.picker.Selected))
		}

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeys(msg)

	default:
		// Update the active input
		var cmd tea.Cmd
		if m.focusIndex == 0 {
			m.pathInput, cmd = m.pathInput.Update(msg)
		} else if m.focusIndex == 1 {
			m.nameInput, cmd = m.nameInput.Update(msg)
		}
		return m, cmd
	}
}

// View renders the source config screen
func (m *SourceConfigModel) View() string {
	// If filepicker is active, show it
	if m.showPicker {
		var b strings.Builder
		b.WriteString("\n  ")
		b.WriteString(m.headerStyle.Render("📁 Select Save Location"))
		b.WriteString("\n\n")
		b.WriteString(m.picker.View())
		b.WriteString("\n\n")
		b.WriteString(m.helpStyle.Render("Navigate: ↑/↓ • Select: enter • Cancel: esc"))
		return b.String()
	}

	var b strings.Builder

	// Beautiful bordered header with warm amber theme
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F59E0B")).
		Foreground(lipgloss.Color("#F59E0B")).
		Padding(1, 2).
		Align(lipgloss.Center).
		Width(62).
		Bold(true).
		Render("⚙️ Configure Source Project")

	b.WriteString(headerBox + "\n\n")

	if m.shared.SourceProject != nil {
		titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#0EA5E9")).Bold(true)
		b.WriteString(fmt.Sprintf("Selected project: %s\n\n", titleStyle.Render(m.shared.SourceProject.Name)))
	}

	// Path input with browse option
	pathLabel := "Save location:"
	if m.focusIndex == 0 {
		pathLabel = m.selectedStyle.Render("► " + pathLabel)
	} else {
		pathLabel = m.normalStyle.Render("  " + pathLabel)
	}
	b.WriteString(pathLabel + "\n")
	b.WriteString("  " + m.pathInput.View() + "\n")
	if m.focusIndex == 0 {
		b.WriteString("  " + m.helpStyle.Render("type path or press 'b' to browse...") + "\n")
	}
	b.WriteString("\n")

	// Name input
	nameLabel := "Project name:"
	if m.focusIndex == 1 {
		nameLabel = m.selectedStyle.Render("► " + nameLabel)
	} else {
		nameLabel = m.normalStyle.Render("  " + nameLabel)
	}
	b.WriteString(nameLabel + "\n")
	b.WriteString("  " + m.nameInput.View() + "\n\n")

	// Continue button
	continueLabel := "Continue to package selection"
	if m.focusIndex == 2 {
		continueLabel = m.selectedStyle.Render("► " + continueLabel)
	} else {
		continueLabel = m.normalStyle.Render("  " + continueLabel)
	}
	b.WriteString(continueLabel + "\n\n")

	// Help text
	b.WriteString(m.helpStyle.Render("tab: next field • shift+tab: previous • enter: select/continue • q: back"))

	return b.String()
}

// handleKeys handles keyboard input
func (m *SourceConfigModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, TransitionToScreen(ScreenMainMenu)

	case "b", "B":
		// Open filepicker for directory browsing (only when on path field)
		if m.focusIndex == 0 {
			m.showPicker = true
			return m, m.picker.Init()
		}
		return m, nil

	case "tab":
		m.focusIndex++
		if m.focusIndex > 2 {
			m.focusIndex = 0
		}
		m.updateFocus()
		return m, nil

	case "shift+tab":
		m.focusIndex--
		if m.focusIndex < 0 {
			m.focusIndex = 2
		}
		m.updateFocus()
		return m, nil

	case "enter":
		if m.focusIndex == 2 {
			// Save configuration and continue to package selection
			m.saveConfig()

			// Copy repos from AvailableSourceRepos to AvailableDependencies for package selection
			m.shared.AvailableDependencies = m.shared.AvailableSourceRepos
			m.shared.AvailableSourceRepos = nil

			return m, TransitionToScreen(ScreenDependencySelection)
		}
		// On input fields, Enter moves to next field
		m.focusIndex++
		if m.focusIndex > 2 {
			m.focusIndex = 0
		}
		m.updateFocus()
		return m, nil

	default:
		// Pass to active input
		var cmd tea.Cmd
		if m.focusIndex == 0 {
			m.pathInput, cmd = m.pathInput.Update(msg)
		} else if m.focusIndex == 1 {
			m.nameInput, cmd = m.nameInput.Update(msg)
		}
		return m, cmd
	}
}

// updateFocus updates which input has focus
func (m *SourceConfigModel) updateFocus() {
	if m.focusIndex == 0 {
		m.pathInput.Focus()
		m.nameInput.Blur()
	} else if m.focusIndex == 1 {
		m.pathInput.Blur()
		m.nameInput.Focus()
	} else {
		m.pathInput.Blur()
		m.nameInput.Blur()
	}
}

// saveConfig saves the configuration to shared state
func (m *SourceConfigModel) saveConfig() {
	if m.shared.SourceProject != nil {
		m.shared.SourceProject.Path = strings.TrimSpace(m.pathInput.Value())
		m.shared.SourceProject.Name = strings.TrimSpace(m.nameInput.Value())

		m.logger.Info("source_config", fmt.Sprintf("Configured source: path=%s, name=%s",
			m.shared.SourceProject.Path, m.shared.SourceProject.Name))
	}
}
