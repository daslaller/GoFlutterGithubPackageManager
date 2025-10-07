// Package components/main_menu_view.go - Main Menu View Component
//
// This file implements the main menu view component following the bubbles
// view component pattern. It handles the main menu state, timeout behavior,
// and user interactions while maintaining shell script parity.

package components

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// MainMenuView represents the main menu view component
type MainMenuView struct {
	cfg            core.Config
	logger         *core.Logger
	choice         int
	selectedChoice int
	timeRemaining  int
	complete       bool

	// Styling
	headerStyle   lipgloss.Style
	checkboxStyle lipgloss.Style
	subtleStyle   lipgloss.Style
	timeoutStyle  lipgloss.Style
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
	return &MainMenuView{
		cfg:           cfg,
		logger:        logger,
		choice:        0,  // Start with first option selected
		timeRemaining: 60, // 60-second timeout (shell script behavior)

		// Initialize styles following bubbletea documentation
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).
			Bold(true),

		checkboxStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")),

		subtleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		timeoutStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("79")),
	}
}

// checkbox renders a checkbox like in the bubbletea documentation
func (v *MainMenuView) checkbox(label string, checked bool) string {
	if checked {
		return v.checkboxStyle.Render("[x] " + label)
	}
	return fmt.Sprintf("[ ] %s", label)
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
	return v.tickTimer()
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
	}

	return v, nil
}

// View renders this view component
func (v *MainMenuView) View() string {
	c := v.choice

	tpl := "What to do today?\n\n"
	tpl += "%s\n\n"
	tpl += "Program quits in %s seconds\n\n"
	tpl += v.subtleStyle.Render("j/k, up/down: select") + " • " +
		v.subtleStyle.Render("enter: choose") + " • " +
		v.subtleStyle.Render("q, esc: quit")

	choices := fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		v.checkbox("Scan directories", c == 0),
		v.checkbox("GitHub repo", c == 1),
		v.checkbox("Configure search", c == 2),
		v.checkbox("🔄 Check for Flutter-PM updates", c == 3),
	)

	return fmt.Sprintf(tpl, choices, v.timeoutStyle.Render(strconv.Itoa(v.timeRemaining)))
}

// handleKeys handles key input for the main menu
func (v *MainMenuView) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return v, tea.Quit

	case "j", "down":
		v.choice++
		if v.choice > 3 {
			v.choice = 3
		}

	case "k", "up":
		v.choice--
		if v.choice < 0 {
			v.choice = 0
		}

	case "enter":
		v.selectedChoice = v.choice + 1 // Convert to 1-based for shell script compatibility
		v.complete = true
		return v, nil

	case "1":
		v.selectedChoice = 1
		v.complete = true
		return v, nil

	case "2":
		v.selectedChoice = 2
		v.complete = true
		return v, nil

	case "3":
		v.selectedChoice = 3
		v.complete = true
		return v, nil

	case "4":
		v.selectedChoice = 4
		v.complete = true
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
