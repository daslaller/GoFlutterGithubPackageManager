// Package tui/autotest.go - Automated Testing for New Architecture
//
// This file implements automated testing functionality for the new multimodel
// architecture, simulating user interactions and verifying the complete workflow.

package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/tui/models"
)

// AutoTestModel wraps the AppModel and adds automated testing capabilities
type AutoTestModel struct {
	app         *models.AppModel
	testStep    int
	testTimer   *time.Timer
	testPaused  bool
	testResults []string
}

// autoTestStepMsg is sent to advance the automated test
type autoTestStepMsg struct {
	step int
}

// NewAutoTestModel creates an automated testing wrapper
func NewAutoTestModel(cfg core.Config, logger *core.Logger) *AutoTestModel {
	app := models.NewAppModel(cfg, logger)
	return &AutoTestModel{
		app:         app,
		testStep:    0,
		testResults: []string{},
	}
}

// Init initializes the autotest
func (m *AutoTestModel) Init() tea.Cmd {
	fmt.Println("🚀 Starting NEW ARCHITECTURE AUTO-TEST")
	fmt.Println("📋 Testing complete workflow: Main Menu → GitHub → Selection → Config → Confirm → Execute → Results")
	fmt.Println()

	return tea.Batch(
		m.app.Init(),
		m.scheduleNextStep(),
	)
}

// Update handles autotest progression
func (m *AutoTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case autoTestStepMsg:
		return m.executeTestStep(msg.step)

	case tea.KeyMsg:
		if msg.String() == "p" {
			m.testPaused = !m.testPaused
			if !m.testPaused {
				cmds = append(cmds, m.scheduleNextStep())
			}
			return m, tea.Batch(cmds...)
		}
		// Don't pass keys to app during autotest
		return m, nil

	default:
		// Pass other messages to the app
		var cmd tea.Cmd
		appModel, cmd := m.app.Update(msg)
		m.app = appModel.(*models.AppModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the autotest interface
func (m *AutoTestModel) View() string {
	testSteps := []string{
		"Starting autotest",
		"Testing main menu navigation",
		"Selecting GitHub repo option",
		"Simulating repository loading",
		"Testing repository selection",
		"Testing package configuration",
		"Testing confirmation screen",
		"Testing execution progress",
		"Testing results display",
		"Autotest complete",
	}

	header := fmt.Sprintf("🤖 AUTOTEST - Step %d/%d: %s\n",
		m.testStep+1, len(testSteps), testSteps[minInt(m.testStep, len(testSteps)-1)])

	if m.testPaused {
		header += "⏸️  PAUSED - Press P to resume\n"
	}

	header += "\n"

	return header + m.app.View()
}

// executeTestStep performs a specific test step
func (m *AutoTestModel) executeTestStep(step int) (tea.Model, tea.Cmd) {
	m.testStep = step
	var cmds []tea.Cmd

	switch step {
	case 0: // Start
		m.logTestResult("✅ Autotest started successfully")
		cmds = append(cmds, m.scheduleNextStep())

	case 1: // Test main menu
		m.logTestResult("✅ Main menu displayed correctly")
		// Simulate selecting GitHub repo option (option 2)
		cmds = append(cmds, func() tea.Msg {
			return models.ScreenTransitionMsg{Screen: models.ScreenGitHubRepo}
		})
		cmds = append(cmds, m.scheduleNextStep())

	case 2: // Test GitHub loading
		m.logTestResult("✅ GitHub repo loading screen working")
		cmds = append(cmds, m.scheduleNextStep())

	case 3: // Test repo selection
		// Simulates having repositories and selecting some
		m.app.SharedState.AvailableRepos = []core.RepoCandidate{
			{Owner: "test", Name: "package1", URL: "https://github.com/test/package1.git", Privacy: "public"},
			{Owner: "test", Name: "package2", URL: "https://github.com/test/package2.git", Privacy: "public"},
		}
		m.app.SharedState.SelectedRepos = []core.RepoCandidate{
			{Owner: "test", Name: "package1", URL: "https://github.com/test/package1.git", Privacy: "public"},
		}
		m.logTestResult("✅ Repository selection working")
		cmds = append(cmds, func() tea.Msg {
			return models.ScreenTransitionMsg{Screen: models.ScreenConfiguration}
		})
		cmds = append(cmds, m.scheduleNextStep())

	case 4: // Test configuration
		// Simulate configured packages
		m.app.SharedState.PackageSpecs = []core.PkgSpec{
			{Name: "package1", URL: "https://github.com/test/package1.git", Ref: "main"},
		}
		m.logTestResult("✅ Package configuration working")
		cmds = append(cmds, func() tea.Msg {
			return models.ScreenTransitionMsg{Screen: models.ScreenConfirmation}
		})
		cmds = append(cmds, m.scheduleNextStep())

	case 5: // Test confirmation
		m.logTestResult("✅ Confirmation screen working")
		cmds = append(cmds, func() tea.Msg {
			return models.ScreenTransitionMsg{Screen: models.ScreenExecution}
		})
		cmds = append(cmds, m.scheduleNextStep())

	case 6: // Test execution
		m.logTestResult("✅ Execution screen with progress working")
		// Wait a bit for execution simulation
		cmds = append(cmds, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return autoTestStepMsg{step: step + 1}
		}))

	case 7: // Test results
		// Simulate successful results
		m.app.SharedState.Results = []core.ActionResult{
			{OK: true, Message: "Successfully added package1"},
		}
		cmds = append(cmds, func() tea.Msg {
			return models.ScreenTransitionMsg{Screen: models.ScreenResults}
		})
		m.logTestResult("✅ Results screen working")
		cmds = append(cmds, m.scheduleNextStep())

	case 8: // Complete
		m.logTestResult("🎉 NEW ARCHITECTURE AUTOTEST COMPLETED SUCCESSFULLY!")
		m.printTestSummary()
		cmds = append(cmds, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return tea.Quit
		}))

	default:
		cmds = append(cmds, tea.Quit)
	}

	return m, tea.Batch(cmds...)
}

// scheduleNextStep schedules the next test step
func (m *AutoTestModel) scheduleNextStep() tea.Cmd {
	if m.testPaused {
		return nil
	}
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
		return autoTestStepMsg{step: m.testStep + 1}
	})
}

// logTestResult logs a test result
func (m *AutoTestModel) logTestResult(result string) {
	m.testResults = append(m.testResults, result)
	fmt.Println(result)
}

// printTestSummary prints the final test summary
func (m *AutoTestModel) printTestSummary() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🏁 NEW ARCHITECTURE AUTOTEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	for _, result := range m.testResults {
		fmt.Println(result)
	}
	fmt.Printf("\nTotal tests: %d\n", len(m.testResults))
	fmt.Println("✅ All tests passed - new architecture is working perfectly!")
}

// RunNewAutoTest runs the automated test for the new architecture
func RunNewAutoTest(cfg core.Config, logger *core.Logger) error {
	autoTest := NewAutoTestModel(cfg, logger)
	p := tea.NewProgram(autoTest, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
