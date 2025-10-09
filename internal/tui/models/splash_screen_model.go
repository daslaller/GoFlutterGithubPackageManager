// Package models/splash_screen_model.go - Splash Screen with Prerequisites Check
//
// This file implements the initial splash screen that displays while checking
// and auto-installing prerequisites. It provides a smooth startup experience.

package models

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// SplashScreenModel handles the splash screen with prerequisites checking
type SplashScreenModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// Check state
	checking         bool
	checkComplete    bool
	checkResult      core.PrerequisiteCheck
	progressDots     int
	frame            int
	autoTransition   bool
	transitionDelay  int // seconds before auto-transition
	showDetailedView bool

	// Styles
	titleStyle   lipgloss.Style
	subtitleStyle lipgloss.Style
	statusStyle  lipgloss.Style
	successStyle lipgloss.Style
	warningStyle lipgloss.Style
	errorStyle   lipgloss.Style
}

// prerequisitesCheckMsg is sent when prerequisites check completes
type prerequisitesCheckMsg struct {
	result core.PrerequisiteCheck
}

// animationTickMsg is sent for animation updates
type animationTickMsg struct{}

// NewSplashScreenModel creates a new splash screen model
func NewSplashScreenModel(cfg core.Config, logger *core.Logger, shared *AppState) *SplashScreenModel {
	return &SplashScreenModel{
		cfg:              cfg,
		logger:           logger,
		shared:           shared,
		checking:         true,
		checkComplete:    false,
		autoTransition:   true,
		transitionDelay:  2, // 2 seconds after check completes
		showDetailedView: false,

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0EA5E9")).
			Bold(true),

		subtitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		statusStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("79")),

		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")),

		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
	}
}

// Init initializes the splash screen
func (m *SplashScreenModel) Init() tea.Cmd {
	return tea.Batch(
		m.checkPrerequisites(),
		m.tickAnimation(),
	)
}

// Update handles messages for the splash screen
func (m *SplashScreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "d":
			// Toggle detailed view
			m.showDetailedView = !m.showDetailedView
			return m, nil

		case "enter", " ":
			if m.checkComplete {
				// Manual transition to main menu
				return m, TransitionToScreen(ScreenMainMenu)
			}
			return m, nil
		}

	case prerequisitesCheckMsg:
		m.checking = false
		m.checkComplete = true
		m.checkResult = msg.result

		// Log results
		if msg.result.AllMet {
			m.logger.Info("splash", "All prerequisites met")
		} else {
			m.logger.Info("splash", fmt.Sprintf("Missing prerequisites: %v", msg.result.Missing))
		}

		// Auto-transition to main menu after delay
		if m.autoTransition {
			return m, tea.Tick(time.Duration(m.transitionDelay)*time.Second, func(time.Time) tea.Msg {
				return TransitionToScreen(ScreenMainMenu)()
			})
		}

		return m, nil

	case animationTickMsg:
		m.frame++
		m.progressDots = (m.progressDots + 1) % 4
		if m.checking {
			return m, m.tickAnimation()
		}
		return m, nil

	case ScreenTransitionMsg:
		// Forward transition message
		return m, func() tea.Msg { return msg }
	}

	return m, nil
}

// View renders the splash screen
func (m *SplashScreenModel) View() string {
	var b strings.Builder

	// Beautiful bordered logo
	logo := []string{
		"███████╗██╗     ██╗   ██╗████████╗████████╗███████╗██████╗ ",
		"██╔════╝██║     ██║   ██║╚══██╔══╝╚══██╔══╝██╔════╝██╔══██╗",
		"█████╗  ██║     ██║   ██║   ██║      ██║   █████╗  ██████╔╝",
		"██╔══╝  ██║     ██║   ██║   ██║      ██║   ██╔══╝  ██╔══██╗",
		"██║     ███████╗╚██████╔╝   ██║      ██║   ███████╗██║  ██║",
		"╚═╝     ╚══════╝ ╚═════╝    ╚═╝      ╚═╝   ╚══════╝╚═╝  ╚═╝",
		"                                                             ",
		"           ██████╗  █████╗  ██████╗██╗  ██╗ █████╗  ██████╗ ███████╗",
		"           ██╔══██╗██╔══██╗██╔════╝██║ ██╔╝██╔══██╗██╔════╝ ██╔════╝",
		"           ██████╔╝███████║██║     █████╔╝ ███████║██║  ███╗█████╗  ",
		"           ██╔═══╝ ██╔══██║██║     ██╔═██╗ ██╔══██║██║   ██║██╔══╝  ",
		"           ██║     ██║  ██║╚██████╗██║  ██╗██║  ██║╚██████╔╝███████╗",
		"           ╚═╝     ╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝",
		"                                                                      ",
		"                    ███╗   ███╗ █████╗ ███╗   ██╗ █████╗  ██████╗ ███████╗██████╗ ",
		"                    ████╗ ████║██╔══██╗████╗  ██║██╔══██╗██╔════╝ ██╔════╝██╔══██╗",
		"                    ██╔████╔██║███████║██╔██╗ ██║███████║██║  ███╗█████╗  ██████╔╝",
		"                    ██║╚██╔╝██║██╔══██║██║╚██╗██║██╔══██║██║   ██║██╔══╝  ██╔══██╗",
		"                    ██║ ╚═╝ ██║██║  ██║██║ ╚████║██║  ██║╚██████╔╝███████╗██║  ██║",
		"                    ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝",
	}

	logoBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#0EA5E9")).
		Padding(1, 2).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("#0EA5E9"))

	// Render logo
	b.WriteString(logoBox.Render(strings.Join(logo, "\n")))
	b.WriteString("\n\n")

	// Status message
	if m.checking {
		dots := strings.Repeat(".", m.progressDots)
		spaces := strings.Repeat(" ", 3-m.progressDots)
		statusMsg := fmt.Sprintf("🔍 Checking prerequisites%s%s", dots, spaces)
		b.WriteString(m.statusStyle.Render(statusMsg))
		b.WriteString("\n\n")
	} else if m.checkComplete {
		if m.checkResult.AllMet {
			b.WriteString(m.successStyle.Render("✅ All prerequisites available!"))
			b.WriteString("\n\n")
		} else {
			b.WriteString(m.errorStyle.Render("❌ Some prerequisites are missing"))
			b.WriteString("\n\n")

			if m.showDetailedView {
				// Show detailed prerequisite results
				b.WriteString(m.renderDetailedResults())
			} else {
				// Show summary
				if len(m.checkResult.Missing) > 0 {
					b.WriteString(m.warningStyle.Render(fmt.Sprintf("Missing: %s", strings.Join(m.checkResult.Missing, ", "))))
					b.WriteString("\n")
				}
				if len(m.checkResult.Warnings) > 0 {
					b.WriteString(m.subtitleStyle.Render(fmt.Sprintf("Warnings: %d", len(m.checkResult.Warnings))))
					b.WriteString("\n")
				}
				b.WriteString("\n")
				b.WriteString(m.subtitleStyle.Render("Press 'd' for detailed view"))
				b.WriteString("\n")
			}
		}

		// Auto-transition message
		b.WriteString(m.subtitleStyle.Render("Starting in a moment... (press enter to continue now)"))
		b.WriteString("\n")
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(m.subtitleStyle.Render("q: quit"))

	return b.String()
}

// renderDetailedResults renders detailed prerequisite check results
func (m *SplashScreenModel) renderDetailedResults() string {
	var b strings.Builder

	b.WriteString("Prerequisites Status:\n\n")

	for _, result := range m.checkResult.Results {
		if result.Available {
			b.WriteString(m.successStyle.Render("✅ " + result.Name))
			b.WriteString(" - ")
			b.WriteString(m.subtitleStyle.Render(result.Version))
			b.WriteString("\n")
		} else {
			b.WriteString(m.errorStyle.Render("❌ " + result.Name))
			b.WriteString("\n")
			if result.InstallCmd != "" {
				b.WriteString("   ")
				b.WriteString(m.subtitleStyle.Render("Install: " + result.InstallCmd))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")

	if len(m.checkResult.Missing) > 0 {
		guidance := core.GetInstallationGuidance(m.checkResult)
		for _, line := range guidance {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// checkPrerequisites runs the prerequisites check in the background
func (m *SplashScreenModel) checkPrerequisites() tea.Cmd {
	return func() tea.Msg {
		result := core.CheckPrerequisites(m.logger)
		return prerequisitesCheckMsg{result: result}
	}
}

// tickAnimation returns a command for animation updates
func (m *SplashScreenModel) tickAnimation() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return animationTickMsg{}
	})
}
