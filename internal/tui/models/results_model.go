// Package models/results_model.go - Results Screen Model

package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ResultsModel handles displaying operation results
type ResultsModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// UI components
	viewport viewport.Model
	ready    bool

	// Styles
	headerStyle  lipgloss.Style
	successStyle lipgloss.Style
	errorStyle   lipgloss.Style
	warningStyle lipgloss.Style
	normalStyle  lipgloss.Style
	codeStyle    lipgloss.Style
}

// NewResultsModel creates a new results model
func NewResultsModel(cfg core.Config, logger *core.Logger, shared *AppState) *ResultsModel {
	vp := viewport.New(78, 20)

	return &ResultsModel{
		cfg:      cfg,
		logger:   logger,
		shared:   shared,
		viewport: vp,

		// Styles
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).
			Bold(true),

		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("202")).
			Bold(true),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		codeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Background(lipgloss.Color("236")).
			Padding(0, 1),
	}
}

// Init initializes the results screen
func (m *ResultsModel) Init() tea.Cmd {
	m.ready = true
	m.updateContent()
	return nil
}

// Update handles messages for results
func (m *ResultsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "enter":
			return m, TransitionToScreen(ScreenMainMenu)

		case "up", "k":
			m.viewport.LineUp(1)
			return m, nil

		case "down", "j":
			m.viewport.LineDown(1)
			return m, nil

		case "pgup":
			m.viewport.HalfViewUp()
			return m, nil

		case "pgdown":
			m.viewport.HalfViewDown()
			return m, nil

		case "home":
			m.viewport.GotoTop()
			return m, nil

		case "end":
			m.viewport.GotoBottom()
			return m, nil
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-8)
			m.ready = true
			m.updateContent()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 8
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the results screen
func (m *ResultsModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.headerStyle.Render("📊 Installation Results") + "\n\n")

	if !m.ready {
		return b.String() + "Preparing results..."
	}

	// Viewport content
	b.WriteString(m.viewport.View() + "\n")

	// Footer
	footerText := "↑/↓ scroll • pgup/pgdown page • home/end • enter/q: back to menu"
	b.WriteString(m.normalStyle.Render(footerText))

	return b.String()
}

// updateContent populates the viewport with results
func (m *ResultsModel) updateContent() {
	var content strings.Builder

	if len(m.shared.Results) == 0 {
		// No results (e.g., when coming from update check)
		content.WriteString(m.warningStyle.Render("ℹ️  No Installation Results") + "\n\n")
		content.WriteString("This could be because:\n")
		content.WriteString("• No packages were installed\n")
		content.WriteString("• You accessed results from the update menu\n")
		content.WriteString("• An error occurred before installation\n\n")

		content.WriteString(m.headerStyle.Render("Next Steps:") + "\n")
		content.WriteString("1. Return to main menu\n")
		content.WriteString("2. Select 'GitHub repo' to browse packages\n")
		content.WriteString("3. Configure and install packages\n\n")

		m.viewport.SetContent(content.String())
		return
	}

	// Results summary with conflict resolution tracking
	successCount := 0
	errorCount := 0
	conflictCount := 0
	for _, result := range m.shared.Results {
		if result.OK {
			successCount++
			// Check if this was a conflict resolution
			if result.Data != nil {
				if conflictResolved, ok := result.Data["conflict_resolved"].(bool); ok && conflictResolved {
					conflictCount++
				}
			}
		} else {
			errorCount++
		}
	}

	if errorCount == 0 {
		if conflictCount > 0 {
			content.WriteString(m.successStyle.Render("✅ All Packages Installed Successfully!") + "\n")
			content.WriteString(m.warningStyle.Render(fmt.Sprintf("🔧 %d conflict(s) automatically resolved", conflictCount)) + "\n\n")
		} else {
			content.WriteString(m.successStyle.Render("✅ All Packages Installed Successfully!") + "\n\n")
		}
	} else {
		if conflictCount > 0 {
			content.WriteString(m.errorStyle.Render(fmt.Sprintf("⚠️  %d Errors, %d Successful (%d conflicts resolved)", errorCount, successCount, conflictCount)) + "\n\n")
		} else {
			content.WriteString(m.errorStyle.Render(fmt.Sprintf("⚠️  %d Errors, %d Successful", errorCount, successCount)) + "\n\n")
		}
	}

	content.WriteString(fmt.Sprintf("Total packages processed: %d\n", len(m.shared.Results)))
	if conflictCount > 0 {
		content.WriteString(fmt.Sprintf("Dependency conflicts resolved: %d\n", conflictCount))
	}
	content.WriteString("\n")

	// Detailed results
	content.WriteString(m.headerStyle.Render("Detailed Results:") + "\n\n")

	for i, result := range m.shared.Results {
		// Package header
		if result.OK {
			content.WriteString(m.successStyle.Render(fmt.Sprintf("✅ Package %d: SUCCESS", i+1)) + "\n")
		} else {
			content.WriteString(m.errorStyle.Render(fmt.Sprintf("❌ Package %d: FAILED", i+1)) + "\n")
		}

		// Message
		content.WriteString(fmt.Sprintf("   %s\n", result.Message))

		// Error details
		if result.Err != "" {
			content.WriteString(m.errorStyle.Render(fmt.Sprintf("   Error: %s", result.Err)) + "\n")
		}

		// Package data and conflict resolution details
		if result.Data != nil {
			if pkg, ok := result.Data["package"].(string); ok {
				content.WriteString(fmt.Sprintf("   Package: %s\n", pkg))
			}
			if url, ok := result.Data["url"].(string); ok {
				content.WriteString(fmt.Sprintf("   URL: %s\n", url))
			}
			if ref, ok := result.Data["ref"].(string); ok {
				content.WriteString(fmt.Sprintf("   Ref: %s\n", ref))
			}

			// Show conflict resolution details if applicable
			if conflictResolved, ok := result.Data["conflict_resolved"].(bool); ok && conflictResolved {
				content.WriteString(m.warningStyle.Render("   🔧 Conflict Resolution Applied:") + "\n")

				if conflictType, ok := result.Data["conflict_type"].(string); ok {
					content.WriteString(fmt.Sprintf("   • Conflict Type: %s\n", conflictType))
				}

				if conflictingPkg, ok := result.Data["conflicting_pkg"].(string); ok && conflictingPkg != "" {
					content.WriteString(fmt.Sprintf("   • Conflicting Package: %s\n", conflictingPkg))
				}

				if resolutionMethod, ok := result.Data["resolution_method"].(string); ok {
					switch resolutionMethod {
					case "inline_dependency_override":
						content.WriteString("   • Resolution: Inline dependency override\n")
						content.WriteString("   • Method: Used dart pub add with override syntax\n")
					default:
						content.WriteString(fmt.Sprintf("   • Resolution Method: %s\n", resolutionMethod))
					}
				}

				if userMessage, ok := result.Data["user_message"].(string); ok {
					content.WriteString(m.successStyle.Render(fmt.Sprintf("   • Result: %s", userMessage)) + "\n")
				}
			}
		}

		// Logs
		if len(result.Logs) > 0 {
			content.WriteString("   Logs:\n")
			for _, log := range result.Logs {
				// Don't render with codeStyle to avoid width constraints
				// Just indent and display the full log
				content.WriteString(fmt.Sprintf("      %s\n", log))
			}
		}

		content.WriteString("\n")
	}

	// Next steps
	content.WriteString(m.headerStyle.Render("Next Steps:") + "\n")
	if errorCount == 0 {
		content.WriteString("🎉 All packages were installed successfully!\n\n")
		content.WriteString("You can now:\n")
		content.WriteString("• Import the packages in your Dart/Flutter code\n")
		content.WriteString("• Run your project to test the new packages\n")
		content.WriteString("• Add more packages by returning to the main menu\n\n")
	} else {
		content.WriteString("❌ Some packages failed to install.\n\n")
		content.WriteString("Please:\n")
		content.WriteString("• Review the error messages above\n")
		content.WriteString("• Check your internet connection\n")
		content.WriteString("• Verify package URLs and refs\n")
		content.WriteString("• Try installing failed packages again\n\n")
	}

	content.WriteString("Press Enter or Q to return to the main menu")

	m.viewport.SetContent(content.String())
}
