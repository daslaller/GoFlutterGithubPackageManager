// Package models/configuration_model.go - Configuration Screen Model
//
// This file implements the package configuration screen where users can
// specify package names, refs, subdirectories, etc.

package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ConfigurationModel handles package configuration
type ConfigurationModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// Configuration state
	currentRepo  int
	currentField int // 0=name, 1=ref, 2=subdir
	packageSpecs []core.PkgSpec
	inputs       []textinput.Model
	complete     bool

	// Styles
	headerStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
	helpStyle     lipgloss.Style
}

// NewConfigurationModel creates a new package configuration model
func NewConfigurationModel(cfg core.Config, logger *core.Logger, shared *AppState) *ConfigurationModel {
	return &ConfigurationModel{
		cfg:          cfg,
		logger:       logger,
		shared:       shared,
		currentRepo:  0,
		currentField: 0,

		// Styles
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).
			Bold(true),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#8B5CF6")).
			Padding(0, 1),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true),
	}
}

// Init initializes the configuration screen
func (m *ConfigurationModel) Init() tea.Cmd {
	m.setupInputs()
	return textinput.Blink
}

// Update handles messages for configuration
func (m *ConfigurationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeys(msg)

	case tea.WindowSizeMsg:
		// Handle window resize - adjust input widths if needed
		maxWidth := msg.Width - 10
		if maxWidth < 20 {
			maxWidth = 20
		}
		for i := range m.inputs {
			m.inputs[i].Width = maxWidth
		}
		return m, nil

	default:
		// Update current input
		if m.currentRepo < len(m.shared.SelectedDependencies) && len(m.inputs) > 0 {
			inputIndex := m.currentRepo*3 + m.currentField
			if inputIndex >= 0 && inputIndex < len(m.inputs) {
				var cmd tea.Cmd
				m.inputs[inputIndex], cmd = m.inputs[inputIndex].Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the configuration screen
func (m *ConfigurationModel) View() string {
	if len(m.shared.SelectedDependencies) == 0 {
		return m.headerStyle.Render("❌ No Repositories Selected") + "\n\nPlease go back and select repositories first.\n\nPress Q to return to main menu"
	}

	var b strings.Builder

	// Header
	b.WriteString(m.headerStyle.Render("🔧 Package Configuration") + "\n")
	b.WriteString(fmt.Sprintf("Configure %d selected packages:\n\n", len(m.shared.SelectedDependencies)))

	// Show current repository being configured
	if m.currentRepo < len(m.shared.SelectedDependencies) {
		repo := m.shared.SelectedDependencies[m.currentRepo]
		b.WriteString(fmt.Sprintf("📦 Configuring: %s/%s\n\n", repo.Owner, repo.Name))

		// Show input fields
		fields := []string{"Package Name:", "Git Ref (branch/tag):", "Subdirectory:"}
		for i, field := range fields {
			if i == m.currentField {
				b.WriteString(m.selectedStyle.Render(field) + "\n")
			} else {
				b.WriteString(m.normalStyle.Render(field) + "\n")
			}

			inputIndex := m.currentRepo*3 + i
			if inputIndex < len(m.inputs) {
				b.WriteString(m.inputs[inputIndex].View() + "\n\n")
			}
		}

		// Progress
		b.WriteString(fmt.Sprintf("Progress: %d/%d packages configured\n\n", m.currentRepo+1, len(m.shared.SelectedDependencies)))
	} else {
		b.WriteString(m.headerStyle.Render("✅ All Packages Configured") + "\n\n")
		b.WriteString("Press Enter to continue to confirmation\n\n")
	}

	// Help
	if m.currentRepo < len(m.shared.SelectedDependencies) {
		b.WriteString(m.helpStyle.Render("tab: next field • shift+tab: prev field • enter: next package • q: back"))
	} else {
		b.WriteString(m.helpStyle.Render("enter: continue • q: back"))
	}

	return b.String()
}

// handleKeys handles keyboard input
func (m *ConfigurationModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, TransitionToScreen(ScreenMainMenu)

	case "tab":
		if m.currentRepo < len(m.shared.SelectedDependencies) {
			m.currentField++
			if m.currentField >= 3 {
				m.currentField = 0
			}
			m.focusCurrentInput()
		}
		return m, nil

	case "shift+tab":
		if m.currentRepo < len(m.shared.SelectedDependencies) {
			m.currentField--
			if m.currentField < 0 {
				m.currentField = 2
			}
			m.focusCurrentInput()
		}
		return m, nil

	case "enter":
		if m.currentRepo >= len(m.shared.SelectedDependencies) {
			// All configured, move to confirmation
			m.generatePackageSpecs()
			return m, TransitionToScreen(ScreenConfirmation)
		} else {
			// Move to next repository
			m.currentRepo++
			m.currentField = 0
			m.focusCurrentInput()
		}
		return m, nil

	default:
		// Pass to current input
		if m.currentRepo < len(m.shared.SelectedDependencies) {
			var cmd tea.Cmd
			inputIndex := m.currentRepo*3 + m.currentField
			if inputIndex < len(m.inputs) {
				m.inputs[inputIndex], cmd = m.inputs[inputIndex].Update(msg)
			}
			return m, cmd
		}
	}

	return m, nil
}

// setupInputs creates text inputs for all repositories
func (m *ConfigurationModel) setupInputs() {
	// Safety check - ensure we have selected repositories
	if len(m.shared.SelectedDependencies) == 0 {
		m.logger.Debug("configuration", "No repositories selected for configuration")
		m.complete = true // Mark as complete to skip configuration
		return
	}

	// Create 3 inputs per repository (name, ref, subdir)
	totalInputs := len(m.shared.SelectedDependencies) * 3
	m.inputs = make([]textinput.Model, totalInputs)

	for i, repo := range m.shared.SelectedDependencies {
		// Package name input
		nameInput := textinput.New()
		nameInput.Placeholder = repo.Name
		nameInput.SetValue(repo.Name)
		nameInput.Width = 40
		m.inputs[i*3] = nameInput

		// Ref input
		refInput := textinput.New()
		refInput.Placeholder = "main"
		refInput.SetValue("main")
		refInput.Width = 40
		m.inputs[i*3+1] = refInput

		// Subdir input
		subdirInput := textinput.New()
		subdirInput.Placeholder = "(optional)"
		subdirInput.Width = 40
		m.inputs[i*3+2] = subdirInput
	}

	m.focusCurrentInput()
}

// focusCurrentInput focuses the current input field
func (m *ConfigurationModel) focusCurrentInput() {
	// Safety check - ensure we have inputs
	if len(m.inputs) == 0 {
		m.logger.Debug("configuration", "No inputs available to focus")
		return
	}

	// Blur all inputs
	for i := range m.inputs {
		m.inputs[i].Blur()
	}

	// Focus current input
	if m.currentRepo < len(m.shared.SelectedDependencies) {
		inputIndex := m.currentRepo*3 + m.currentField
		if inputIndex >= 0 && inputIndex < len(m.inputs) {
			m.inputs[inputIndex].Focus()
		} else {
			m.logger.Debug("configuration", fmt.Sprintf("Invalid input index: %d (total: %d)", inputIndex, len(m.inputs)))
		}
	}
}

// generatePackageSpecs creates package specifications from the inputs
func (m *ConfigurationModel) generatePackageSpecs() {
	// Safety check - ensure we have selected repositories
	if len(m.shared.SelectedDependencies) == 0 {
		m.logger.Debug("configuration", "No repositories to generate package specs for")
		return
	}

	m.packageSpecs = make([]core.PkgSpec, len(m.shared.SelectedDependencies))

	for i, repo := range m.shared.SelectedDependencies {
		// Safety check for input array bounds
		if i*3+2 >= len(m.inputs) {
			m.logger.Debug("configuration", fmt.Sprintf("Insufficient inputs for repo %d", i))
			// Create default spec
			m.packageSpecs[i] = core.PkgSpec{
				Name:   repo.Name,
				URL:    repo.URL,
				Ref:    "main",
				Subdir: "",
			}
			continue
		}

		name := m.inputs[i*3].Value()
		if name == "" {
			name = repo.Name
		}

		ref := m.inputs[i*3+1].Value()
		if ref == "" {
			ref = "main"
		}

		subdir := m.inputs[i*3+2].Value()

		m.packageSpecs[i] = core.PkgSpec{
			Name:   name,
			URL:    repo.URL,
			Ref:    ref,
			Subdir: subdir,
		}
	}

	m.shared.PackageSpecs = m.packageSpecs
	m.logger.Info("configuration", fmt.Sprintf("Generated %d package specifications", len(m.packageSpecs)))
}
