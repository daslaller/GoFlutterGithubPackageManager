// Package models/main_menu_model.go - Main Menu Screen Model
//
// This file implements the main menu screen model using the checkbox style
// from the bubbletea documentation. It handles menu selection, timeout behavior,
// and transitions to other screens.

package models

import (
	"fmt"
	"strconv"
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
}

// Menu options
type MenuOption struct {
	title       string
	description string
	action      AppScreen
}

var menuOptions = []MenuOption{
	{"Scan directories", "Scan for Flutter projects in common directories", ScreenScanDirectories},
	{"GitHub repo", "Browse and select packages from GitHub repositories", ScreenGitHubRepo},
	{"Configure search", "Set up search filters and preferences", ScreenConfiguration},
	{"🔄 Check for Flutter-PM updates", "Update Flutter Package Manager to latest version", ScreenResults}, // Show update results
}

// timerTickMsg represents a timer tick
type timerTickMsg struct{}

// NewMainMenuModel creates a new main menu model
func NewMainMenuModel(cfg core.Config, logger *core.Logger, shared *AppState) *MainMenuModel {
	return &MainMenuModel{
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
	}
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

// View renders the main menu
func (m *MainMenuModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	c := m.choice

	tpl := m.headerStyle.Render("📱 Flutter Package Manager") + "\n\n"
	tpl += "What to do today?\n\n"
	tpl += "%s\n\n"
	tpl += "Program quits in %s seconds\n\n"
	tpl += m.subtleStyle.Render("j/k, up/down: select") + " • " +
		m.subtleStyle.Render("enter: choose") + " • " +
		m.subtleStyle.Render("1-4: direct select") + " • " +
		m.subtleStyle.Render("q, esc: quit")

	choices := ""
	for i, option := range menuOptions {
		choices += m.checkbox(option.title, c == i) + "\n"
	}

	return fmt.Sprintf(tpl, choices, m.ticksStyle.Render(strconv.Itoa(m.menuTimeout)))
}

// handleKeys handles keyboard input
func (m *MainMenuModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit

	case "j", "down":
		m.choice++
		if m.choice >= len(menuOptions) {
			m.choice = len(menuOptions) - 1
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

	case "1":
		m.choice = 0
		return m.selectCurrentChoice()

	case "2":
		m.choice = 1
		return m.selectCurrentChoice()

	case "3":
		m.choice = 2
		return m.selectCurrentChoice()

	case "4":
		m.choice = 3
		return m.selectCurrentChoice()
	}

	return m, nil
}

// selectCurrentChoice handles selection of the current menu item
func (m *MainMenuModel) selectCurrentChoice() (tea.Model, tea.Cmd) {
	if m.choice >= 0 && m.choice < len(menuOptions) {
		selectedOption := menuOptions[m.choice]
		m.shared.ProjectSourceChoice = m.choice + 1 // Convert to 1-based for shell script compatibility

		// Log the selection
		m.logger.Info("menu", fmt.Sprintf("Selected: %s", selectedOption.title))

		// Transition to the appropriate screen
		return m, TransitionToScreen(selectedOption.action)
	}

	return m, nil
}

// checkbox renders a checkbox like in the bubbletea documentation
func (m *MainMenuModel) checkbox(label string, checked bool) string {
	if checked {
		return m.checkboxStyle.Render("[x] " + label)
	}
	return fmt.Sprintf("[ ] %s", label)
}

// tickTimer returns a command for the countdown timer
func (m *MainMenuModel) tickTimer() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return timerTickMsg{}
	})
}
