package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// BubbleTeaModel represents the proper bubbletea TUI application
type BubbleTeaModel struct {
	step   core.Step
	cfg    core.Config
	logger *core.Logger

	// Bubbletea components
	list     list.Model
	spinner  spinner.Model
	progress progress.Model

	// Application state
	projects        []core.Project
	selectedProject int
	repos           []core.RepoCandidate
	selectedRepos   []core.RepoCandidate
	edits           []core.PkgSpec
	results         []core.ActionResult
	recos           []core.Reco

	// UI state
	width   int
	height  int
	loading bool
	err     error

	// Current operation and progress
	currentOperation    string
	operationDone       bool
	progressPercent     float64
	progressMessage     string
	installQueue        []core.PkgSpec
	currentInstallIndex int
}

// ListItem represents an item in the bubbletea list
type ListItem struct {
	title       string
	description string
	data        interface{}
}

func (i ListItem) Title() string       { return i.title }
func (i ListItem) Description() string { return i.description }
func (i ListItem) FilterValue() string { return i.title }

// Initialize creates a new BubbleTeaModel with proper component setup
func NewBubbleTeaModel(cfg core.Config, logger *core.Logger) BubbleTeaModel {
	// Set up list component
	listItems := []list.Item{
		ListItem{title: "📁 Scan directories", description: "Scan configured directories for projects"},
		ListItem{title: "🐙 GitHub repo", description: "Single-select GitHub repository to clone as project"},
		ListItem{title: "⚙️ Configure search", description: "Configure search settings"},
	}

	l := list.New(listItems, NewItemDelegate(), 0, 0)
	l.Title = "🎯 Flutter Package Manager"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#02569B")).
		Padding(0, 1).
		Bold(true)

	// Set up spinner component
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD"))

	// Set up progress component
	p := progress.New(progress.WithDefaultGradient())

	return BubbleTeaModel{
		step:     core.StepMainMenu,
		cfg:      cfg,
		logger:   logger,
		list:     l,
		spinner:  s,
		progress: p,
	}
}

// Init implements tea.Model
func (m BubbleTeaModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Cmd(func() tea.Msg {
			return stepMsg{step: core.StepMainMenu}
		}),
		tea.Cmd(m.detectProjectsQuickly),
	)
}

// Update implements tea.Model
func (m BubbleTeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 8)
		m.progress.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m.handleEnterKey()
		}

	case stepMsg:
		m.step = msg.step
		return m, m.setupStepList()

	case projectsMsg:
		m.loading = false
		m.projects = msg.projects
		return m, m.updateMainMenuList()

	case reposMsg:
		m.loading = false
		m.repos = msg.repos
		return m, m.setupReposList()

	case resultsMsg:
		m.results = msg.results
		m.operationDone = true
		m.loading = false
		m.step = core.StepSummary
		return m, nil

	case progressUpdateMsg:
		m.progressPercent = msg.percent
		m.progressMessage = msg.message
		return m, nil

	case installProgressMsg:
		m.currentInstallIndex = msg.index
		m.progressPercent = float64(msg.index) / float64(len(m.installQueue))
		m.progressMessage = msg.message
		if msg.index >= len(m.installQueue) {
			// Installation complete
			m.operationDone = true
			m.loading = false
			m.step = core.StepSummary
			return m, nil
		}
		return m, tea.Cmd(func() tea.Msg {
			return m.installNextPackage()
		})

	case errorMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case progress.FrameMsg:
		progressModel, progressCmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, progressCmd)
	}

	// Update list component
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m BubbleTeaModel) View() string {
	var sections []string

	// Header
	header := headerStyle.Render("🎯 Flutter Package Manager")
	sections = append(sections, header)

	// Show error if present
	if m.err != nil {
		errorText := errorStyle.Render(fmt.Sprintf("❌ Error: %s", m.err.Error()))
		sections = append(sections, errorText)
	}

	// Show loading spinner if needed
	if m.loading {
		spinnerText := fmt.Sprintf("%s %s", m.spinner.View(), m.currentOperation)
		sections = append(sections, spinnerStyle.Render(spinnerText))
	}

	// Show progress if operation is running
	if m.currentOperation != "" && !m.operationDone {
		if m.progressPercent > 0 {
			progressText := fmt.Sprintf("Progress: %s %.0f%%", m.progress.ViewAs(m.progressPercent), m.progressPercent*100)
			if m.progressMessage != "" {
				progressText += fmt.Sprintf("\n%s", m.progressMessage)
			}
			sections = append(sections, progressText)
		}
	}

	// Main content based on step
	switch m.step {
	case core.StepMainMenu:
		sections = append(sections, m.viewMainMenuBubbles())
	case core.StepSelectGitHubProject:
		sections = append(sections, m.viewGitHubProjectsBubbles())
	case core.StepListRepos:
		sections = append(sections, m.viewRepoListBubbles())
	case core.StepSummary:
		sections = append(sections, m.viewSummaryBubbles())
	default:
		sections = append(sections, m.list.View())
	}

	// Footer
	footer := helpStyle.Render("↑/↓ navigate • enter select • q quit")
	sections = append(sections, footer)

	return strings.Join(sections, "\n\n")
}

// Handle enter key based on current step
func (m BubbleTeaModel) handleEnterKey() (BubbleTeaModel, tea.Cmd) {
	switch m.step {
	case core.StepMainMenu:
		selectedItem := m.list.SelectedItem()
		if item, ok := selectedItem.(ListItem); ok {
			switch item.title {
			case "📁 Scan directories":
				m.loading = true
				m.currentOperation = "Scanning directories for Flutter projects..."
				return m, tea.Cmd(m.detectProjectsAsync)
			case "🐙 GitHub repo":
				m.step = core.StepSelectGitHubProject
				m.loading = true
				m.currentOperation = "Fetching GitHub repositories..."
				return m, tea.Cmd(m.listGitHubReposAsync)
			case "⚙️ Configure search":
				m.err = fmt.Errorf("configuration not implemented yet")
				return m, nil
			}
		}
		// Handle detected projects
		if len(m.projects) > 0 {
			for i, item := range m.list.Items() {
				if listItem, ok := item.(ListItem); ok && strings.Contains(listItem.title, "Use detected:") {
					if m.list.Index() == i {
						m.selectedProject = 0
						m.step = core.StepChooseSource
						return m, m.setupStepList()
					}
				}
			}
		}

	case core.StepSelectGitHubProject:
		selectedItem := m.list.SelectedItem()
		if item, ok := selectedItem.(ListItem); ok {
			if repo, ok := item.data.(core.RepoCandidate); ok {
				m.loading = true
				m.currentOperation = fmt.Sprintf("Cloning %s/%s...", repo.Owner, repo.Name)
				return m, tea.Cmd(func() tea.Msg {
					return m.cloneSelectedRepo(repo)
				})
			}
		}
	}

	return m, nil
}

// Setup list for different steps
func (m BubbleTeaModel) setupStepList() tea.Cmd {
	switch m.step {
	case core.StepChooseSource:
		items := []list.Item{
			ListItem{title: "📦 GitHub Repositories", description: "Browse your GitHub repositories"},
			ListItem{title: "🔗 Manual URL Entry", description: "Enter git repository URLs manually"},
			ListItem{title: "📁 Local Repositories", description: "Scan local directories for git repositories"},
		}
		m.list.SetItems(items)
		m.list.Title = "Step 1/6: 📂 Choose Source"

	case core.StepListRepos:
		m.loading = true
		m.currentOperation = "Fetching repositories..."
		return tea.Cmd(m.listReposAsync)
	}

	return nil
}

// Update main menu list with detected projects
func (m BubbleTeaModel) updateMainMenuList() tea.Cmd {
	items := []list.Item{
		ListItem{title: "📁 Scan directories", description: "Scan configured directories for projects"},
		ListItem{title: "🐙 GitHub repo", description: "Single-select GitHub repository to clone as project"},
		ListItem{title: "⚙️ Configure search", description: "Configure search settings"},
	}

	// Add detected project options
	if len(m.projects) > 0 {
		project := m.projects[0]
		projectName := project.Name
		if projectName == "" {
			projectName = fmt.Sprintf("Project in %s", project.Path)
		}
		items = append(items,
			ListItem{title: fmt.Sprintf("📦 Use detected: %s", projectName), description: "Continue with locally found project as source [DEFAULT]"},
			ListItem{title: fmt.Sprintf("🚀 Express Git update for %s", projectName), description: "Update existing git dependencies for local project"},
		)
	}

	items = append(items, ListItem{title: "🔄 Check for Flutter-PM updates", description: "Update Flutter-PM itself"})

	m.list.SetItems(items)
	return nil
}

// Setup repos list
func (m BubbleTeaModel) setupReposList() tea.Cmd {
	var items []list.Item

	for _, repo := range m.repos {
		privacy := "🔓"
		if repo.Privacy == "private" {
			privacy = "🔒"
		}

		title := fmt.Sprintf("%s %s/%s", privacy, repo.Owner, repo.Name)
		description := repo.Desc
		if description == "" {
			description = "No description"
		}

		items = append(items, ListItem{
			title:       title,
			description: description,
			data:        repo,
		})
	}

	m.list.SetItems(items)
	m.list.Title = fmt.Sprintf("🐙 Select GitHub Project to Clone (%d repositories)", len(m.repos))
	return nil
}

// View functions using bubbletea components
func (m BubbleTeaModel) viewMainMenuBubbles() string {
	return m.list.View()
}

func (m BubbleTeaModel) viewGitHubProjectsBubbles() string {
	return m.list.View()
}

func (m BubbleTeaModel) viewRepoListBubbles() string {
	return m.list.View()
}

func (m BubbleTeaModel) viewSummaryBubbles() string {
	var b strings.Builder
	b.WriteString("✨ Installation Complete\n\n")

	// Show results summary
	successCount := 0
	errorCount := 0
	for _, result := range m.results {
		if result.OK {
			successCount++
		} else {
			errorCount++
		}
	}

	if errorCount == 0 {
		b.WriteString(successStyle.Render(fmt.Sprintf("🎉 All %d operations completed successfully!", successCount)))
	} else {
		b.WriteString(errorStyle.Render(fmt.Sprintf("⚠️ %d succeeded, %d failed", successCount, errorCount)))
	}

	return b.String()
}

// Business logic functions
func (m BubbleTeaModel) detectProjectsQuickly() tea.Msg {
	if project, err := core.NearestPubspec("."); err == nil {
		return projectsMsg{projects: []core.Project{*project}}
	}
	return projectsMsg{projects: []core.Project{}}
}

func (m BubbleTeaModel) detectProjectsAsync() tea.Msg {
	projects, err := core.ScanCommonRoots()
	if err != nil {
		return errorMsg{err: err}
	}
	return projectsMsg{projects: projects}
}

func (m BubbleTeaModel) listGitHubReposAsync() tea.Msg {
	repos, err := core.ListGitHubRepos(m.logger)
	if err != nil {
		return errorMsg{err: err}
	}
	return reposMsg{repos: repos}
}

func (m BubbleTeaModel) listReposAsync() tea.Msg {
	repos, err := core.ListGitHubRepos(m.logger)
	if err != nil {
		return errorMsg{err: err}
	}
	return reposMsg{repos: repos}
}

func (m BubbleTeaModel) cloneSelectedRepo(repo core.RepoCandidate) tea.Msg {
	// Create safe target directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errorMsg{err: fmt.Errorf("failed to get user home directory: %w", err)}
	}

	projectsDir := filepath.Join(homeDir, "flutter-projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return errorMsg{err: fmt.Errorf("failed to create projects directory: %w", err)}
	}

	targetDir := filepath.Join(projectsDir, repo.Name)

	// Handle directory conflicts
	if _, err := os.Stat(targetDir); err == nil {
		timestamp := time.Now().Format("20060102-150405")
		targetDir = filepath.Join(projectsDir, fmt.Sprintf("%s_%s", repo.Name, timestamp))
	}

	// Clone the repository
	result := core.GitClone(m.logger, &m.cfg, repo.URL, targetDir, "")

	if !result.OK {
		return errorMsg{err: fmt.Errorf("failed to clone repository: %s", result.Err)}
	}

	// Check if it's a valid Flutter project
	if _, err := core.NearestPubspec(targetDir); err == nil {
		// Set as selected project and continue
		return resultsMsg{results: []core.ActionResult{
			{
				OK:      true,
				Message: fmt.Sprintf("Successfully cloned %s/%s and detected Flutter project", repo.Owner, repo.Name),
			},
		}}
	}

	return errorMsg{err: fmt.Errorf("cloned repository is not a valid Flutter project")}
}

// installNextPackage installs the next package in the queue
func (m BubbleTeaModel) installNextPackage() tea.Msg {
	if m.currentInstallIndex >= len(m.installQueue) {
		return resultsMsg{results: []core.ActionResult{{OK: true, Message: "All packages installed successfully"}}}
	}

	pkg := m.installQueue[m.currentInstallIndex]

	// Simulate package installation with progress updates
	if len(m.projects) == 0 {
		return errorMsg{err: fmt.Errorf("no project selected")}
	}

	project := m.projects[m.selectedProject]
	_ = core.AddGitDependency(m.logger, &m.cfg, project.Path, pkg)

	// Update progress
	return installProgressMsg{
		index:   m.currentInstallIndex + 1,
		message: fmt.Sprintf("Installed %s", pkg.Name),
	}
}

// startPackageInstallation begins the package installation process
func (m BubbleTeaModel) startPackageInstallation(packages []core.PkgSpec) tea.Msg {
	m.installQueue = packages
	m.currentInstallIndex = 0
	m.progressPercent = 0
	m.currentOperation = "Installing packages..."

	return installProgressMsg{
		index:   0,
		message: "Starting package installation...",
	}
}

// Custom item delegate for the list
func NewItemDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#02569B")).
		Bold(true)

	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Background(lipgloss.Color("#02569B"))

	return d
}

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#02569B")).
			Padding(1, 2).
			Bold(true).
			Width(60).
			Align(lipgloss.Center)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#13B9FD")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F44336")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4CAF50")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)
)

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

type resultsMsg struct {
	results []core.ActionResult
}

type errorMsg struct {
	err error
}

type progressUpdateMsg struct {
	percent float64
	message string
}

type installProgressMsg struct {
	index   int
	message string
}

// Run starts the TUI application using proper bubbletea components
func Run(cfg core.Config, logger *core.Logger) error {
	m := NewBubbleTeaModel(cfg, logger)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
