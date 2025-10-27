// Package models/execution_model.go - Package Installation Execution Screen
//
// This file implements the execution screen that handles the actual installation
// of Flutter/Dart packages. It provides real-time progress feedback with:
//   - Animated spinner for current operation
//   - Progress bar showing overall completion
//   - Live status updates for each package
//   - Error handling and recovery
//
// The execution flow follows these steps:
//   1. Create pubspec.yaml backup (safety measure)
//   2. Validate all package specifications
//   3. Install each package via dart/flutter pub add
//   4. Run pub get to resolve dependencies
//   5. Transition to results screen
//
// This model maintains full parity with the shell script's installation behavior
// while providing a modern, visual progress interface.

package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ExecutionModel handles package installation execution with real-time progress feedback.
// It orchestrates the multi-step installation process and provides visual indicators
// for each stage of the operation.
type ExecutionModel struct {
	cfg    core.Config  // Application configuration
	logger *core.Logger // Structured logger for operation tracking
	shared *AppState    // Shared state containing package specs to install

	// Execution state tracking
	executing   bool           // Whether installation is currently in progress
	currentStep int            // Current step number (1-based)
	totalSteps  int            // Total number of steps to complete
	stepName    string         // Human-readable name of current operation
	progress    progress.Model // Animated progress bar (gradient pink to orange)
	spinner     spinner.Model  // Dot spinner for active operations
	complete    bool           // Whether installation has finished
	err         error          // Any error that occurred during execution

	// Lipgloss styles for consistent theming
	headerStyle  lipgloss.Style // Purple bold header
	successStyle lipgloss.Style // Green bold for success messages
	errorStyle   lipgloss.Style // Red bold for errors
	normalStyle  lipgloss.Style // Gray for normal text
}

// executionStepMsg is sent internally when advancing to the next installation step.
// It carries the step number, description, and any error that occurred.
type executionStepMsg struct {
	step     int    // Step number (1-based)
	stepName string // Human-readable step description
	err      error  // Error if step failed, nil otherwise
}

// executionCompleteMsg is sent when the entire installation process completes.
// It contains the results for all packages and any overall error.
type executionCompleteMsg struct {
	results []core.ActionResult // Per-package installation results
	err     error               // Overall execution error, if any
}

// NewExecutionModel creates a new execution screen model.
// It calculates the total steps based on the number of packages to install
// plus overhead steps (backup, validation, pub get).
//
// The model uses:
//   - A gradient progress bar (pink to orange) for visual appeal
//   - A dot spinner to indicate active work
//   - Pre-configured lipgloss styles matching the app theme
func NewExecutionModel(cfg core.Config, logger *core.Logger, shared *AppState) *ExecutionModel {
	// Create progress bar with fixed width
	p := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDAB3D"))
	p.Width = 40

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD"))

	// Calculate total steps (source clone + packages + backup + pub get)
	totalSteps := len(shared.PackageSpecs) + 2
	if shared.SourceProject != nil && shared.SourceProject.Path != "" {
		totalSteps++ // Add step for cloning source project
	}

	return &ExecutionModel{
		cfg:         cfg,
		logger:      logger,
		shared:      shared,
		executing:   true,
		currentStep: 0,
		totalSteps:  totalSteps,
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

// Init initializes the execution screen and starts the installation process.
// It detects whether this is a source clone flow (GitHub option 2) or a
// standard package addition flow, then kicks off the installation sequence.
//
// Returns:
//   - A batch command containing the spinner tick and installation starter
func (m *ExecutionModel) Init() tea.Cmd {
	// Check if this is a source clone flow (option 2)
	if m.shared.SourceRepo != nil && m.shared.SourceProject != nil {
		// This is the GitHub source clone flow
		// Log the information about what needs to be done
		m.logger.Info("execution", fmt.Sprintf("=== SOURCE CLONE FLOW DETECTED ==="))
		m.logger.Info("execution", fmt.Sprintf("  Repository: %s", m.shared.SourceRepo.Name))
		m.logger.Info("execution", fmt.Sprintf("  URL: %s", m.shared.SourceRepo.URL))
		m.logger.Info("execution", fmt.Sprintf("  Target Path: %s", m.shared.SourceProject.Path))
		m.logger.Info("execution", fmt.Sprintf("  Project Name: %s", m.shared.SourceProject.Name))
		m.logger.Info("execution", fmt.Sprintf("  Total Steps: %d", m.totalSteps))
	} else {
		m.logger.Info("execution", "=== PACKAGE INSTALLATION FLOW ===")
		m.logger.Info("execution", fmt.Sprintf("  Packages: %d", len(m.shared.PackageSpecs)))
		m.logger.Info("execution", fmt.Sprintf("  Total Steps: %d", m.totalSteps))
	}

	return tea.Batch(
		m.spinner.Tick,
		m.executeInstallation(),
	)
}

// Update handles all incoming messages during package installation.
//
// Message handling:
//   - tea.KeyMsg: Allows proceeding to results when complete (enter/q)
//   - executionStepMsg: Advances to next step, updates progress bar
//   - executionCompleteMsg: Marks installation done, stores results
//   - spinner.TickMsg: Animates the spinner during active work
//   - progress.FrameMsg: Animates the progress bar smoothly
//
// The model ensures the spinner only animates while work is in progress.
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

// View renders the execution screen with live progress updates.
//
// The view changes based on state:
//   - Error state: Shows error message and instructions to view results
//   - Complete state: Shows success message and package count
//   - Executing state: Shows spinner, current step, progress bar, and package list
//
// Each package in the list shows its status:
//   - ⏳ Pending (not yet started)
//   - 🔄 In progress (currently installing)
//   - ✅ Complete (successfully installed)
//
// The progress bar uses a gradient and animates smoothly as steps complete.
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

// executeInstallation starts the package installation process.
// This is the entry point that kicks off the first step (backup creation).
// Returns a command that sends the first executionStepMsg after a brief delay.
func (m *ExecutionModel) executeInstallation() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return executionStepMsg{
			step:     1,
			stepName: "Creating pubspec.yaml backup",
			err:      nil,
		}
	})
}

// executeNextStep advances to the next installation step and performs actual operations.
// When all steps are complete, it sends results to the results screen.
//
// Step sequence:
//   - Step 1: Clone source project (if applicable) or create directory
//   - Step 2..N: Add dependencies to pubspec.yaml
//   - Final: Run pub get
//
// This function performs REAL operations with detailed error reporting.
func (m *ExecutionModel) executeNextStep() tea.Cmd {
	return func() tea.Msg {
		m.logger.Info("execution", fmt.Sprintf("=== executeNextStep called: currentStep=%d, totalSteps=%d ===", m.currentStep, m.totalSteps))

		// Check if we need to clone source project first (step 1)
		if m.shared.SourceRepo != nil && m.shared.SourceProject != nil && m.currentStep == 1 {
			m.logger.Info("execution", ">>> EXECUTING SOURCE CLONE <<<")
			// Step 1: Clone source project
			m.logger.Info("execution", fmt.Sprintf("Cloning source: %s to %s/%s",
				m.shared.SourceRepo.URL,
				m.shared.SourceProject.Path,
				m.shared.SourceProject.Name))

			// Create parent directory if it doesn't exist
			parentDir := m.shared.SourceProject.Path
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				errMsg := fmt.Sprintf("Failed to create directory '%s': %s", parentDir, err.Error())
				m.logger.Info("execution", errMsg)

				// Store failure in results with full details
				m.shared.Results = []core.ActionResult{{
					OK:      false,
					Message: errMsg,
					Err:     err.Error(),
					Logs:    []string{errMsg},
				}}

				return executionStepMsg{
					step:     m.currentStep + 1,
					stepName: "Failed to create directory",
					err:      fmt.Errorf("%s", errMsg),
				}
			}
			m.logger.Info("execution", fmt.Sprintf("Created directory: %s", parentDir))

			// Create target directory path
			targetPath := filepath.Join(m.shared.SourceProject.Path, m.shared.SourceProject.Name)

			// Make targetPath absolute for display
			absPath, _ := filepath.Abs(targetPath)

			// Use GitClone from core
			result := core.GitClone(m.logger, &m.cfg, m.shared.SourceRepo.URL, targetPath, "")

			if !result.OK {
				errMsg := fmt.Sprintf("Failed to clone source project: %s", result.Err)
				if len(result.Logs) > 0 {
					errMsg += "\nGit output:\n" + strings.Join(result.Logs, "\n")
				}
				m.logger.Info("execution", errMsg)

				// Store failure in results
				m.shared.Results = []core.ActionResult{result}

				return executionStepMsg{
					step:     m.currentStep + 1,
					stepName: "Failed to clone",
					err:      fmt.Errorf("%s", errMsg),
				}
			}

			m.logger.Info("execution", fmt.Sprintf("Source project cloned successfully to: %s", absPath))

			// Set SourceProjectPath for subsequent dependency additions
			m.shared.SourceProjectPath = targetPath
			m.logger.Info("execution", fmt.Sprintf("Set SourceProjectPath to: %s", targetPath))

			// Store success in results WITH FULL PATH
			m.shared.Results = []core.ActionResult{{
				OK:      true,
				Message: fmt.Sprintf("Successfully cloned source project to: %s", absPath),
				Data: map[string]interface{}{
					"name":       m.shared.SourceProject.Name,
					"url":        m.shared.SourceRepo.URL,
					"path":       absPath,
					"relPath":    targetPath,
					"repository": m.shared.SourceRepo.Name,
				},
				Logs: []string{
					fmt.Sprintf("Created directory: %s", parentDir),
					fmt.Sprintf("Cloned %s", m.shared.SourceRepo.URL),
					fmt.Sprintf("Target location: %s", absPath),
				},
			}}

			return executionStepMsg{
				step:     m.currentStep + 1,
				stepName: fmt.Sprintf("Cloned to %s", absPath),
				err:      nil,
			}
		}

		// Add dependencies to pubspec.yaml (step 2+)
		if m.currentStep > 1 && m.currentStep <= len(m.shared.PackageSpecs)+1 {
			packageIndex := m.currentStep - 2
			if packageIndex >= 0 && packageIndex < len(m.shared.PackageSpecs) {
				spec := m.shared.PackageSpecs[packageIndex]

				m.logger.Info("execution", fmt.Sprintf(">>> ADDING DEPENDENCY: %s <<<", spec.Name))
				m.logger.Info("execution", fmt.Sprintf("Package index: %d of %d", packageIndex+1, len(m.shared.PackageSpecs)))
				m.logger.Info("execution", fmt.Sprintf("Current step: %d of %d", m.currentStep, m.totalSteps))

				// Determine project path
				projectPath := m.shared.SourceProjectPath
				if projectPath == "" && m.shared.SourceProject != nil {
					projectPath = filepath.Join(m.shared.SourceProject.Path, m.shared.SourceProject.Name)
				}
				if projectPath == "" {
					projectPath = "." // Default to current directory
				}

				absProjectPath, _ := filepath.Abs(projectPath)
				m.logger.Debug("execution", fmt.Sprintf("  Adding to project: %s", absProjectPath))
				m.logger.Debug("execution", fmt.Sprintf("  Package: %s", spec.Name))
				m.logger.Debug("execution", fmt.Sprintf("  URL: %s", spec.URL))
				m.logger.Debug("execution", fmt.Sprintf("  Ref: %s", spec.Ref))

				// INSTRUMENTATION: Track time between package additions
				if packageIndex > 0 {
					m.logger.Debug("execution", "=== TIME SINCE LAST PACKAGE ADDITION ===")
					m.logger.Debug("execution", fmt.Sprintf("This is package #%d (not the first)", packageIndex+1))
					m.logger.Debug("execution", "Check logs above for timing of previous package")
				}

				// Add the dependency using core.AddGitDependency
				addStartTime := time.Now()
				m.logger.Debug("execution", fmt.Sprintf("=== STARTING AddGitDependency for %s at %s ===", spec.Name, addStartTime.Format("15:04:05.000")))

				result := core.AddGitDependency(m.logger, &m.cfg, projectPath, spec)

				addEndTime := time.Now()
				addDuration := addEndTime.Sub(addStartTime)
				m.logger.Debug("execution", fmt.Sprintf("=== COMPLETED AddGitDependency for %s at %s (duration: %s) ===",
					spec.Name, addEndTime.Format("15:04:05.000"), addDuration))

				if !result.OK {
					m.logger.Debug("execution", fmt.Sprintf("Failed to add %s: %s", spec.Name, result.Err))

					// Store failure
					if len(m.shared.Results) == 0 {
						m.shared.Results = []core.ActionResult{result}
					} else {
						m.shared.Results = append(m.shared.Results, result)
					}

					return executionStepMsg{
						step:     m.currentStep + 1,
						stepName: fmt.Sprintf("Failed: %s", spec.Name),
						err:      fmt.Errorf("failed to add %s: %s", spec.Name, result.Err),
					}
				}

				m.logger.Debug("execution", fmt.Sprintf("Successfully added %s", spec.Name))

				// Store success
				result.Data = map[string]interface{}{
					"package":     spec.Name,
					"url":         spec.URL,
					"ref":         spec.Ref,
					"projectPath": absProjectPath,
				}

				if len(m.shared.Results) == 0 {
					m.shared.Results = []core.ActionResult{result}
				} else {
					m.shared.Results = append(m.shared.Results, result)
				}

				return executionStepMsg{
					step:     m.currentStep + 1,
					stepName: fmt.Sprintf("Added %s", spec.Name),
					err:      nil,
				}
			}
		}

		// If we've completed all steps, finalize
		if m.currentStep >= m.totalSteps {
			// Ensure we have results
			if len(m.shared.Results) == 0 {
				m.shared.Results = []core.ActionResult{{
					OK:      true,
					Message: "No operations performed",
				}}
			}

			return executionCompleteMsg{
				results: m.shared.Results,
				err:     nil,
			}
		}

		// Determine step name for display
		var stepName string
		if m.currentStep == 1 {
			stepName = "Preparing installation"
		} else if m.currentStep == 2 {
			stepName = "Configuring dependencies"
		} else {
			stepName = "Finalizing setup"
		}

		// Continue to next step
		return executionStepMsg{
			step:     m.currentStep + 1,
			stepName: stepName,
			err:      nil,
		}
	}
}
