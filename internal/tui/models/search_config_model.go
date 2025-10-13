// Package models/search_config_model.go - Directory Search Configuration Screen Model
//
// This file implements the directory search configuration screen where users can
// configure search paths, depth, and other directory scanning settings.
// Matches the shell script's configure_search_settings() function.

package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// SearchConfigModel handles directory search configuration
type SearchConfigModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// Configuration state
	currentOption  int             // 0=add path, 1=change depth, 2=continue
	inputMode      bool            // Whether we're in input mode
	pathInput      textinput.Model // For adding search paths
	depthInput     textinput.Model // For changing search depth
	searchPaths    []string        // Current search paths
	searchDepth    int             // Current search depth
	fullDiskSearch bool            // Full disk search toggle

	// Styles
	headerStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
	helpStyle     lipgloss.Style
}

// NewSearchConfigModel creates a new search configuration model
func NewSearchConfigModel(cfg core.Config, logger *core.Logger, shared *AppState) *SearchConfigModel {
	// Initialize default search paths (like the shell script)
	defaultPaths := []string{
		".",
		"./Development",
		"./Projects",
		"./dev",
	}

	model := &SearchConfigModel{
		cfg:            cfg,
		logger:         logger,
		shared:         shared,
		currentOption:  0,
		inputMode:      false,
		searchPaths:    defaultPaths,
		searchDepth:    3,
		fullDiskSearch: false,

		// Styles matching the existing configuration model
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

	// Setup input fields
	model.setupInputs()
	return model
}

// setupInputs creates text inputs for path and depth configuration
func (m *SearchConfigModel) setupInputs() {
	// Path input
	m.pathInput = textinput.New()
	m.pathInput.Placeholder = "e.g., /home/user/Development"
	m.pathInput.Width = 50

	// Depth input
	m.depthInput = textinput.New()
	m.depthInput.Placeholder = "3"
	m.depthInput.SetValue(fmt.Sprintf("%d", m.searchDepth))
	m.depthInput.Width = 20
}

// Init initializes the search configuration screen
func (m *SearchConfigModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for search configuration
func (m *SearchConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeys(msg)

	case tea.WindowSizeMsg:
		// Handle window resize - adjust input widths if needed
		maxWidth := msg.Width - 15
		if maxWidth < 20 {
			maxWidth = 20
		}
		m.pathInput.Width = maxWidth
		m.depthInput.Width = maxWidth / 2
		return m, nil

	default:
		// Update current input if in input mode
		if m.inputMode {
			var cmd tea.Cmd
			if m.currentOption == 0 {
				m.pathInput, cmd = m.pathInput.Update(msg)
			} else if m.currentOption == 1 {
				m.depthInput, cmd = m.depthInput.Update(msg)
			}
			return m, cmd
		}
	}

	return m, nil
}

// View renders the search configuration screen
func (m *SearchConfigModel) View() string {
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
		Render("⚙️ Configure Directory Search")

	b.WriteString(headerBox + "\n\n")

	// Current settings display (matching shell script output)
	b.WriteString("📂 Current Search Configuration:\n")
	pathsDisplay := strings.Join(m.searchPaths, " ")
	if len(pathsDisplay) > 50 {
		pathsDisplay = pathsDisplay[:47] + "..."
	}
	b.WriteString(fmt.Sprintf("  Paths: %s\n", pathsDisplay))
	b.WriteString(fmt.Sprintf("  Depth: %d levels\n", m.searchDepth))
	b.WriteString(fmt.Sprintf("  Full disk search: %s\n\n", map[bool]string{true: "enabled", false: "disabled"}[m.fullDiskSearch]))

	// Configuration options (matching shell script: 1. Add path  2. Change depth  3. Toggle full search  4. Continue)
	options := []string{
		"1. Add search path",
		"2. Change search depth",
		"3. Toggle full disk search",
		"4. Continue [DEFAULT]",
	}

	for i, option := range options {
		if i == m.currentOption {
			b.WriteString(m.selectedStyle.Render(option) + "\n")
		} else {
			b.WriteString(m.normalStyle.Render(option) + "\n")
		}
	}
	b.WriteString("\n")

	// Show input field if in input mode
	if m.inputMode {
		if m.currentOption == 0 {
			b.WriteString("New search path:\n")
			b.WriteString(m.pathInput.View() + "\n\n")
		} else if m.currentOption == 1 {
			b.WriteString("Search depth (number of directory levels):\n")
			b.WriteString(m.depthInput.View() + "\n\n")
		}
	}

	// Help text (matching shell script behavior)
	if m.inputMode {
		b.WriteString(m.helpStyle.Render("enter: save • esc: cancel • q: back to menu"))
	} else {
		b.WriteString(m.helpStyle.Render("j/k or ↑/↓: navigate • enter: select option • q: back to menu"))
	}

	return b.String()
}

// handleKeys handles keyboard input
func (m *SearchConfigModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.inputMode {
		return m.handleInputMode(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, TransitionToScreen(ScreenMainMenu)

	case "j", "down":
		m.currentOption++
		if m.currentOption >= 4 {
			m.currentOption = 0
		}
		return m, nil

	case "k", "up":
		m.currentOption--
		if m.currentOption < 0 {
			m.currentOption = 3
		}
		return m, nil

	case "enter":
		return m.selectCurrentOption()
	}

	return m, nil
}

// handleInputMode handles keyboard input when in input mode
func (m *SearchConfigModel) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Save the input
		if m.currentOption == 0 {
			// Add search path
			newPath := strings.TrimSpace(m.pathInput.Value())
			if newPath != "" {
				m.searchPaths = append(m.searchPaths, newPath)
				m.pathInput.SetValue("")
				m.logger.Info("search_config", fmt.Sprintf("Added search path: %s", newPath))
			}
		} else if m.currentOption == 1 {
			// Change search depth
			newDepthStr := strings.TrimSpace(m.depthInput.Value())
			if newDepthStr != "" {
				if newDepth := parseDepth(newDepthStr); newDepth > 0 {
					m.searchDepth = newDepth
					m.logger.Info("search_config", fmt.Sprintf("Changed search depth to: %d", newDepth))
				}
			}
		}

		// Exit input mode
		m.inputMode = false
		m.pathInput.Blur()
		m.depthInput.Blur()
		return m, nil

	case "esc", "q":
		// Cancel input
		m.inputMode = false
		m.pathInput.Blur()
		m.depthInput.Blur()
		return m, nil

	default:
		// Pass to current input
		var cmd tea.Cmd
		if m.currentOption == 0 {
			m.pathInput, cmd = m.pathInput.Update(msg)
		} else if m.currentOption == 1 {
			m.depthInput, cmd = m.depthInput.Update(msg)
		}
		return m, cmd
	}
}

// selectCurrentOption handles option selection
func (m *SearchConfigModel) selectCurrentOption() (tea.Model, tea.Cmd) {
	switch m.currentOption {
	case 0:
		// Option 1: Add search path
		m.inputMode = true
		m.pathInput.Focus()
		return m, nil

	case 1:
		// Option 2: Change search depth
		m.inputMode = true
		m.depthInput.Focus()
		return m, nil

	case 2:
		// Option 3: Toggle full disk search
		m.fullDiskSearch = !m.fullDiskSearch
		m.logger.Info("search_config", fmt.Sprintf("Toggled full disk search: %t", m.fullDiskSearch))
		return m, nil

	case 3:
		// Option 4: Continue - save settings and return to main menu
		m.saveSettings()
		return m, TransitionToScreen(ScreenMainMenu)
	}

	return m, nil
}

// saveSettings saves the search configuration settings
func (m *SearchConfigModel) saveSettings() {
	m.logger.Info("search_config", fmt.Sprintf("Saved search settings: paths=%v, depth=%d, fullSearch=%t",
		m.searchPaths, m.searchDepth, m.fullDiskSearch))

	// TODO: Store these settings in shared state for use by directory scanning
	// m.shared.SearchPaths = m.searchPaths
	// m.shared.SearchDepth = m.searchDepth
	// m.shared.FullDiskSearch = m.fullDiskSearch
}

// parseDepth parses a depth string to integer, returns 0 if invalid
func parseDepth(s string) int {
	// Simple validation - must be a positive number
	for _, char := range s {
		if char < '0' || char > '9' {
			return 0
		}
	}

	if s == "" {
		return 0
	}

	// Manual parsing to avoid importing strconv
	result := 0
	for _, char := range s {
		result = result*10 + int(char-'0')
	}

	if result > 10 { // Reasonable max depth
		return 10
	}

	return result
}
