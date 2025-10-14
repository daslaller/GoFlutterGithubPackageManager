// Package models/main_menu_model.go - Main Menu Screen Model
//
// This file implements the main menu screen model using the checkbox style
// from the bubbletea documentation. It handles menu selection, timeout behavior,
// and transitions to other screens.

package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// MainMenuModel handles the main menu screen
type MainMenuModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// Menu state
	choice      int // 0-based cursor position
	menuTimeout int // seconds remaining
	quitting    bool

	// Styles (bubbletea documentation colors)
	subtleStyle   lipgloss.Style
	ticksStyle    lipgloss.Style
	checkboxStyle lipgloss.Style
	headerStyle   lipgloss.Style

	// Performance optimization: pre-allocated render buffer
	renderBuffer strings.Builder
	menuLines    []string // Pre-allocated slice for menu lines
}

// Menu options
type MenuOption struct {
	title       string
	description string
	action      AppScreen
}

// getMenuOptions returns the menu options, conditionally including local project option
func (m *MainMenuModel) getMenuOptions() []MenuOption {
	var options []MenuOption

	// Option 1: Add packages to local project (if detected)
	if m.shared.LocalPubspecAvailable {
		options = append(options, MenuOption{
			fmt.Sprintf("📦 Add package to local - (%s)", m.shared.DetectedProject),
			fmt.Sprintf("Add Git packages to local project: %s", m.shared.DetectedProject),
			ScreenDependencySelection, // Will add packages to detected project
		})
	}

	// Option 2 (or 1 if no local): Check prerequisites
	options = append(options, MenuOption{
		"🔧 Check prerequisites",
		"Verify required tools (git, dart/flutter, gh) are installed",
		ScreenPrerequisites,
	})

	// Option 3 (or 2): GitHub repo
	options = append(options, MenuOption{
		"🐙 GitHub repo",
		"Browse and select packages from GitHub repositories",
		ScreenGitHubRepo,
	})

	// Option 4 (or 3): Configure search
	options = append(options, MenuOption{
		"⚙️ Configure search",
		"Set up search filters and preferences",
		ScreenSearchConfig,
	})

	// Option 5 (or 4): Update local package - show project name or greyed out
	var updateTitle, updateDesc string
	if m.shared.LocalPubspecAvailable {
		updateTitle = fmt.Sprintf("📁 Flutter package update - (%s)", m.shared.DetectedProject)
		updateDesc = fmt.Sprintf("Update stale packages in %s", m.shared.DetectedProject)
	} else {
		updateTitle = "📁 Flutter package update - (none found)"
		updateDesc = "No local Flutter project detected within +-3 levels"
	}
	options = append(options, MenuOption{
		updateTitle,
		updateDesc,
		ScreenScanDirectories,
	})

	// Option 6 (or 5): Self-update
	options = append(options, MenuOption{
		"🔄 Check for Flutter-PM updates",
		"Update Flutter Package Manager to latest version",
		ScreenSelfUpdate,
	})

	return options
}

// timerTickMsg represents a timer tick
type timerTickMsg struct{}

// NewMainMenuModel creates a new main menu model
func NewMainMenuModel(cfg core.Config, logger *core.Logger, shared *AppState) *MainMenuModel {
	model := &MainMenuModel{
		cfg:         cfg,
		logger:      logger,
		shared:      shared,
		choice:      0,
		menuTimeout: 60, // 60-second timeout like shell script

		// Styles matching bubbletea documentation
		subtleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		ticksStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("79")),

		checkboxStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")),

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).
			Bold(true),

		// Pre-allocate render buffer and menu lines
		menuLines: make([]string, 0, 20), // Capacity for typical menu size
	}

	// Pre-size the string builder for typical content
	model.renderBuffer.Grow(1024)

	return model
}

// Init initializes the main menu
func (m *MainMenuModel) Init() tea.Cmd {
	return m.tickTimer()
}

// Update handles messages for the main menu
func (m *MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeys(msg)

	case timerTickMsg:
		m.menuTimeout--
		if m.menuTimeout <= 0 {
			// Auto-select default choice (shell script behavior)
			m.shared.ProjectSourceChoice = 1 // Default to scan directories
			return m, TransitionToScreen(ScreenScanDirectories)
		}
		return m, m.tickTimer()

	case tea.WindowSizeMsg:
		// Handle window resize if needed
		return m, nil
	}

	return m, nil
}

// View renders the main menu with beautiful bordered styling (optimized)
func (m *MainMenuModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Reset the pre-allocated builder instead of creating new one
	m.renderBuffer.Reset()

	// Reset menu lines slice (keep capacity)
	m.menuLines = m.menuLines[:0]

	c := m.choice

	// Beautiful bordered header with warm amber color (consistent with source selection)
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F59E0B")).
		Foreground(lipgloss.Color("#F59E0B")).
		Padding(1, 2).
		Align(lipgloss.Center).
		Width(62).
		Bold(true).
		Render("🎯 Flutter Package Manager")

	// Build content using pre-allocated slice
	m.menuLines = append(m.menuLines, headerBox)
	m.menuLines = append(m.menuLines, "")
	m.menuLines = append(m.menuLines, "📱 Flutter Package Manager - Main Menu:")

	// Get dynamic menu options
	options := m.getMenuOptions()

	// Menu options with optimized string building
	// Determine which option is the "update" option for greying out
	updateOptionIndex := -1
	if m.shared.LocalPubspecAvailable {
		updateOptionIndex = 4 // Option 5 when local project exists
	} else {
		updateOptionIndex = 3 // Option 4 when no local project
	}

	for i, option := range options {
		var line string
		// Check if this is the update option and no project found
		isDisabled := (i == updateOptionIndex && !m.shared.LocalPubspecAvailable)

		if c == i {
			line = "► " + strconv.Itoa(i+1) + ". " + option.title
			if isDisabled {
				line = m.subtleStyle.Render(line) // Grey out disabled option
			} else {
				line = m.checkboxStyle.Render(line)
			}
		} else {
			line = "  " + strconv.Itoa(i+1) + ". " + option.title
			if isDisabled {
				line = m.subtleStyle.Render(line) // Grey out disabled option
			}
		}
		m.menuLines = append(m.menuLines, line)
	}

	m.menuLines = append(m.menuLines, "")

	// Timeout info with pre-computed string
	timeoutText := "Program quits in " + m.ticksStyle.Render(strconv.Itoa(m.menuTimeout)) + " seconds"
	m.menuLines = append(m.menuLines, timeoutText)
	m.menuLines = append(m.menuLines, "")

	// Help text in beautiful style
	helpText := "↑/↓ navigate • enter/1-5 select • q quit"
	m.menuLines = append(m.menuLines, m.subtleStyle.Render(helpText))

	// Join all lines efficiently using pre-allocated builder
	for i, line := range m.menuLines {
		if i > 0 {
			m.renderBuffer.WriteByte('\n')
		}
		m.renderBuffer.WriteString(line)
	}

	return m.renderBuffer.String()
}

// handleKeys handles keyboard input
func (m *MainMenuModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	options := m.getMenuOptions()

	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit

	case "j", "down":
		m.choice++
		if m.choice >= len(options) {
			m.choice = len(options) - 1
		}
		return m, nil

	case "k", "up":
		m.choice--
		if m.choice < 0 {
			m.choice = 0
		}
		return m, nil

	case "enter":
		return m.selectCurrentChoice()

	case "1", "2", "3", "4", "5", "6":
		// Handle number selection dynamically
		num := int(msg.String()[0] - '0')
		if num > 0 && num <= len(options) {
			m.choice = num - 1
			return m.selectCurrentChoice()
		}
	}

	return m, nil
}

// selectCurrentChoice handles selection of the current menu item
func (m *MainMenuModel) selectCurrentChoice() (tea.Model, tea.Cmd) {
	options := m.getMenuOptions()

	if m.choice >= 0 && m.choice < len(options) {
		selectedOption := options[m.choice]
		m.shared.ProjectSourceChoice = m.choice + 1 // Convert to 1-based for shell script compatibility

		// Log the selection
		m.logger.Info("menu", fmt.Sprintf("Selected: %s", selectedOption.title))

		// Transition to the appropriate screen
		return m, TransitionToScreen(selectedOption.action)
	}

	return m, nil
}

// tickTimer returns a command for the countdown timer
func (m *MainMenuModel) tickTimer() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return timerTickMsg{}
	})
}
