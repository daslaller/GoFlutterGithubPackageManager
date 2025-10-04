// Package components/main_menu_view.go - Main Menu View Component
//
// This file implements the main menu view component following the bubbles
// view component pattern. It handles the main menu state, timeout behavior,
// and user interactions while maintaining shell script parity.

package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// MainMenuView represents the main menu view component
type MainMenuView struct {
	cfg            core.Config
	logger         *core.Logger
	list           list.Model
	selectedChoice int
	timeRemaining  int
	complete       bool

	// Styling
	headerStyle  lipgloss.Style
	menuStyle    lipgloss.Style
	timeoutStyle lipgloss.Style
}

// MainMenuData represents data passed to the main menu
type MainMenuData struct {
	Project    *core.Project
	HasGitDeps bool
	CanUpdate  bool
}

// MainMenuResult represents the result from main menu selection
type MainMenuResult struct {
	Choice int
}

// NewMainMenuView creates a new main menu view component
func NewMainMenuView(cfg core.Config, logger *core.Logger) *MainMenuView {
	// Create menu items with shell script parity
	items := []list.Item{
		MenuItem{number: 1, title: "Scan directories", description: "Scan for Flutter projects in common directories"},
		MenuItem{number: 2, title: "GitHub repo", description: "Browse and select packages from GitHub repositories"},
		MenuItem{number: 3, title: "Configure search", description: "Set up search filters and preferences"},
		MenuItem{number: 6, title: "🔄 Check for Flutter-PM updates", description: "Update Flutter Package Manager to latest version"},
	}

	// Create list with styling
	l := list.New(items, NewMainMenuDelegate(), 0, 0)
	l.Title = "📱 Flutter Package Manager - Main Menu"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return &MainMenuView{
		cfg:           cfg,
		logger:        logger,
		list:          l,
		timeRemaining: 60, // 60 second timeout (shell script behavior)

		// Initialize styles
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")).
			Background(lipgloss.Color("#F3F4F6")).
			Bold(true).
			Width(60).
			Align(lipgloss.Center).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8B5CF6")),

		menuStyle: lipgloss.NewStyle().
			Padding(1, 2),

		timeoutStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true),
	}
}

// MenuItem represents a menu item
type MenuItem struct {
	number      int
	title       string
	description string
}

func (i MenuItem) Title() string       { return fmt.Sprintf("%d. %s", i.number, i.title) }
func (i MenuItem) Description() string { return i.description }
func (i MenuItem) FilterValue() string { return i.title }

// NewMainMenuDelegate creates a delegate for main menu items
func NewMainMenuDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#8B5CF6")).
		Padding(0, 1).
		Bold(true)

	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Background(lipgloss.Color("#8B5CF6"))

	return d
}

// SetData sets the data for this view component
func (v *MainMenuView) SetData(data interface{}) {
	if menuData, ok := data.(MainMenuData); ok {
		// Update menu items based on project state
		v.updateMenuItems(menuData)
	}
}

// GetResult returns the result from this view component
func (v *MainMenuView) GetResult() interface{} {
	return MainMenuResult{Choice: v.selectedChoice}
}

// IsComplete returns whether this view component has completed
func (v *MainMenuView) IsComplete() bool {
	return v.complete
}

// Init initializes the view component
func (v *MainMenuView) Init() tea.Cmd {
	return tea.Batch(
		v.list.StartSpinner(),
		v.tickTimer(),
	)
}

// Update handles messages for this view component
func (v *MainMenuView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return v.handleKeys(msg)

	case timerTickMsg:
		v.timeRemaining--
		if v.timeRemaining <= 0 {
			// Auto-select default choice (shell script behavior)
			v.selectedChoice = 1 // Default to scan directories
			v.complete = true
			return v, nil
		}
		return v, v.tickTimer()

	default:
		var cmd tea.Cmd
		v.list, cmd = v.list.Update(msg)
		return v, cmd
	}
}

// View renders this view component
func (v *MainMenuView) View() string {
	var b strings.Builder

	// Header
	b.WriteString(v.headerStyle.Render("📱 Flutter Package Manager") + "\n\n")

	// Menu content
	b.WriteString(v.menuStyle.Render(v.list.View()) + "\n\n")

	// Timeout information (shell script behavior)
	timeoutText := fmt.Sprintf("Choice (1-6, default: 1, auto in %ds)", v.timeRemaining)
	b.WriteString(v.timeoutStyle.Render(timeoutText) + "\n\n")

	// Help text
	helpText := "1-6 select option • enter default • q quit"
	b.WriteString(v.timeoutStyle.Render(helpText))

	return b.String()
}

// handleKeys handles key input for the main menu
func (v *MainMenuView) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return v, tea.Quit

	case "1", "2", "3", "6":
		// Direct number selection (shell script behavior)
		choice := 0
		switch msg.String() {
		case "1":
			choice = 1
		case "2":
			choice = 2
		case "3":
			choice = 3
		case "6":
			choice = 6
		}
		v.selectedChoice = choice
		v.complete = true
		return v, nil

	case "enter":
		// Select current item or default
		if v.list.SelectedItem() != nil {
			if item, ok := v.list.SelectedItem().(MenuItem); ok {
				v.selectedChoice = item.number
			}
		} else {
			v.selectedChoice = 1 // Default choice
		}
		v.complete = true
		return v, nil

	case "up", "k":
		v.list.CursorUp()
		return v, nil

	case "down", "j":
		v.list.CursorDown()
		return v, nil
	}

	return v, nil
}

// updateMenuItems updates menu items based on project state
func (v *MainMenuView) updateMenuItems(data MainMenuData) {
	// Update menu items to reflect current project state
	// This could include showing/hiding options based on project detection
}

// tickTimer returns a command for the countdown timer
func (v *MainMenuView) tickTimer() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return timerTickMsg{}
	})
}

// timerTickMsg represents a timer tick message
type timerTickMsg struct{}
