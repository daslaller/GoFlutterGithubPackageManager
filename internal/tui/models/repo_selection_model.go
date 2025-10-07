// Package models/repo_selection_model.go - Repository Selection Screen Model
//
// This file implements the repository selection screen where users can
// browse and multi-select GitHub repositories to add as dependencies.

package models

import (
	"fmt"
	"strings"

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
	// Simple clean format like list-simple
	return fmt.Sprintf("%s/%s", i.repo.Owner, i.repo.Name)
}

func (i RepoItem) Description() string {
	// Minimal description for list-simple style
	if i.repo.Desc == "" {
		return ""
	}
	if len(i.repo.Desc) > 50 {
		return i.repo.Desc[:47] + "..."
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
	// Create simple list without title or status bar (list-simple style)
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)

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

// View renders the repository selection screen with overflow indicators
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

	var b strings.Builder

	// Beautiful bordered header matching main menu style
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#0EA5E9")).
		Padding(1, 2).
		Align(lipgloss.Center).
		Width(62).
		Render(fmt.Sprintf("🎯 Found %d repositories", len(m.shared.AvailableRepos)))

	b.WriteString(headerBox + "\n\n")

	// Calculate overflow indicators using list's internal pagination
	totalItems := len(m.shared.AvailableRepos)
	if totalItems > 0 {
		visibleItems := m.list.Paginator.ItemsOnPage(totalItems)
		currentPage := m.list.Paginator.Page
		itemsPerPage := m.list.Paginator.PerPage

		startIndex := currentPage * itemsPerPage
		endIndex := startIndex + visibleItems

		// Show overflow indicator at top
		if startIndex > 0 {
			overflowText := fmt.Sprintf("▲ %d more above", startIndex)
			b.WriteString(m.selectedStyle.Render(overflowText) + "\n")
		}

		// Main list content
		b.WriteString(m.list.View())

		// Show overflow indicator at bottom
		if endIndex < totalItems {
			overflowText := fmt.Sprintf("▼ %d more below", totalItems-endIndex)
			b.WriteString("\n" + m.selectedStyle.Render(overflowText))
		}
	} else {
		b.WriteString(m.list.View())
	}

	// Footer with selection info (list-simple style)
	b.WriteString("\n\n")
	if len(m.selectedRepos) > 0 {
		// Show selected items in a simple list-simple style
		selectedNames := []string{}
		for _, idx := range m.selectedRepos {
			if idx < len(m.shared.AvailableRepos) {
				repo := m.shared.AvailableRepos[idx]
				selectedNames = append(selectedNames, repo.Owner+"/"+repo.Name)
			}
		}
		if len(selectedNames) > 0 {
			selectionText := fmt.Sprintf("Selected: %s", strings.Join(selectedNames, ", "))
			if len(selectionText) > 60 {
				selectionText = fmt.Sprintf("Selected %d repositories", len(m.selectedRepos))
			}
			b.WriteString(m.selectedStyle.Render(selectionText) + "\n")
		}
	}
	b.WriteString("space: toggle • enter: confirm • q: back to menu")

	return b.String()
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

// setupList configures the list with repository items and custom delegate
func (m *RepoSelectionModel) setupList() {
	items := make([]list.Item, len(m.shared.AvailableRepos))
	for i, repo := range m.shared.AvailableRepos {
		items[i] = RepoItem{
			repo:  repo,
			index: i,
		}
	}
	m.list.SetItems(items)
	// Use default delegate for clean list-simple style
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

// Simple list-simple style - no custom delegate needed
