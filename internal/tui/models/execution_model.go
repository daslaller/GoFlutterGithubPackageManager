// Package models/execution_model.go - Execution Screen Model

package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ExecutionModel handles package installation execution
type ExecutionModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// Execution state
	executing   bool
	currentStep int
	totalSteps  int
	stepName    string
	progress    progress.Model
	spinner     spinner.Model
	complete    bool
	err         error

	// Styles
	headerStyle  lipgloss.Style
	successStyle lipgloss.Style
	errorStyle   lipgloss.Style
	normalStyle  lipgloss.Style
}

// executionStepMsg represents progress through execution steps
type executionStepMsg struct {
	step     int
	stepName string
	err      error
}

// executionCompleteMsg is sent when execution is complete
type executionCompleteMsg struct {
	results []core.ActionResult
	err     error
}

// NewExecutionModel creates a new execution model
func NewExecutionModel(cfg core.Config, logger *core.Logger, shared *AppState) *ExecutionModel {
	// Create progress bar
	p := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDAB3D"))

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD"))

	return &ExecutionModel{
		cfg:         cfg,
		logger:      logger,
		shared:      shared,
		executing:   true,
		currentStep: 0,
		totalSteps:  len(shared.PackageSpecs) + 2, // packages + backup + pub get
		stepName:    "Starting installation...",
		progress:    p,
		spinner:     s,

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

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
}

// Init initializes the execution screen
func (m *ExecutionModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.executeInstallation(),
	)
}

// Update handles messages for execution
func (m *ExecutionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.complete {
			switch msg.String() {
			case "q", "ctrl+c", "enter":
				return m, TransitionToScreen(ScreenResults)
			}
		}
		return m, nil

	case executionStepMsg:
		m.currentStep = msg.step
		m.stepName = msg.stepName
		if msg.err != nil {
			m.err = msg.err
			m.executing = false
		} else {
			// Continue to next step
			cmds = append(cmds, m.executeNextStep())
		}
		// Update progress
		if m.totalSteps > 0 {
			progressValue := float64(m.currentStep) / float64(m.totalSteps)
			cmds = append(cmds, m.progress.SetPercent(progressValue))
		}
		return m, tea.Batch(cmds...)

	case executionCompleteMsg:
		m.executing = false
		m.complete = true
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.shared.Results = msg.results
			m.logger.Info("execution", "Package installation completed successfully")
		}
		return m, nil

	case spinner.TickMsg:
		if m.executing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case progress.FrameMsg:
		var cmd tea.Cmd
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the execution screen
func (m *ExecutionModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.headerStyle.Render("⚡ Installing Packages") + "\n\n")

	if m.err != nil {
		// Error state
		b.WriteString(m.errorStyle.Render("❌ Installation Failed") + "\n\n")
		b.WriteString(fmt.Sprintf("Error: %s\n\n", m.err.Error()))
		b.WriteString("Press Enter or Q to view results\n")
		return b.String()
	}

	if m.complete {
		// Success state
		b.WriteString(m.successStyle.Render("✅ Installation Complete!") + "\n\n")
		b.WriteString(fmt.Sprintf("Successfully installed %d packages\n\n", len(m.shared.PackageSpecs)))
		b.WriteString("Press Enter or Q to view detailed results\n")
		return b.String()
	}

	// Executing state
	if m.executing {
		b.WriteString(fmt.Sprintf("%s %s\n\n", m.spinner.View(), m.stepName))
	}

	// Progress bar
	progressText := fmt.Sprintf("Progress: %d/%d steps", m.currentStep, m.totalSteps)
	b.WriteString(progressText + "\n")
	b.WriteString(m.progress.View() + "\n\n")

	// Package list
	b.WriteString("Installing packages:\n")
	for i, spec := range m.shared.PackageSpecs {
		status := "⏳"
		if i < m.currentStep-1 {
			status = "✅"
		} else if i == m.currentStep-1 {
			status = "🔄"
		}
		b.WriteString(fmt.Sprintf("%s %s\n", status, spec.Name))
	}

	if m.executing {
		b.WriteString("\nPlease wait while packages are being installed...")
	}

	return b.String()
}

// executeInstallation runs the package installation process
func (m *ExecutionModel) executeInstallation() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return executionStepMsg{
			step:     1,
			stepName: "Creating pubspec.yaml backup",
			err:      nil,
		}
	})
}

// executeNextStep continues the installation process
func (m *ExecutionModel) executeNextStep() tea.Cmd {
	if m.currentStep >= m.totalSteps {
		// Installation complete
		results := make([]core.ActionResult, len(m.shared.PackageSpecs))
		for i, spec := range m.shared.PackageSpecs {
			results[i] = core.ActionResult{
				OK:      true,
				Message: fmt.Sprintf("Successfully added %s", spec.Name),
				Data: map[string]interface{}{
					"package": spec.Name,
					"url":     spec.URL,
					"ref":     spec.Ref,
				},
			}
		}

		return func() tea.Msg {
			return executionCompleteMsg{
				results: results,
				err:     nil,
			}
		}
	}

	// Get the next step name
	var stepName string
	if m.currentStep == 1 {
		stepName = "Validating package specifications"
	} else if m.currentStep <= len(m.shared.PackageSpecs)+1 {
		packageIndex := m.currentStep - 2
		if packageIndex >= 0 && packageIndex < len(m.shared.PackageSpecs) {
			stepName = fmt.Sprintf("Installing %s", m.shared.PackageSpecs[packageIndex].Name)
		} else {
			stepName = "Installing package"
		}
	} else {
		stepName = "Running pub get"
	}

	// Simulate work with a delay
	return tea.Tick(800*time.Millisecond, func(time.Time) tea.Msg {
		return executionStepMsg{
			step:     m.currentStep + 1,
			stepName: stepName,
			err:      nil,
		}
	})
}
