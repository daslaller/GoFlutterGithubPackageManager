// Package models/conflict_resolver_model.go - Dependency Conflict Resolution Screen
//
// This file implements the interactive conflict resolution screen that allows users
// to manually resolve dependency conflicts that couldn't be automatically resolved.
// It provides:
//   - Clear presentation of conflict details (type, conflicting packages, etc.)
//   - "Override All (Recommended)" option for batch resolution
//   - Progress feedback with spinner during resolution
//   - Interactive options for resolution strategies
//   - Ability to skip conflicting packages
//   - Retry installation with chosen resolution method
//
// The conflict resolver is invoked when the execution phase detects packages with
// needs_resolution=true in their result data.

package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ConflictResolverModel handles interactive resolution of dependency conflicts
type ConflictResolverModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// Conflict tracking
	conflictIndices []int  // Indices into shared.Results that have conflicts
	currentIndex    int    // Current position in conflictIndices
	selectedOption  int    // Currently selected resolution option
	resolving       bool   // Whether we're currently attempting resolution
	resolveError    string // Error from last resolution attempt
	resolveSuccess  bool   // Whether last resolution succeeded

	// Batch resolution state
	batchResolving     bool   // Whether we're in batch resolution mode
	batchCurrentIndex  int    // Current package being resolved in batch mode
	batchSuccessCount  int    // Number of successfully resolved packages
	batchFailureCount  int    // Number of failed resolutions
	batchStatusMessage string // Current status message during batch resolution

	// UI components
	spinner  spinner.Model  // Animated spinner for resolution progress
	progress progress.Model // Progress bar for batch resolution

	// UI styles
	headerStyle   lipgloss.Style
	titleStyle    lipgloss.Style
	conflictStyle lipgloss.Style
	optionStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	errorStyle    lipgloss.Style
	successStyle  lipgloss.Style
	normalStyle   lipgloss.Style
	progressStyle lipgloss.Style
}

// Resolution options available to the user
const (
	optionOverrideAll = iota // Override all conflicts (recommended)
	optionUseOverride        // Try using dependency_overrides for current package
	optionSkipPackage        // Skip this package and continue
	optionRetry              // Retry without changes
	optionContinue           // Continue to results
)

// resolveCompleteMsg is sent when a single package resolution completes
type resolveCompleteMsg struct {
	success bool
	err     error
}

// batchResolveNextMsg is sent to trigger resolution of the next package in batch mode
type batchResolveNextMsg struct{}

// NewConflictResolverModel creates a new conflict resolver screen
func NewConflictResolverModel(cfg core.Config, logger *core.Logger, shared *AppState) *ConflictResolverModel {
	// Find all packages that need conflict resolution
	var conflictIndices []int
	for i, result := range shared.Results {
		if !result.OK && result.Data != nil {
			if needsResolution, ok := result.Data["needs_resolution"].(bool); ok && needsResolution {
				conflictIndices = append(conflictIndices, i)
			}
		}
	}

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD"))

	// Create progress bar
	p := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDAB3D"))
	p.Width = 40

	return &ConflictResolverModel{
		cfg:             cfg,
		logger:          logger,
		shared:          shared,
		conflictIndices: conflictIndices,
		currentIndex:    0,
		selectedOption:  optionOverrideAll, // Default to recommended option
		spinner:         s,
		progress:        p,

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).
			Bold(true),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),

		conflictStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		optionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")).
			Bold(true),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		progressStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")).
			Bold(true),
	}
}

// Init initializes the conflict resolver
func (m *ConflictResolverModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles user input for conflict resolution
func (m *ConflictResolverModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't allow input while resolving
		if m.resolving || m.batchResolving {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			// Skip to results
			return m, TransitionToScreen(ScreenResults)
		case "up", "k":
			if m.selectedOption > 0 {
				m.selectedOption--
			}
		case "down", "j":
			if m.selectedOption < optionContinue {
				m.selectedOption++
			}
		case "left", "h":
			if m.currentIndex > 0 {
				m.currentIndex--
				m.selectedOption = optionOverrideAll
				m.resolveError = ""
				m.resolveSuccess = false
			}
		case "right", "l":
			if m.currentIndex < len(m.conflictIndices)-1 {
				m.currentIndex++
				m.selectedOption = optionOverrideAll
				m.resolveError = ""
				m.resolveSuccess = false
			}
		case "enter":
			return m, m.handleOptionSelection()
		}
		return m, nil

	case resolveCompleteMsg:
		// Single package resolution completed
		m.resolving = false
		if msg.success {
			m.resolveSuccess = true
			m.resolveError = ""
		} else {
			m.resolveSuccess = false
			if msg.err != nil {
				m.resolveError = msg.err.Error()
			}
		}
		return m, nil

	case batchResolveNextMsg:
		// Process next package in batch resolution
		if m.batchCurrentIndex < len(m.conflictIndices) {
			return m, m.resolveBatchPackage()
		}
		// Batch resolution complete
		m.batchResolving = false
		m.logger.Info("conflict_resolver", fmt.Sprintf("Batch resolution complete: %d succeeded, %d failed", m.batchSuccessCount, m.batchFailureCount))

		// If all resolved successfully, transition to results
		if m.batchFailureCount == 0 && len(m.conflictIndices) == 0 {
			return m, TransitionToScreen(ScreenResults)
		}
		return m, nil

	case spinner.TickMsg:
		if m.resolving || m.batchResolving {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case progress.FrameMsg:
		if m.batchResolving {
			var cmd tea.Cmd
			progressModel, cmd := m.progress.Update(msg)
			m.progress = progressModel.(progress.Model)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the conflict resolver screen
func (m *ConflictResolverModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.headerStyle.Render("🔧 Dependency Conflict Resolver") + "\n\n")

	// If in batch resolution mode, show progress
	if m.batchResolving {
		b.WriteString(m.progressStyle.Render(fmt.Sprintf("%s Resolving conflicts automatically...", m.spinner.View())) + "\n\n")

		// Calculate progress percentage
		totalConflicts := m.batchSuccessCount + m.batchFailureCount + len(m.conflictIndices)
		completed := m.batchSuccessCount + m.batchFailureCount
		progressPercent := 0.0
		if totalConflicts > 0 {
			progressPercent = float64(completed) / float64(totalConflicts)
		}

		// Show progress bar
		b.WriteString(m.progress.ViewAs(progressPercent) + "\n\n")

		// Show statistics
		b.WriteString(m.normalStyle.Render(fmt.Sprintf("Progress: %d / %d packages", completed, totalConflicts)) + "\n")
		b.WriteString(m.normalStyle.Render(fmt.Sprintf("✅ Success: %d  ❌ Failed: %d  ⏳ Remaining: %d", m.batchSuccessCount, m.batchFailureCount, len(m.conflictIndices))) + "\n\n")

		if m.batchStatusMessage != "" {
			b.WriteString(m.normalStyle.Render(m.batchStatusMessage) + "\n")
		}

		return b.String()
	}

	// If no conflicts, show message and continue option
	if len(m.conflictIndices) == 0 {
		b.WriteString(m.successStyle.Render("✅ All conflicts resolved!") + "\n\n")
		b.WriteString(m.normalStyle.Render(fmt.Sprintf("Successfully resolved: %d packages", m.batchSuccessCount)) + "\n")
		if m.batchFailureCount > 0 {
			b.WriteString(m.errorStyle.Render(fmt.Sprintf("Failed to resolve: %d packages", m.batchFailureCount)) + "\n")
		}
		b.WriteString("\nPress Enter to continue to results\n")
		return b.String()
	}

	// Show current conflict
	resultIndex := m.conflictIndices[m.currentIndex]
	result := m.shared.Results[resultIndex]
	spec := m.shared.PackageSpecs[resultIndex]

	// Conflict counter
	b.WriteString(m.normalStyle.Render(fmt.Sprintf("Conflict %d of %d", m.currentIndex+1, len(m.conflictIndices))) + "\n\n")

	// Package name
	b.WriteString(m.titleStyle.Render(fmt.Sprintf("Package: %s", spec.Name)) + "\n")
	b.WriteString(m.normalStyle.Render(fmt.Sprintf("URL: %s", spec.URL)) + "\n\n")

	// Conflict details
	if result.Data != nil {
		if conflictType, ok := result.Data["conflict_type"].(string); ok && conflictType != "" {
			b.WriteString(m.conflictStyle.Render(fmt.Sprintf("⚠️  Conflict Type: %s", conflictType)) + "\n")
		}
		if conflictingPkg, ok := result.Data["conflicting_pkg"].(string); ok && conflictingPkg != "" {
			b.WriteString(m.normalStyle.Render(fmt.Sprintf("   Conflicting Package: %s", conflictingPkg)) + "\n")
		}
		if userMessage, ok := result.Data["user_message"].(string); ok && userMessage != "" {
			b.WriteString(m.normalStyle.Render(fmt.Sprintf("   %s", userMessage)) + "\n")
		}
	}
	b.WriteString("\n")

	// Show resolution status if present
	if m.resolving {
		b.WriteString(m.progressStyle.Render(fmt.Sprintf("%s Resolving conflict...", m.spinner.View())) + "\n\n")
	} else if m.resolveSuccess {
		b.WriteString(m.successStyle.Render("✅ Conflict resolved successfully!") + "\n\n")
	} else if m.resolveError != "" {
		b.WriteString(m.errorStyle.Render(fmt.Sprintf("❌ Resolution failed: %s", m.resolveError)) + "\n\n")
	}

	// Resolution options
	b.WriteString(m.headerStyle.Render("Resolution Options:") + "\n\n")

	options := []string{
		"🚀 Override All Conflicts (Recommended)",
		"📝 Use dependency override for this package",
		"⏭️  Skip this package",
		"🔄 Retry installation",
		"✅ Continue to results",
	}

	for i, option := range options {
		prefix := "  "
		style := m.optionStyle
		if i == m.selectedOption {
			prefix = "> "
			style = m.selectedStyle
		}
		b.WriteString(style.Render(prefix+option) + "\n")
	}

	// Navigation hints
	b.WriteString("\n")
	if len(m.conflictIndices) > 1 {
		b.WriteString(m.normalStyle.Render("↑/↓: Select option  ←/→: Navigate conflicts  Enter: Execute  Q: Skip to results") + "\n")
	} else {
		b.WriteString(m.normalStyle.Render("↑/↓: Select option  Enter: Execute  Q: Skip to results") + "\n")
	}

	return b.String()
}

// handleOptionSelection executes the selected resolution option
func (m *ConflictResolverModel) handleOptionSelection() tea.Cmd {
	switch m.selectedOption {
	case optionOverrideAll:
		// Start batch resolution of all conflicts
		return m.startBatchResolution()
	case optionUseOverride:
		// Try resolution with dependency override for current package
		return m.resolveSinglePackage()
	case optionSkipPackage:
		// Mark as skipped and move to next
		return m.skipPackage()
	case optionRetry:
		// Retry installation without changes
		return m.retryInstallation()
	case optionContinue:
		// Continue to results screen
		return TransitionToScreen(ScreenResults)
	}
	return nil
}

// startBatchResolution begins automatic resolution of all conflicts
func (m *ConflictResolverModel) startBatchResolution() tea.Cmd {
	m.batchResolving = true
	m.batchCurrentIndex = 0
	m.batchSuccessCount = 0
	m.batchFailureCount = 0
	m.batchStatusMessage = "Starting batch resolution..."
	m.logger.Info("conflict_resolver", fmt.Sprintf("Starting batch resolution for %d conflicts", len(m.conflictIndices)))

	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg { return batchResolveNextMsg{} },
	)
}

// resolveBatchPackage resolves the current package in batch mode
func (m *ConflictResolverModel) resolveBatchPackage() tea.Cmd {
	return func() tea.Msg {
		if m.batchCurrentIndex >= len(m.conflictIndices) {
			return batchResolveNextMsg{} // Trigger completion
		}

		resultIndex := m.conflictIndices[m.batchCurrentIndex]
		spec := m.shared.PackageSpecs[resultIndex]

		m.batchStatusMessage = fmt.Sprintf("Resolving: %s", spec.Name)
		m.logger.Info("conflict_resolver", fmt.Sprintf("Batch resolving package %d/%d: %s", m.batchCurrentIndex+1, len(m.conflictIndices), spec.Name))

		// Determine project path
		projectPath := m.shared.SourceProjectPath
		if projectPath == "" {
			projectPath = "."
		}

		// Attempt resolution with autoResolve=true
		newResult := core.AddGitDependency(m.logger, &m.cfg, projectPath, spec, true)

		if newResult.OK {
			// Success! Update the result
			m.shared.Results[resultIndex] = newResult
			m.batchSuccessCount++
			m.logger.Info("conflict_resolver", fmt.Sprintf("✅ Batch resolved: %s", spec.Name))

			// Remove this index from conflict list
			m.conflictIndices = append(m.conflictIndices[:m.batchCurrentIndex], m.conflictIndices[m.batchCurrentIndex+1:]...)
			// Don't increment index since we removed an element
		} else {
			// Failed - keep in list and move to next
			m.batchFailureCount++
			m.logger.Info("conflict_resolver", fmt.Sprintf("❌ Failed to batch resolve: %s - %s", spec.Name, newResult.Err))
			m.batchCurrentIndex++
		}

		// Trigger next resolution
		return batchResolveNextMsg{}
	}
}

// resolveSinglePackage attempts to resolve the current package only
func (m *ConflictResolverModel) resolveSinglePackage() tea.Cmd {
	return func() tea.Msg {
		m.resolving = true
		defer func() { m.resolving = false }()

		resultIndex := m.conflictIndices[m.currentIndex]
		spec := m.shared.PackageSpecs[resultIndex]

		m.logger.Info("conflict_resolver", fmt.Sprintf("Attempting override resolution for %s", spec.Name))

		// Determine project path
		projectPath := m.shared.SourceProjectPath
		if projectPath == "" {
			projectPath = "."
		}

		// Attempt resolution with override
		newResult := core.AddGitDependency(m.logger, &m.cfg, projectPath, spec, true)

		if newResult.OK {
			// Success! Update the result
			m.shared.Results[resultIndex] = newResult
			m.logger.Info("conflict_resolver", fmt.Sprintf("✅ Successfully resolved conflict for %s", spec.Name))

			// Remove this index from conflict list
			m.conflictIndices = append(m.conflictIndices[:m.currentIndex], m.conflictIndices[m.currentIndex+1:]...)

			// If no more conflicts, transition to results
			if len(m.conflictIndices) == 0 {
				return resolveCompleteMsg{success: true, err: nil}
			}

			// Otherwise, stay on current index (which now points to next conflict)
			if m.currentIndex >= len(m.conflictIndices) {
				m.currentIndex = len(m.conflictIndices) - 1
			}

			return resolveCompleteMsg{success: true, err: nil}
		}

		// Failed - show error
		m.logger.Info("conflict_resolver", fmt.Sprintf("❌ Failed to resolve conflict for %s: %s", spec.Name, newResult.Err))
		return resolveCompleteMsg{success: false, err: fmt.Errorf("%s", newResult.Err)}
	}
}

// skipPackage marks the current package as skipped and moves to the next
func (m *ConflictResolverModel) skipPackage() tea.Cmd {
	return func() tea.Msg {
		resultIndex := m.conflictIndices[m.currentIndex]
		spec := m.shared.PackageSpecs[resultIndex]

		m.logger.Info("conflict_resolver", fmt.Sprintf("⏭️  Skipping package: %s", spec.Name))

		// Update result to indicate it was skipped
		if m.shared.Results[resultIndex].Data == nil {
			m.shared.Results[resultIndex].Data = make(map[string]interface{})
		}
		m.shared.Results[resultIndex].Data["skipped"] = true

		// Remove from conflict list
		m.conflictIndices = append(m.conflictIndices[:m.currentIndex], m.conflictIndices[m.currentIndex+1:]...)

		// If no more conflicts, transition to results
		if len(m.conflictIndices) == 0 {
			return ScreenTransitionMsg{Screen: ScreenResults}
		}

		// Otherwise, adjust index if needed
		if m.currentIndex >= len(m.conflictIndices) {
			m.currentIndex = len(m.conflictIndices) - 1
		}

		return nil
	}
}

// retryInstallation retries the installation without changes
func (m *ConflictResolverModel) retryInstallation() tea.Cmd {
	return func() tea.Msg {
		m.resolving = true
		defer func() { m.resolving = false }()

		resultIndex := m.conflictIndices[m.currentIndex]
		spec := m.shared.PackageSpecs[resultIndex]

		m.logger.Info("conflict_resolver", fmt.Sprintf("🔄 Retrying installation for %s", spec.Name))

		// Determine project path
		projectPath := m.shared.SourceProjectPath
		if projectPath == "" {
			projectPath = "."
		}

		// Retry with autoResolve=false (same as initial attempt)
		newResult := core.AddGitDependency(m.logger, &m.cfg, projectPath, spec, false)

		if newResult.OK {
			// Success! Update the result
			m.shared.Results[resultIndex] = newResult
			m.logger.Info("conflict_resolver", fmt.Sprintf("✅ Successfully installed %s on retry", spec.Name))

			// Remove this index from conflict list
			m.conflictIndices = append(m.conflictIndices[:m.currentIndex], m.conflictIndices[m.currentIndex+1:]...)

			// If no more conflicts, transition to results
			if len(m.conflictIndices) == 0 {
				return resolveCompleteMsg{success: true, err: nil}
			}

			// Otherwise, stay on current index
			if m.currentIndex >= len(m.conflictIndices) {
				m.currentIndex = len(m.conflictIndices) - 1
			}

			return resolveCompleteMsg{success: true, err: nil}
		}

		// Failed - show error
		m.logger.Info("conflict_resolver", fmt.Sprintf("❌ Retry failed for %s: %s", spec.Name, newResult.Err))
		return resolveCompleteMsg{success: false, err: fmt.Errorf("%s", newResult.Err)}
	}
}
