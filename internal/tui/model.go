package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// Model represents the TUI application state following Junie's plan
type Model struct {
	step   core.Step
	cfg    core.Config
	logger *core.Logger
	msgs   []string
	err    error

	// Discovery
	projects        []core.Project
	selectedProject int
	loading         bool
	loadingText     string

	// Source selection
	source core.SourceMode
	repos  []core.RepoCandidate
	picks  map[int]bool
	cursor int

	// Per-pick package spec editing
	editIdx int
	edits   []core.PkgSpec

	// Run queue + progress
	jobs    []core.PkgSpec
	results []core.ActionResult

	// Recommendations
	recos []core.Reco

	// UI state
	width       int
	height      int
	spinnerIdx  int
	lastSpinner time.Time
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.Cmd(func() tea.Msg {
			return stepMsg{step: core.StepMainMenu}
		}),
		tea.Cmd(m.detectProjectQuick), // Quick, non-blocking detection
		m.tickSpinnerOptimized(),      // Optimized spinner animation
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case stepMsg:
		m.step = msg.step
		return m, m.getStepCommand()

	case projectsMsg:
		m.loading = false
		oldProjectCount := len(m.projects)
		m.projects = msg.projects

		// If we're on the main menu and just did a quick detection, stay on main menu
		if m.step == core.StepMainMenu && oldProjectCount == 0 {
			return m, nil
		}

		// If we were scanning directories (option 1), show results or go to source selection
		if len(m.projects) == 1 {
			m.selectedProject = 0
			m.step = core.StepChooseSource
			return m, m.getStepCommand()
		} else if len(m.projects) == 0 {
			m.err = fmt.Errorf("no Flutter projects found. Create a new Flutter project with 'flutter create' or navigate to an existing one")
		}
		return m, nil

	case loadingMsg:
		m.loading = true
		m.loadingText = msg.text
		return m, nil

	case tickMsg:
		// Only update spinner if enough time has passed (throttle for performance)
		now := time.Now()
		if now.Sub(m.lastSpinner) >= 100*time.Millisecond {
			m.spinnerIdx = (m.spinnerIdx + 1) % 10 // 10 spinner frames
			m.lastSpinner = now
		}
		if m.loading {
			return m, m.tickSpinnerOptimized()
		}
		return m, nil

	case reposMsg:
		m.loading = false
		m.repos = msg.repos
		m.picks = make(map[int]bool)
		return m, nil

	case specsMsg:
		m.edits = msg.specs
		m.step = core.StepConfirm
		return m, nil

	case resultsMsg:
		m.results = msg.results
		m.step = core.StepSummary
		return m, m.getStepCommand()

	case recosMsg:
		m.recos = msg.recos
		return m, nil

	case errorMsg:
		m.err = msg.err
		return m, nil

	case messageMsg:
		m.msgs = append(m.msgs, msg.message)
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input based on current step
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	switch m.step {
	case core.StepMainMenu:
		return m.handleMainMenuKeys(msg)
	case core.StepChooseSource:
		return m.handleChooseSourceKeys(msg)
	case core.StepListRepos:
		return m.handleListReposKeys(msg)
	case core.StepEditSpecs:
		return m.handleEditSpecsKeys(msg)
	case core.StepConfirm:
		return m.handleConfirmKeys(msg)
	case core.StepExecute:
		return m.handleExecuteKeys(msg)
	case core.StepSummary:
		return m.handleSummaryKeys(msg)
	}

	return m, nil
}

// Step-specific key handlers
func (m Model) handleMainMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		// Navigate through available options (skip unavailable ones)
		maxOptions := 6 // Total menu options (1-6)
		if m.cursor < maxOptions-1 {
			m.cursor++
		}
	case "enter", " ":
		return m.handleMenuSelection()
	// Direct option selection matching shell script exactly
	case "1":
		m.cursor = 0 // Scan directories
		return m.handleMenuSelection()
	case "2":
		m.cursor = 1 // GitHub repo
		return m.handleMenuSelection()
	case "3":
		m.cursor = 2 // Configure search
		return m.handleMenuSelection()
	case "4":
		if len(m.projects) > 0 {
			m.cursor = 3 // Use detected project
			return m.handleMenuSelection()
		}
	case "5":
		if len(m.projects) > 0 {
			m.cursor = 4 // Express Git update
			return m.handleMenuSelection()
		}
	case "6":
		m.cursor = 5 // Check for Flutter-PM updates
		return m.handleMenuSelection()
	}
	return m, nil
}

func (m Model) handleMenuSelection() (tea.Model, tea.Cmd) {
	// Handle selection based on SHELL SCRIPT menu structure (1-6)
	switch m.cursor {
	case 0: // "1. Scan directories" - scan configured directories for projects
		return m, tea.Batch(
			m.startLoading("Scanning directories for Flutter projects..."),
			tea.Cmd(m.detectProjectsAsync),
		)
	case 1: // "2. GitHub repo" - use GitHub repositories as source for packages
		m.step = core.StepListRepos
		m.source = core.SourceGitHub
		return m, m.getStepCommand()
	case 2: // "3. Configure search" - configure search settings
		// TODO: Implement configuration
		m.err = fmt.Errorf("configuration not implemented yet")
		return m, nil
	case 3: // "4. Use detected project" - continue with locally found project as source [DEFAULT]
		if len(m.projects) > 0 {
			m.selectedProject = 0
			m.step = core.StepChooseSource
			return m, m.getStepCommand()
		}
	case 4: // "5. 🚀 Express Git update" - update existing git dependencies for local project
		if len(m.projects) > 0 {
			return m, tea.Batch(
				m.startLoading("🚀 Express Git update - updating dependencies..."),
				tea.Cmd(m.expressGitUpdate),
			)
		}
	case 5: // "6. 🔄 Check for Flutter-PM updates" - update Flutter-PM itself
		return m, tea.Batch(
			m.startLoading("🔄 Checking for Flutter-PM updates..."),
			tea.Cmd(m.checkSelfUpdate),
		)
	}
	return m, nil
}

func (m Model) handleChooseSourceKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 2 { // 3 source modes
			m.cursor++
		}
	case "enter":
		m.source = core.SourceMode(m.cursor)
		m.step = core.StepListRepos
		return m, m.getStepCommand()
	}
	return m, nil
}

func (m Model) handleListReposKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.repos)-1 {
			m.cursor++
		}
	case " ": // Space to toggle
		if m.picks == nil {
			m.picks = make(map[int]bool)
		}
		m.picks[m.cursor] = !m.picks[m.cursor]
	case "enter":
		// Move to edit specs for selected repos
		m.step = core.StepEditSpecs
		return m, m.getStepCommand()
	}
	return m, nil
}

func (m Model) handleEditSpecsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.step = core.StepConfirm
		return m, nil
	}
	return m, nil
}

func (m Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.step = core.StepExecute
		return m, m.getStepCommand()
	case "n":
		m.step = core.StepEditSpecs
		return m, nil
	}
	return m, nil
}

func (m Model) handleExecuteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// No input during execution
	return m, nil
}

func (m Model) handleSummaryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "r":
		// Restart
		return m, tea.Cmd(func() tea.Msg {
			return stepMsg{step: core.StepDetectProject}
		})
	}
	return m, nil
}

// getStepCommand returns the command to execute for the current step
func (m Model) getStepCommand() tea.Cmd {
	switch m.step {
	case core.StepMainMenu:
		return nil // Main menu is UI only
	case core.StepChooseSource:
		return nil // UI only
	case core.StepListRepos:
		return tea.Batch(
			m.startLoading("Fetching repositories..."),
			tea.Cmd(m.listReposAsync),
		)
	case core.StepEditSpecs:
		return tea.Cmd(m.editSpecs)
	case core.StepExecute:
		return tea.Cmd(m.executeJobs)
	case core.StepSummary:
		return tea.Cmd(m.generateRecommendations)
	}
	return nil
}

// Commands
func (m Model) startLoading(text string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return loadingMsg{text: text}
	})
}

func (m Model) tickSpinner() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// tickSpinnerOptimized provides more efficient spinner animation
func (m Model) tickSpinnerOptimized() tea.Cmd {
	// Use 150ms interval instead of 100ms to reduce CPU usage while maintaining smooth animation
	return tea.Tick(time.Millisecond*150, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) detectProjectQuick() tea.Msg {
	// Only try to find nearest pubspec (quick detection like PowerShell)
	if project, err := core.NearestPubspec(""); err == nil {
		return projectsMsg{projects: []core.Project{*project}}
	}
	// If no local project found, that's fine - just show empty projects
	return projectsMsg{projects: []core.Project{}}
}

func (m Model) detectProjectsAsync() tea.Msg {
	// Fall back to full scanning when explicitly requested
	projects, err := core.ScanCommonRoots()
	if err != nil {
		return errorMsg{err: err}
	}
	return projectsMsg{projects: projects}
}

// Legacy function for compatibility
func (m Model) detectProjects() tea.Msg {
	return m.detectProjectsAsync()
}

func (m Model) listReposAsync() tea.Msg {
	switch m.source {
	case core.SourceGitHub:
		repos, err := core.ListGitHubRepos(m.logger)
		if err != nil {
			return errorMsg{err: err}
		}
		return reposMsg{repos: repos}
	case core.SourceLocalScan, core.SourceManualURL:
		// These would need additional implementation
		return errorMsg{err: fmt.Errorf("source mode %d not implemented yet", m.source)}
	}
	return nil
}

// Legacy function for compatibility
func (m Model) listRepos() tea.Msg {
	return m.listReposAsync()
}

func (m Model) editSpecs() tea.Msg {
	var specs []core.PkgSpec

	for i, selected := range m.picks {
		if selected && i < len(m.repos) {
			repo := m.repos[i]
			spec := core.PkgSpec{
				Name: strings.ReplaceAll(repo.Name, "-", "_"), // Dart-safe name
				URL:  repo.URL,
				Ref:  "main", // Default
			}
			specs = append(specs, spec)
		}
	}

	return specsMsg{specs: specs}
}

func (m Model) executeJobs() tea.Msg {
	if len(m.projects) == 0 {
		return errorMsg{err: fmt.Errorf("no project selected")}
	}

	project := m.projects[m.selectedProject]
	var results []core.ActionResult

	// Create backup first
	if backupInfo, err := core.CreateBackup(project.Path); err != nil {
		m.logger.Error("backup", err)
	} else {
		m.logger.Info("backup", fmt.Sprintf("Created backup: %s", backupInfo.BackupPath))
	}

	// Use batch operation for better performance when adding multiple dependencies
	if len(m.edits) > 1 {
		m.logger.Info("execute", fmt.Sprintf("Batch adding %d dependencies", len(m.edits)))
		batchResult := core.AddGitDependenciesBatch(m.logger, &m.cfg, project.Path, m.edits)
		results = append(results, batchResult)
	} else if len(m.edits) == 1 {
		// Single dependency - use individual method
		result := core.AddGitDependency(m.logger, &m.cfg, project.Path, m.edits[0])
		results = append(results, result)
	}

	// Run pub get if all succeeded
	if len(results) > 0 && results[len(results)-1].OK {
		syncResult := core.Sync(m.logger, &m.cfg, project.Path)
		results = append(results, syncResult)
	}

	return resultsMsg{results: results}
}

func (m Model) generateRecommendations() tea.Msg {
	if len(m.projects) == 0 {
		return nil
	}

	project := m.projects[m.selectedProject]
	recos, err := core.GenerateFullRecommendations(m.logger, project.Path)
	if err != nil {
		return errorMsg{err: err}
	}

	return recosMsg{recos: recos}
}

// expressGitUpdate performs express git update for existing git dependencies
func (m Model) expressGitUpdate() tea.Msg {
	if len(m.projects) == 0 {
		return errorMsg{err: fmt.Errorf("no project selected")}
	}

	project := m.projects[m.selectedProject]
	result := core.ExpressGitUpdate(m.logger, &m.cfg, project.Path)

	// Return result as a single-item results list
	return resultsMsg{results: []core.ActionResult{result}}
}

// checkSelfUpdate checks for Flutter-PM updates
func (m Model) checkSelfUpdate() tea.Msg {
	result := core.CheckSelfUpdate(m.logger, &m.cfg)

	// Return result as a single-item results list
	return resultsMsg{results: []core.ActionResult{result}}
}

// nuclearCacheUpdate performs nuclear cache clearing + update (remove pubspec.lock + clear pub cache)
func (m Model) nuclearCacheUpdate() tea.Msg {
	if len(m.projects) == 0 {
		return errorMsg{err: fmt.Errorf("no project selected")}
	}

	project := m.projects[m.selectedProject]
	result := core.NuclearCacheUpdate(m.logger, &m.cfg, project.Path)

	// Return result as a single-item results list
	return resultsMsg{results: []core.ActionResult{result}}
}

// View implements tea.Model
func (m Model) View() string {
	var b strings.Builder

	// Header with proper spacing
	header := headerStyle.Render("🎯 Flutter Package Manager")
	b.WriteString(header + "\n\n")

	// Step indicator (only show for non-main menu)
	if m.step != core.StepMainMenu {
		stepText := m.getStepText()
		b.WriteString(stepStyle.Render(stepText) + "\n\n")
	}

	// Main content based on current step
	content := m.getStepView()
	b.WriteString(content)

	// Footer with help
	footer := m.getFooter()
	b.WriteString("\n\n" + helpStyle.Render(footer))

	return b.String()
}

func (m Model) getStepText() string {
	switch m.step {
	case core.StepMainMenu:
		return "🎯 Flutter Package Manager - Main Menu"
	case core.StepChooseSource:
		return "Step 1/5: 📂 Choose Source"
	case core.StepListRepos:
		return "Step 2/5: 📋 List Repositories"
	case core.StepEditSpecs:
		return "Step 3/5: ✏️ Edit Specifications"
	case core.StepConfirm:
		return "Step 4/5: ✅ Confirm"
	case core.StepExecute:
		return "Step 5/5: ⚡ Execute"
	case core.StepSummary:
		return "✨ Summary & Recommendations"
	}
	return "Unknown Step"
}

func (m Model) getStepView() string {
	switch m.step {
	case core.StepMainMenu:
		return m.viewMainMenu()
	case core.StepChooseSource:
		return m.viewChooseSource()
	case core.StepListRepos:
		return m.viewListRepos()
	case core.StepEditSpecs:
		return m.viewEditSpecs()
	case core.StepConfirm:
		return m.viewConfirm()
	case core.StepExecute:
		return m.viewExecute()
	case core.StepSummary:
		return m.viewSummary()
	}
	return "Unknown step"
}

func (m Model) getFooter() string {
	switch m.step {
	case core.StepMainMenu:
		if len(m.projects) > 0 {
			return "↑/↓ navigate • enter/1-6 select • q quit"
		} else {
			return "↑/↓ navigate • enter/1-3 select • q quit"
		}
	case core.StepChooseSource:
		return "↑/↓ navigate • enter select • q quit"
	case core.StepListRepos:
		return "↑/↓ navigate • space toggle • enter confirm • q quit"
	case core.StepEditSpecs:
		return "enter continue • q quit"
	case core.StepConfirm:
		return "y/enter confirm • n back • q quit"
	case core.StepExecute:
		return "please wait..."
	case core.StepSummary:
		return "r restart • enter restart • q quit"
	}
	return "q quit"
}

// Message types
type stepMsg struct {
	step core.Step
}

type projectsMsg struct {
	projects []core.Project
}

type reposMsg struct {
	repos []core.RepoCandidate
}

type specsMsg struct {
	specs []core.PkgSpec
}

type resultsMsg struct {
	results []core.ActionResult
}

type recosMsg struct {
	recos []core.Reco
}

type errorMsg struct {
	err error
}

type messageMsg struct {
	message string
}

type loadingMsg struct {
	text string
}

type tickMsg struct{}

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#02569B")).
			Padding(1, 2).
			Bold(true).
			Width(60).
			Align(lipgloss.Center)

	stepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#13B9FD")).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#02569B")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)
)

// Run starts the TUI application
func Run(cfg core.Config, logger *core.Logger) error {
	m := Model{
		step:   core.StepDetectProject,
		cfg:    cfg,
		logger: logger,
		cursor: 0,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
