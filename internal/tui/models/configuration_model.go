// Package models/configuration_model.go - Package Configuration Screen
//
// This file implements the interactive configuration screen where users fine-tune
// each selected package before installation. For each package, users can specify:
//   - Package name (defaults to repository name)
//   - Git ref (branch, tag, or commit hash - defaults to "main")
//   - Subdirectory (optional, for monorepo packages)
//
// The screen uses a wizard-style flow, presenting one package at a time with
// three text input fields. Navigation uses Tab/Shift+Tab between fields and
// Enter to advance to the next package. All inputs support window resizing.
//
// After all packages are configured, this screen generates core.PkgSpec objects
// that contain all the information needed for the execution screen to perform
// the actual `dart pub add` or `flutter pub add` commands.

package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ConfigurationModel handles the interactive package configuration wizard.
// It manages a collection of text inputs (3 per package) and provides
// a focused, one-package-at-a-time configuration experience.
type ConfigurationModel struct {
	cfg    core.Config  // Application configuration
	logger *core.Logger // Structured logger for tracking user choices
	shared *AppState    // Shared state containing selected dependencies

	// Configuration wizard state
	currentRepo  int               // Index of current package being configured (0-based)
	currentField int               // Current field focus: 0=name, 1=ref, 2=subdir
	packageSpecs []core.PkgSpec    // Generated specs ready for installation
	inputs       []textinput.Model // Flat array: [pkg0_name, pkg0_ref, pkg0_subdir, pkg1_name, ...]
	complete     bool              // Whether all packages have been configured

	// Package name fetching state
	fetchingNames bool   // Whether we're currently fetching package names from git
	fetchError    string // Error message if fetching failed

	// Lipgloss styles for visual hierarchy
	headerStyle   lipgloss.Style // Purple bold for headers
	selectedStyle lipgloss.Style // White on purple background for active field
	normalStyle   lipgloss.Style // Gray for inactive labels
	helpStyle     lipgloss.Style // Gray italic for help text
}

// packageNamesFetchedMsg is sent when package names have been fetched from git repositories
type packageNamesFetchedMsg struct {
	err error
}

// NewConfigurationModel creates a new package configuration wizard.
// The model creates three text inputs per selected package and initializes
// them with sensible defaults (package name from repo, "main" for ref).
//
// Color scheme matches the app theme:
//   - Headers: Purple (color 211)
//   - Selected field: White text on purple background (#8B5CF6)
//   - Normal text: Gray (color 241)
//   - Help text: Gray italic
func NewConfigurationModel(cfg core.Config, logger *core.Logger, shared *AppState) *ConfigurationModel {
	return &ConfigurationModel{
		cfg:          cfg,
		logger:       logger,
		shared:       shared,
		currentRepo:  0,
		currentField: 1, // Start at field 1 (ref) since field 0 (name) is read-only

		// Styles
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).
			Bold(true),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#8B5CF6")).
			Padding(0, 1),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true),
	}
}

// Init initializes the configuration screen by creating and populating all text inputs.
// Returns a batch of commands including cursor blink and package name fetching.
func (m *ConfigurationModel) Init() tea.Cmd {
	m.fetchingNames = true
	return tea.Batch(
		textinput.Blink,
		m.fetchPackageNames(),
	)
}

// Update handles all messages for the configuration wizard.
//
// Message handling:
//   - tea.KeyMsg: Navigation (tab/shift+tab), advancing (enter), quitting (q)
//   - Other: Forwarded to the currently focused text input for typing
//
// Input array layout: Each package uses 3 consecutive inputs [name, ref, subdir].
// To get inputs for package N, use indices [N*3, N*3+1, N*3+2].
func (m *ConfigurationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case packageNamesFetchedMsg:
		// Package names have been fetched, now setup inputs with correct names
		m.fetchingNames = false
		if msg.err != nil {
			m.fetchError = msg.err.Error()
			m.logger.Info("configuration", fmt.Sprintf("Error fetching package names: %s", msg.err))
		}
		m.setupInputs()
		return m, nil

	case tea.KeyMsg:
		// Don't allow navigation while fetching package names
		if m.fetchingNames {
			return m, nil
		}
		return m.handleKeys(msg)

	default:
		// Update current input
		if !m.fetchingNames && m.currentRepo < len(m.shared.SelectedDependencies) && len(m.inputs) > 0 {
			inputIndex := m.currentRepo*3 + m.currentField
			if inputIndex >= 0 && inputIndex < len(m.inputs) {
				var cmd tea.Cmd
				m.inputs[inputIndex], cmd = m.inputs[inputIndex].Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the configuration wizard interface.
//
// Display modes:
//   - No packages selected: Shows error message
//   - Configuring: Shows current package (N/Total), three input fields,
//     and progress indicator
//   - All configured: Shows success message and instruction to continue
//
// The currently active field is highlighted with the selectedStyle (purple bg).
// Help text changes based on state (configuring vs. complete).
func (m *ConfigurationModel) View() string {
	if len(m.shared.SelectedDependencies) == 0 {
		return m.headerStyle.Render("❌ No Repositories Selected") + "\n\nPlease go back and select repositories first.\n\nPress Q to return to main menu"
	}

	// Show loading message while fetching package names
	if m.fetchingNames {
		return m.headerStyle.Render("🔧 Package Configuration") + "\n\n" +
			"⏳ Fetching actual package names from repositories...\n\n" +
			m.helpStyle.Render("This ensures the correct package names are used for dart pub add")
	}

	var b strings.Builder

	// Header
	b.WriteString(m.headerStyle.Render("🔧 Package Configuration") + "\n")
	b.WriteString(fmt.Sprintf("Configure %d selected packages:\n\n", len(m.shared.SelectedDependencies)))

	// Show current repository being configured
	if m.currentRepo < len(m.shared.SelectedDependencies) {
		repo := m.shared.SelectedDependencies[m.currentRepo]
		b.WriteString(fmt.Sprintf("📦 Configuring: %s/%s\n\n", repo.Owner, repo.Name))

		// Show input fields
		fields := []string{"Package Name (read-only):", "Git Ref (branch/tag):", "Subdirectory:"}
		for i, field := range fields {
			if i == m.currentField {
				b.WriteString(m.selectedStyle.Render(field) + "\n")
			} else {
				b.WriteString(m.normalStyle.Render(field) + "\n")
			}

			inputIndex := m.currentRepo*3 + i
			if inputIndex < len(m.inputs) {
				b.WriteString(m.inputs[inputIndex].View() + "\n\n")
			}
		}

		// Progress
		b.WriteString(fmt.Sprintf("Progress: %d/%d packages configured\n\n", m.currentRepo+1, len(m.shared.SelectedDependencies)))
	} else {
		b.WriteString(m.headerStyle.Render("✅ All Packages Configured") + "\n\n")
		b.WriteString("Press Enter to continue to confirmation\n\n")
	}

	// Help
	if m.currentRepo < len(m.shared.SelectedDependencies) {
		b.WriteString(m.helpStyle.Render("tab: next field • shift+tab: prev field • enter: next package • q: back"))
	} else {
		b.WriteString(m.helpStyle.Render("enter: continue • q: back"))
	}

	return b.String()
}

// handleKeys processes keyboard navigation and input.
//
// Keyboard shortcuts:
//   - Tab: Move to next field (name → ref → subdir → name)
//   - Shift+Tab: Move to previous field (reverse of Tab)
//   - Enter: Save current package and move to next, or proceed to confirmation
//   - Q/Ctrl+C: Return to main menu (abandons configuration)
//
// All other keys are forwarded to the active text input for typing.
func (m *ConfigurationModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, TransitionToScreen(ScreenMainMenu)

	case "tab":
		if m.currentRepo < len(m.shared.SelectedDependencies) {
			m.currentField++
			if m.currentField >= 3 {
				m.currentField = 1 // Skip field 0 (name is read-only), go to field 1 (ref)
			}
			m.focusCurrentInput()
		}
		return m, nil

	case "shift+tab":
		if m.currentRepo < len(m.shared.SelectedDependencies) {
			m.currentField--
			if m.currentField < 1 { // Skip field 0 (name is read-only)
				m.currentField = 2 // Go to field 2 (subdir)
			}
			m.focusCurrentInput()
		}
		return m, nil

	case "enter":
		if m.currentRepo >= len(m.shared.SelectedDependencies) {
			// All configured, move to confirmation
			m.generatePackageSpecs()
			return m, TransitionToScreen(ScreenConfirmation)
		} else {
			// Move to next repository
			m.currentRepo++
			m.currentField = 1 // Start at field 1 (ref) since field 0 (name) is read-only
			m.focusCurrentInput()
		}
		return m, nil

	default:
		// Pass to current input
		if m.currentRepo < len(m.shared.SelectedDependencies) {
			var cmd tea.Cmd
			inputIndex := m.currentRepo*3 + m.currentField
			if inputIndex < len(m.inputs) {
				m.inputs[inputIndex], cmd = m.inputs[inputIndex].Update(msg)
			}
			return m, cmd
		}
	}

	return m, nil
}

// setupInputs creates a text input for each configuration field of each package.
// Creates exactly 3 * len(SelectedDependencies) inputs in a flat array.
//
// For each package, creates:
//  1. Name input: Pre-filled with repository name, width 40
//  2. Ref input: Pre-filled with "main", width 40
//  3. Subdir input: Empty with "(optional)" placeholder, width 40
//
// The first input (name of first package) is automatically focused.
// If no packages are selected, marks the screen as complete (no-op state).
func (m *ConfigurationModel) setupInputs() {
	// Safety check - ensure we have selected repositories
	if len(m.shared.SelectedDependencies) == 0 {
		m.logger.Debug("configuration", "No repositories selected for configuration")
		m.complete = true // Mark as complete to skip configuration
		return
	}

	// Create 3 inputs per repository (name, ref, subdir)
	totalInputs := len(m.shared.SelectedDependencies) * 3
	m.inputs = make([]textinput.Model, totalInputs)

	for i, repo := range m.shared.SelectedDependencies {
		// Package name input - use actual package name if available, otherwise use repo name
		// This field is read-only because the package name is fetched from pubspec.yaml
		// and cannot be changed (dart pub add requires exact match with pubspec.yaml)
		packageName := repo.PackageName
		if packageName == "" {
			packageName = repo.Name
		}

		nameInput := textinput.New()
		nameInput.Placeholder = packageName
		nameInput.SetValue(packageName)
		nameInput.Width = 40
		// Make the name input read-only by disabling cursor and text entry
		nameInput.Blur()
		nameInput.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // Dimmed gray
		m.inputs[i*3] = nameInput

		// Ref input
		refInput := textinput.New()
		refInput.Placeholder = "main"
		refInput.SetValue("main")
		refInput.Width = 40
		m.inputs[i*3+1] = refInput

		// Subdir input
		subdirInput := textinput.New()
		subdirInput.Placeholder = "(optional)"
		subdirInput.Width = 40
		m.inputs[i*3+2] = subdirInput
	}

	m.focusCurrentInput()
}

// focusCurrentInput updates focus state of all inputs.
// Blurs all inputs, then focuses the one at [currentRepo*3 + currentField].
//
// Performs bounds checking and logs debug messages if indices are invalid.
// This prevents panics from out-of-bounds access during edge cases.
func (m *ConfigurationModel) focusCurrentInput() {
	// Safety check - ensure we have inputs
	if len(m.inputs) == 0 {
		m.logger.Debug("configuration", "No inputs available to focus")
		return
	}

	// Blur all inputs
	for i := range m.inputs {
		m.inputs[i].Blur()
	}

	// Focus current input
	if m.currentRepo < len(m.shared.SelectedDependencies) {
		inputIndex := m.currentRepo*3 + m.currentField
		if inputIndex >= 0 && inputIndex < len(m.inputs) {
			m.inputs[inputIndex].Focus()
		} else {
			m.logger.Debug("configuration", fmt.Sprintf("Invalid input index: %d (total: %d)", inputIndex, len(m.inputs)))
		}
	}
}

// generatePackageSpecs converts user input into core.PkgSpec structs.
// Called when all packages have been configured and user presses Enter to proceed.
//
// For each package:
//   - Reads name, ref, and subdir from their respective inputs
//   - Uses defaults if fields are empty (repo name, "main", "")
//   - Combines with repository URL from shared state
//   - Creates a core.PkgSpec ready for dart/flutter pub add
//
// The generated specs are stored in both the model and shared state for access
// by the confirmation and execution screens. Performs defensive bounds checking.
func (m *ConfigurationModel) generatePackageSpecs() {
	// Safety check - ensure we have selected repositories
	if len(m.shared.SelectedDependencies) == 0 {
		m.logger.Debug("configuration", "No repositories to generate package specs for")
		return
	}

	m.packageSpecs = make([]core.PkgSpec, len(m.shared.SelectedDependencies))

	for i, repo := range m.shared.SelectedDependencies {
		// Safety check for input array bounds
		if i*3+2 >= len(m.inputs) {
			m.logger.Debug("configuration", fmt.Sprintf("Insufficient inputs for repo %d", i))
			// Create default spec using pre-fetched package name
			packageName := repo.PackageName
			if packageName == "" {
				packageName = repo.Name
			}
			m.packageSpecs[i] = core.PkgSpec{
				Name:   packageName,
				URL:    repo.URL,
				Ref:    "main",
				Subdir: "",
			}
			continue
		}

		// Use pre-fetched package name from repo (field 0 is read-only)
		packageName := repo.PackageName
		if packageName == "" {
			packageName = repo.Name
		}

		ref := m.inputs[i*3+1].Value()
		if ref == "" {
			ref = "main"
		}

		subdir := m.inputs[i*3+2].Value()

		m.packageSpecs[i] = core.PkgSpec{
			Name:   packageName,
			URL:    repo.URL,
			Ref:    ref,
			Subdir: subdir,
		}
	}

	m.shared.PackageSpecs = m.packageSpecs
	m.logger.Info("configuration", fmt.Sprintf("Generated %d package specifications", len(m.packageSpecs)))
}

// fetchPackageNames fetches the actual package names from git repositories asynchronously
// This prevents the UI from showing incorrect package names (repo name vs actual package name)
func (m *ConfigurationModel) fetchPackageNames() tea.Cmd {
	return func() tea.Msg {
		m.logger.Info("configuration", "Fetching actual package names from repositories...")

		// Fetch package name for each selected dependency
		for i := range m.shared.SelectedDependencies {
			repo := &m.shared.SelectedDependencies[i]

			// Skip if package name is already set
			if repo.PackageName != "" {
				m.logger.Info("configuration", fmt.Sprintf("Package name already set for %s: %s", repo.Name, repo.PackageName))
				continue
			}

			// Fetch the actual package name from pubspec.yaml
			packageName, err := core.FetchPackageNameFromGit(m.logger, repo.URL, "main", "")
			if err != nil {
				m.logger.Info("configuration", fmt.Sprintf("Failed to fetch package name for %s: %s (will use repo name)", repo.Name, err))
				// Fallback to repo name - don't fail the entire operation
				repo.PackageName = repo.Name
				continue
			}

			m.logger.Info("configuration", fmt.Sprintf("Fetched package name for %s: %s", repo.Name, packageName))
			repo.PackageName = packageName
		}

		return packageNamesFetchedMsg{err: nil}
	}
}
