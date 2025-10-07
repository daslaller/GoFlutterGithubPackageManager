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

	// Beautiful bordered header like the README (cached style)
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#0EA5E9")).
		Padding(1, 2).
		Align(lipgloss.Center).
		Width(62).
		Render("🎯 Flutter Package Manager")

	// Build content using pre-allocated slice
	m.menuLines = append(m.menuLines, headerBox)
	m.menuLines = append(m.menuLines, "")
	m.menuLines = append(m.menuLines, "📱 Flutter Package Manager - Main Menu:")

	// Pre-compute emoji array to avoid switch in hot path
	emojis := [4]string{"📁", "🐙", "⚙️", "🔄"}

	// Menu options with optimized string building
	for i, option := range menuOptions {
		var line string
		if c == i {
			line = "► " + strconv.Itoa(i+1) + ". " + emojis[i] + " " + option.title
			line = m.checkboxStyle.Render(line)
		} else {
			line = "  " + strconv.Itoa(i+1) + ". " + emojis[i] + " " + option.title
		}
		m.menuLines = append(m.menuLines, line)
	}

	m.menuLines = append(m.menuLines, "")

	// Detected project info (placeholder for now)
	if m.shared.LocalPubspecAvailable {
		detectedText := "💡 Detected Flutter project: " + m.shared.DetectedProject
		m.menuLines = append(m.menuLines, m.subtleStyle.Render(detectedText))
		m.menuLines = append(m.menuLines, "")
	}

	// Timeout info with pre-computed string
	timeoutText := "Program quits in " + m.ticksStyle.Render(strconv.Itoa(m.menuTimeout)) + " seconds"
	m.menuLines = append(m.menuLines, timeoutText)
	m.menuLines = append(m.menuLines, "")

	// Help text in beautiful style
	helpText := "↑/↓ navigate • enter/1-4 select • q quit"
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
