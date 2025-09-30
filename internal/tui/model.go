package tui

import (
	"fmt"
	"strings"

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
	width  int
	height int
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return stepMsg{step: core.StepDetectProject}
	})
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
		m.projects = msg.projects
		if len(m.projects) == 1 {
			m.selectedProject = 0
			m.step = core.StepChooseSource
			return m, m.getStepCommand()
		}
		return m, nil

	case reposMsg:
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
	case core.StepDetectProject:
		return m.handleDetectProjectKeys(msg)
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
func (m Model) handleDetectProjectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedProject > 0 {
			m.selectedProject--
		}
	case "down", "j":
		if m.selectedProject < len(m.projects)-1 {
			m.selectedProject++
		}
	case "enter":
		if len(m.projects) > 0 {
			m.step = core.StepChooseSource
			return m, m.getStepCommand()
		}
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
	case core.StepDetectProject:
		return tea.Cmd(m.detectProjects)
	case core.StepChooseSource:
		return nil // UI only
	case core.StepListRepos:
		return tea.Cmd(m.listRepos)
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
func (m Model) detectProjects() tea.Msg {
	// Try to find nearest pubspec first
	if project, err := core.NearestPubspec(""); err == nil {
		return projectsMsg{projects: []core.Project{*project}}
	}

	// Fall back to scanning common roots
	projects, err := core.ScanCommonRoots()
	if err != nil {
		return errorMsg{err: err}
	}

	return projectsMsg{projects: projects}
}

func (m Model) listRepos() tea.Msg {
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

	// Execute each job
	for _, spec := range m.edits {
		result := core.AddGitDependency(m.logger, &m.cfg, project.Path, spec)
		results = append(results, result)

		if !result.OK {
			break // Stop on first error
		}
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

// View implements tea.Model
func (m Model) View() string {
	var b strings.Builder

	// Header
	header := headerStyle.Render("🎯 Flutter Package Manager")
	b.WriteString(header + "\\n\\n")

	// Step indicator
	stepText := m.getStepText()
	b.WriteString(stepStyle.Render(stepText) + "\\n\\n")

	// Main content based on current step
	content := m.getStepView()
	b.WriteString(content)

	// Footer with help
	footer := m.getFooter()
	b.WriteString("\\n\\n" + helpStyle.Render(footer))

	return b.String()
}

func (m Model) getStepText() string {
	switch m.step {
	case core.StepDetectProject:
		return "Step 1/6: 🔍 Detect Project"
	case core.StepChooseSource:
		return "Step 2/6: 📂 Choose Source"
	case core.StepListRepos:
		return "Step 3/6: 📋 List Repositories"
	case core.StepEditSpecs:
		return "Step 4/6: ✏️ Edit Specifications"
	case core.StepConfirm:
		return "Step 5/6: ✅ Confirm"
	case core.StepExecute:
		return "Step 6/6: ⚡ Execute"
	case core.StepSummary:
		return "✨ Summary & Recommendations"
	}
	return "Unknown Step"
}

func (m Model) getStepView() string {
	switch m.step {
	case core.StepDetectProject:
		return m.viewDetectProject()
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
	case core.StepDetectProject:
		return "↑/↓ navigate • enter select • q quit"
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

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#02569B")).
			Padding(1, 2).
			Bold(true)

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
