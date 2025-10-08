// Package models/repo_selection_model.go - Repository Selection Screen Model
//
// This file implements the repository selection screen where users can
// browse GitHub repositories and choose a single source to add as a dependency.

package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
	"io"
)

// RepoSelectionModel handles repository browsing and selection using list-simple style
type RepoSelectionModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// UI components with custom delegate
	list     list.Model
	delegate *simpleSingleSelectDelegate
	spinner  spinner.Model

	// State
	loading     bool
	loadingText string
	ready       bool
	quitting    bool

	// Styles
	headerStyle   lipgloss.Style
	overflowStyle lipgloss.Style
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

// simpleSingleSelectDelegate is a custom delegate for list-simple style with > markers
type simpleSingleSelectDelegate struct {
	selectedIndex int
	cursorStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
}

func newSimpleSingleSelectDelegate() *simpleSingleSelectDelegate {
	return &simpleSingleSelectDelegate{
		selectedIndex: -1,
		cursorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0EA5E9")).
			Bold(true),
		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#10B981")).
			Bold(true),
		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151")),
	}
}

func (d *simpleSingleSelectDelegate) Height() int                               { return 1 }
func (d *simpleSingleSelectDelegate) Spacing() int                              { return 0 }
func (d *simpleSingleSelectDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d *simpleSingleSelectDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if item, ok := listItem.(RepoItem); ok {
		cursor := "  "
		if index == m.Index() {
			cursor = "> " // Simple > marker like list-simple
		}

		itemText := item.Title()
		if index == d.selectedIndex {
			// Highlighted selected item
			itemText = d.selectedStyle.Render("✓ " + itemText)
		} else {
			itemText = d.normalStyle.Render(itemText)
		}

		line := d.cursorStyle.Render(cursor) + itemText
		fmt.Fprint(w, line)
	}
}

// setSelected updates the selected index
func (d *simpleSingleSelectDelegate) setSelected(index int) {
	if index < 0 {
		d.selectedIndex = -1
		return
	}
	d.selectedIndex = index
}

// clearSelection clears any selected index
func (d *simpleSingleSelectDelegate) clearSelection() {
	d.selectedIndex = -1
}

// getSelectedIndex returns the selected index if one exists
func (d *simpleSingleSelectDelegate) getSelectedIndex() (int, bool) {
	if d.selectedIndex >= 0 {
		return d.selectedIndex, true
	}
	return 0, false
}

// reposLoadedMsg is sent when repositories are loaded
type reposLoadedMsg struct {
	repos []core.RepoCandidate
	err   error
}

// NewRepoSelectionModel creates a new repository selection model using list-simple style
func NewRepoSelectionModel(cfg core.Config, logger *core.Logger, shared *AppState) *RepoSelectionModel {
	// Create custom delegate for list-simple style with > markers and highlights
	delegate := newSimpleSingleSelectDelegate()

	// Create list with custom delegate
	l := list.New([]list.Item{}, delegate, 80, 20)
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
		delegate:    delegate,
		spinner:     s,
		loading:     true,
		loadingText: "Fetching GitHub repositories...",

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")).
			Bold(true),

		overflowStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")).
			Bold(true),
	}
}

// Init initializes the repository selection screen
func (m *RepoSelectionModel) Init() tea.Cmd {
	// Reset selection state each time the screen starts
	m.delegate.clearSelection()

	if len(m.shared.AvailableDependencies) > 0 {
		m.loading = false
		m.ready = true
		m.setupList()
		return nil
	}

	m.loading = true
	m.ready = false
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
		m.shared.AvailableDependencies = msg.repos
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
		Render(fmt.Sprintf("📦 Add Dependencies (%d available)", len(m.shared.AvailableDependencies)))

	b.WriteString(headerBox + "\n\n")

	// Calculate overflow indicators using list's internal pagination
	totalItems := len(m.shared.AvailableDependencies)
	if totalItems > 0 {
		visibleItems := m.list.Paginator.ItemsOnPage(totalItems)
		currentPage := m.list.Paginator.Page
		itemsPerPage := m.list.Paginator.PerPage

		startIndex := currentPage * itemsPerPage
		endIndex := startIndex + visibleItems

		// Show overflow indicator at top
		if startIndex > 0 {
			overflowText := fmt.Sprintf("▲ %d more above", startIndex)
			b.WriteString(m.overflowStyle.Render(overflowText) + "\n")
		}

		// Beautiful native list rendering (single-select style)
		b.WriteString(m.list.View())

		// Show overflow indicator at bottom
		if endIndex < totalItems {
			overflowText := fmt.Sprintf("▼ %d more below", totalItems-endIndex)
			b.WriteString("\n" + m.overflowStyle.Render(overflowText))
		}
	} else {
		b.WriteString(m.list.View())
	}

	// Footer with selection info (list-simple style)
	b.WriteString("\n\n")
	if selectedIndex, ok := m.delegate.getSelectedIndex(); ok {
		if selectedIndex >= 0 && selectedIndex < len(m.shared.AvailableDependencies) {
			repo := m.shared.AvailableDependencies[selectedIndex]
			selectionText := fmt.Sprintf("Selected: %s/%s", repo.Owner, repo.Name)
			if repo.Desc != "" {
				selectionText += fmt.Sprintf(" — %s", repo.Desc)
			}
			if len(selectionText) > 80 {
				selectionText = fmt.Sprintf("Selected: %s/%s", repo.Owner, repo.Name)
			}
			b.WriteString(m.delegate.selectedStyle.Render(selectionText) + "\n")
		}
	}

	// Footer instructions
	b.WriteString("enter: add repository • space: mark selection • q: back to menu")

	return b.String()
}

// handleKeys handles keyboard input
func (m *RepoSelectionModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, TransitionToScreen(ScreenMainMenu)

	case "enter":
		// Confirm selection and move to configuration
		currentIndex := m.list.Index()
		if currentIndex < 0 || currentIndex >= len(m.shared.AvailableDependencies) {
			return m, nil
		}
		m.delegate.setSelected(currentIndex)
		m.finalizeSelection()
		return m, TransitionToScreen(ScreenConfiguration)

	case " ":
		// Space marks the current selection without leaving the screen
		currentIndex := m.list.Index()
		if currentIndex >= 0 && currentIndex < len(m.shared.AvailableDependencies) {
			m.delegate.setSelected(currentIndex)
		}
		return m, nil

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
	items := make([]list.Item, len(m.shared.AvailableDependencies))
	for i, repo := range m.shared.AvailableDependencies {
		items[i] = RepoItem{
			repo:  repo,
			index: i,
		}
	}
	m.list.SetItems(items)
	// Restore selection marker from shared state when returning to the screen
	if len(m.shared.SelectedDependencies) > 0 {
		selectedRepo := m.shared.SelectedDependencies[0]
		for i, repo := range m.shared.AvailableDependencies {
			if repo.Owner == selectedRepo.Owner && repo.Name == selectedRepo.Name && repo.URL == selectedRepo.URL {
				m.delegate.setSelected(i)
				m.list.Select(i)
				break
			}
		}
	}
	// Use default delegate for clean list-simple style with > indicator
}

// finalizeSelection saves the selected repositories to shared state
func (m *RepoSelectionModel) finalizeSelection() {
	if index, ok := m.delegate.getSelectedIndex(); ok {
		if index >= 0 && index < len(m.shared.AvailableDependencies) {
			repo := m.shared.AvailableDependencies[index]
			m.shared.SelectedDependencies = []core.RepoCandidate{repo}
			m.logger.Info("repo_selection", fmt.Sprintf("Selected repository %s/%s", repo.Owner, repo.Name))
			return
		}
		m.logger.Debug("repo_selection", fmt.Sprintf("Invalid repository index: %d", index))
	}

	// If selection is invalid or missing, clear shared selection
	m.shared.SelectedDependencies = nil
}

// isSelected checks if a repository at the given index is selected
func (m *RepoSelectionModel) isSelected(index int) bool {
	selectedIndex, ok := m.delegate.getSelectedIndex()
	return ok && selectedIndex == index
}

// Single-select mode - using native list rendering with beautiful styling
