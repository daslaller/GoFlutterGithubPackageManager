// Package models/conflict_resolver_model.go - Dependency Conflict Resolution Screen
//
// This file implements the interactive conflict resolution screen that allows users
// to manually resolve dependency conflicts that couldn't be automatically resolved.
// It provides:
//   - Clear presentation of conflict details (type, conflicting packages, etc.)
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

	// UI styles
	headerStyle   lipgloss.Style
	titleStyle    lipgloss.Style
	conflictStyle lipgloss.Style
	optionStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	errorStyle    lipgloss.Style
	successStyle  lipgloss.Style
	normalStyle   lipgloss.Style
}

// Resolution options available to the user
const (
	optionUseOverride = iota // Try using dependency_overrides
	optionSkipPackage        // Skip this package and continue
	optionRetry              // Retry without changes
	optionViewDetails        // Show full error output
	optionContinue           // Continue to results
)

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

	return &ConflictResolverModel{
		cfg:             cfg,
		logger:          logger,
		shared:          shared,
		conflictIndices: conflictIndices,
		currentIndex:    0,
		selectedOption:  optionUseOverride,

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
	}
}

// Init initializes the conflict resolver
func (m *ConflictResolverModel) Init() tea.Cmd {
	return nil
}

// Update handles user input for conflict resolution
func (m *ConflictResolverModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if !m.resolving {
				// Skip to results
				return m, TransitionToScreen(ScreenResults)
			}
		case "up", "k":
			if !m.resolving && m.selectedOption > 0 {
				m.selectedOption--
			}
		case "down", "j":
			if !m.resolving && m.selectedOption < optionContinue {
				m.selectedOption++
			}
		case "left", "h":
			if !m.resolving && m.currentIndex > 0 {
				m.currentIndex--
				m.selectedOption = optionUseOverride
				m.resolveError = ""
			}
		case "right", "l":
			if !m.resolving && m.currentIndex < len(m.conflictIndices)-1 {
				m.currentIndex++
				m.selectedOption = optionUseOverride
				m.resolveError = ""
			}
		case "enter":
			if !m.resolving {
				return m, m.handleOptionSelection()
			}
		}
	}

	return m, nil
}

// View renders the conflict resolver screen
func (m *ConflictResolverModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.headerStyle.Render("🔧 Dependency Conflict Resolver") + "\n\n")

	// If no conflicts, show message and continue option
	if len(m.conflictIndices) == 0 {
		b.WriteString(m.successStyle.Render("✅ No conflicts to resolve") + "\n\n")
		b.WriteString("Press Enter to continue to results\n")
		return b.String()
	}

	// Show current conflict
	resultIndex := m.conflictIndices[m.currentIndex]
	result := m.shared.Results[resultIndex]
	spec := m.shared.PackageSpecs[resultIndex]

	// Conflict counter
	b.WriteString(m.normalStyle.Render(fmt.Sprintf("Conflict %d of %d", m.currentIndex+1, len(m.conflictIndices))) + "\n\n")

	// Package name
	b.WriteString(m.titleStyle.Render(fmt.Sprintf("Package: %s", spec.Name)) + "\n\n")

	// Conflict details
	if result.Data != nil {
		if conflictType, ok := result.Data["conflict_type"].(string); ok && conflictType != "" {
			b.WriteString(m.conflictStyle.Render(fmt.Sprintf("Conflict Type: %s", conflictType)) + "\n")
		}
		if conflictingPkg, ok := result.Data["conflicting_pkg"].(string); ok && conflictingPkg != "" {
			b.WriteString(m.normalStyle.Render(fmt.Sprintf("Conflicting Package: %s", conflictingPkg)) + "\n")
		}
		if userMessage, ok := result.Data["user_message"].(string); ok && userMessage != "" {
			b.WriteString(m.normalStyle.Render(fmt.Sprintf("\n%s", userMessage)) + "\n")
		}
		if suggestedFix, ok := result.Data["suggested_fix"].(string); ok && suggestedFix != "" {
			b.WriteString(m.normalStyle.Render(fmt.Sprintf("Suggested Fix: %s", suggestedFix)) + "\n")
		}
	}
	b.WriteString("\n")

	// Error message if retry failed
	if m.resolveError != "" {
		b.WriteString(m.errorStyle.Render(fmt.Sprintf("❌ Resolution failed: %s", m.resolveError)) + "\n\n")
	}

	// Resolution options
	b.WriteString(m.headerStyle.Render("Resolution Options:") + "\n\n")

	options := []string{
		"📝 Use dependency override and retry",
		"⏭️  Skip this package",
		"🔄 Retry installation",
		"📋 View full error details",
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
	b.WriteString(m.normalStyle.Render("↑/↓: Select option  ←/→: Navigate conflicts  Enter: Execute  Q: Skip to results") + "\n")

	return b.String()
}

// handleOptionSelection executes the selected resolution option
func (m *ConflictResolverModel) handleOptionSelection() tea.Cmd {
	switch m.selectedOption {
	case optionUseOverride:
		// Try resolution with dependency override
		return m.resolveWithOverride()
	case optionSkipPackage:
		// Mark as skipped and move to next
		return m.skipPackage()
	case optionRetry:
		// Retry installation without changes
		return m.retryInstallation()
	case optionViewDetails:
		// Show full error (for now, just continue - could be expanded later)
		return nil
	case optionContinue:
		// Continue to results screen
		return TransitionToScreen(ScreenResults)
	}
	return nil
}

// resolveWithOverride attempts to resolve the conflict using dependency overrides
func (m *ConflictResolverModel) resolveWithOverride() tea.Cmd {
	return func() tea.Msg {
		m.resolving = true
		defer func() { m.resolving = false }()

		resultIndex := m.conflictIndices[m.currentIndex]
		spec := m.shared.PackageSpecs[resultIndex]
		result := m.shared.Results[resultIndex]

		m.logger.Info("conflict_resolver", fmt.Sprintf("Attempting override resolution for %s", spec.Name))

		// Extract conflict analysis from result data
		analysis := core.ConflictAnalysis{}
		if result.Data != nil {
			if conflictType, ok := result.Data["conflict_type"].(string); ok {
				analysis.ConflictType = conflictType
			}
			if conflictingPkg, ok := result.Data["conflicting_pkg"].(string); ok {
				analysis.ConflictingPkg = conflictingPkg
			}
		}

		// Determine project path
		projectPath := m.shared.SourceProjectPath
		if projectPath == "" {
			projectPath = "."
		}

		// Attempt resolution with override
		// For now, we'll retry with autoResolve=true
		newResult := core.AddGitDependency(m.logger, &m.cfg, projectPath, spec, true)

		if newResult.OK {
			// Success! Update the result
			m.shared.Results[resultIndex] = newResult
			m.logger.Info("conflict_resolver", fmt.Sprintf("✅ Successfully resolved conflict for %s", spec.Name))

			// Remove this index from conflict list
			m.conflictIndices = append(m.conflictIndices[:m.currentIndex], m.conflictIndices[m.currentIndex+1:]...)

			// If no more conflicts, transition to results
			if len(m.conflictIndices) == 0 {
				return ScreenTransitionMsg{Screen: ScreenResults}
			}

			// Otherwise, stay on current index (which now points to next conflict)
			if m.currentIndex >= len(m.conflictIndices) {
				m.currentIndex = len(m.conflictIndices) - 1
			}
		} else {
			// Failed - show error
			m.resolveError = newResult.Err
			m.logger.Info("conflict_resolver", fmt.Sprintf("❌ Failed to resolve conflict for %s: %s", spec.Name, newResult.Err))
		}

		return nil
	}
}

// skipPackage marks the current package as skipped and moves to the next
func (m *ConflictResolverModel) skipPackage() tea.Cmd {
	return func() tea.Msg {
		resultIndex := m.conflictIndices[m.currentIndex]
		spec := m.shared.PackageSpecs[resultIndex]

		m.logger.Info("conflict_resolver", fmt.Sprintf("⏭️  Skipping package: %s", spec.Name))

		// Update result to indicate it was skipped
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
				return ScreenTransitionMsg{Screen: ScreenResults}
			}

			// Otherwise, stay on current index
			if m.currentIndex >= len(m.conflictIndices) {
				m.currentIndex = len(m.conflictIndices) - 1
			}
		} else {
			// Failed - show error
			m.resolveError = newResult.Err
			m.logger.Info("conflict_resolver", fmt.Sprintf("❌ Retry failed for %s: %s", spec.Name, newResult.Err))
		}

		return nil
	}
}
