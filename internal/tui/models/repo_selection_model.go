// Package models/repo_selection_model.go - Repository Selection Screen Model
//
// This file implements the repository selection screen where users can
// browse and multi-select GitHub repositories to add as dependencies.

package models

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// RepoSelectionModel handles repository browsing and selection
type RepoSelectionModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// UI components
	list          list.Model
	spinner       spinner.Model
	selectedRepos []int // indices of selected repositories

	// State
	loading     bool
	loadingText string
	ready       bool
	quitting    bool

	// Styles
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
	headerStyle   lipgloss.Style
}

// RepoItem represents a repository in the list
type RepoItem struct {
	repo  core.RepoCandidate
	index int
}

func (i RepoItem) Title() string {
	privacy := "🔓"
	if i.repo.Privacy == "private" {
		privacy = "🔒"
	}
	return fmt.Sprintf("%s %s/%s", privacy, i.repo.Owner, i.repo.Name)
}

func (i RepoItem) Description() string {
	if i.repo.Desc == "" {
		return "No description"
	}
	if len(i.repo.Desc) > 60 {
		return i.repo.Desc[:57] + "..."
	}
	return i.repo.Desc
}

func (i RepoItem) FilterValue() string {
	return fmt.Sprintf("%s/%s %s", i.repo.Owner, i.repo.Name, i.repo.Desc)
}

// reposLoadedMsg is sent when repositories are loaded
type reposLoadedMsg struct {
	repos []core.RepoCandidate
	err   error
}

// NewRepoSelectionModel creates a new repository selection model
func NewRepoSelectionModel(cfg core.Config, logger *core.Logger, shared *AppState) *RepoSelectionModel {
	// Create list with default delegate (will be updated after model creation)
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20)
	l.Title = "📋 GitHub Repositories - Select packages to add"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD"))

	return &RepoSelectionModel{
		cfg:         cfg,
		logger:      logger,
		shared:      shared,
		list:        l,
		spinner:     s,
		loading:     true,
		loadingText: "Fetching GitHub repositories...",

		// Styles (restored original styling)
		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#8B5CF6")).
			Padding(0, 1).
			Bold(true).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#0EA5E9")),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151")).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#D1D5DB")).
			Padding(1, 2).
			Margin(0, 1),

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")).
			Bold(true),
	}
}

// Init initializes the repository selection screen
func (m *RepoSelectionModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadRepositories(),
	)
}

// Update handles messages for repository selection
func (m *RepoSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			// Don't handle keys while loading
			return m, nil
		}
		return m.handleKeys(msg)

	case reposLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.logger.Error("repo_selection", fmt.Errorf("failed to load repositories: %w", msg.err))
			// Could show error screen or return to main menu
			return m, TransitionToScreen(ScreenMainMenu)
		}

		// Update shared state and list
		m.shared.AvailableRepos = msg.repos
		m.setupList()
		m.ready = true
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 8) // Leave space for header and footer
		return m, nil

	default:
		if m.ready {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the repository selection screen
func (m *RepoSelectionModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		return fmt.Sprintf("\n%s %s\n\n", m.spinner.View(), m.loadingText)
	}

	if !m.ready {
		return "\nPreparing repository list...\n\n"
	}

	header := m.headerStyle.Render(fmt.Sprintf("🎯 Found %d repositories", len(m.shared.AvailableRepos)))

	content := m.list.View()

	footer := ""
	if len(m.selectedRepos) > 0 {
		footer = m.selectedStyle.Render(fmt.Sprintf("✨ %d repositories selected", len(m.selectedRepos)))
		footer += "\n"
	}
	footer += "space: toggle • enter: confirm • q: back to menu"

	return header + "\n\n" + content + "\n\n" + footer
}

// handleKeys handles keyboard input
func (m *RepoSelectionModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, TransitionToScreen(ScreenMainMenu)

	case " ":
		// Toggle selection
		currentIndex := m.list.Index()
		m.toggleSelection(currentIndex)
		return m, nil

	case "enter":
		// Confirm selection and move to next screen
		m.finalizeSelection()
		return m, TransitionToScreen(ScreenConfiguration)

	default:
		// Pass other keys to the list
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
}

// loadRepositories loads repositories from GitHub
func (m *RepoSelectionModel) loadRepositories() tea.Cmd {
	return func() tea.Msg {
		repos, err := core.ListGitHubRepos(m.logger)
		return reposLoadedMsg{repos: repos, err: err}
	}
}

// setupList configures the list with repository items
func (m *RepoSelectionModel) setupList() {
	items := make([]list.Item, len(m.shared.AvailableRepos))
	for i, repo := range m.shared.AvailableRepos {
		items[i] = RepoItem{
			repo:  repo,
			index: i,
		}
	}
	m.list.SetItems(items)
}

// toggleSelection toggles the selection state of a repository
func (m *RepoSelectionModel) toggleSelection(index int) {
	// Check if already selected
	for i, selectedIndex := range m.selectedRepos {
		if selectedIndex == index {
			// Remove from selection
			m.selectedRepos = append(m.selectedRepos[:i], m.selectedRepos[i+1:]...)
			m.logger.Debug("repo_selection", fmt.Sprintf("Deselected repository at index %d", index))
			return
		}
	}
	// Add to selection
	m.selectedRepos = append(m.selectedRepos, index)
	m.logger.Debug("repo_selection", fmt.Sprintf("Selected repository at index %d", index))
}

// finalizeSelection saves the selected repositories to shared state
func (m *RepoSelectionModel) finalizeSelection() {
	selectedRepos := make([]core.RepoCandidate, len(m.selectedRepos))
	for i, index := range m.selectedRepos {
		selectedRepos[i] = m.shared.AvailableRepos[index]
	}
	m.shared.SelectedRepos = selectedRepos

	m.logger.Info("repo_selection", fmt.Sprintf("Selected %d repositories", len(selectedRepos)))
}

// isSelected checks if a repository at the given index is selected
func (m *RepoSelectionModel) isSelected(index int) bool {
	for _, selectedIndex := range m.selectedRepos {
		if selectedIndex == index {
			return true
		}
	}
	return false
}

// RepoSelectionDelegate handles rendering with selection checkmarks (restored original styling)
type RepoSelectionDelegate struct {
	list.DefaultDelegate
	view *RepoSelectionModel
}

// NewRepoSelectionDelegate creates a delegate for repository selection with original styling
func NewRepoSelectionDelegate(view *RepoSelectionModel) *RepoSelectionDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = view.selectedStyle
	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Background(lipgloss.Color("#8B5CF6"))
	d.Styles.NormalTitle = view.normalStyle
	d.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	return &RepoSelectionDelegate{
		DefaultDelegate: d,
		view:            view,
	}
}

// Render implements custom rendering for repository items with selection checkmarks (original logic)
func (d RepoSelectionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	if repoItem, ok := item.(RepoItem); ok {
		// Check if this item is selected
		isSelected := d.view.isSelected(index)

		// Create prefix
		var prefix string
		if isSelected {
			prefix = "✅ "
		} else {
			prefix = "   "
		}

		// Build title and description
		title := prefix + repoItem.Title()
		desc := repoItem.Description()

		// Apply styles
		var titleStyle, descStyle lipgloss.Style
		if isSelected {
			titleStyle = d.view.selectedStyle
			descStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#10B981"))
		} else {
			titleStyle = d.DefaultDelegate.Styles.NormalTitle
			descStyle = d.DefaultDelegate.Styles.NormalDesc
		}

		// Use a reasonable width for rendering (list items)
		width := 70 // Standard terminal width minus margins

		// Render with proper width constraints
		renderedTitle := titleStyle.Width(width).Render(title)
		renderedDesc := descStyle.Width(width).Render(desc)

		_, err := fmt.Fprint(w, renderedTitle+"\n"+renderedDesc)
		if err != nil {
			return
		}
		return
	}

	// Fallback to default rendering
	d.DefaultDelegate.Render(w, m, index, item)
}
