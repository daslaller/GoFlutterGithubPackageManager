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
	"io"
)

// RepoSelectionModel handles repository browsing and selection using list-simple style
type RepoSelectionModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// UI components with custom delegate
	list     list.Model
	delegate *simpleMultiSelectDelegate
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

// simpleMultiSelectDelegate is a custom delegate for list-simple style with > markers
type simpleMultiSelectDelegate struct {
	selectedItems map[int]bool
	cursorStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
}

func newSimpleMultiSelectDelegate() *simpleMultiSelectDelegate {
	return &simpleMultiSelectDelegate{
		selectedItems: make(map[int]bool),
		cursorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")). // Vibrant amber/orange
			Bold(true),
		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#8B5CF6")). // Beautiful purple
			Bold(true).
			Padding(0, 1),
		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")), // Lighter gray
	}
}

func (d *simpleMultiSelectDelegate) Height() int                               { return 1 }
func (d *simpleMultiSelectDelegate) Spacing() int                              { return 0 }
func (d *simpleMultiSelectDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d *simpleMultiSelectDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if item, ok := listItem.(RepoItem); ok {
		var cursor string
		if index == m.Index() {
			cursor = d.cursorStyle.Render("▶ ") // Beautiful arrow instead of >
		} else {
			cursor = "  "
		}

		itemText := item.Title()
		if d.selectedItems[index] {
			// Highlighted selected item with glowing effect
			itemText = d.selectedStyle.Render(" ✓ " + itemText + " ")
		} else {
			if index == m.Index() {
				// Highlight current item even if not selected
				highlightStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#10B981")). // Green for hover
					Bold(true)
				itemText = highlightStyle.Render(itemText)
			} else {
				itemText = d.normalStyle.Render(itemText)
			}
		}

		line := cursor + itemText
		fmt.Fprint(w, line)
	}
}

// toggleSelection toggles the selection state of an item
func (d *simpleMultiSelectDelegate) toggleSelection(index int) {
	if d.selectedItems[index] {
		delete(d.selectedItems, index)
	} else {
		d.selectedItems[index] = true
	}
}

// getSelectedIndices returns a slice of selected indices
func (d *simpleMultiSelectDelegate) getSelectedIndices() []int {
	var indices []int
	for idx := range d.selectedItems {
		indices = append(indices, idx)
	}
	return indices
}

// reposLoadedMsg is sent when repositories are loaded
type reposLoadedMsg struct {
	repos []core.RepoCandidate
	err   error
}

// NewRepoSelectionModel creates a new repository selection model using list-simple style
func NewRepoSelectionModel(cfg core.Config, logger *core.Logger, shared *AppState) *RepoSelectionModel {
	// Create custom delegate for list-simple style with > markers and highlights
	delegate := newSimpleMultiSelectDelegate()

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
	// Check if this is SOURCE selection mode (AvailableSourceRepos populated)
	if len(m.shared.AvailableSourceRepos) > 0 {
		// SOURCE SELECTION MODE - single select, don't reset selections
		m.loading = false
		m.ready = true
		m.setupListFromSource()
		return nil
	}

	// PACKAGE SELECTION MODE - multiselect
	// Reset selection state each time the screen starts
	m.delegate.selectedItems = make(map[int]bool)

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

	// Check if we're in SOURCE mode
	isSourceMode := len(m.shared.AvailableSourceRepos) > 0

	// Beautiful bordered header with gradient-like colors
	var headerText string
	var itemCount int
	var headerColor string
	if isSourceMode {
		headerText = "📁 Select Source Flutter Project"
		itemCount = len(m.shared.AvailableSourceRepos)
		headerColor = "#F59E0B" // Warm amber for source
	} else {
		headerText = "📦 Add Dependencies"
		itemCount = len(m.shared.AvailableDependencies)
		headerColor = "#8B5CF6" // Cool purple for packages
	}

	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(headerColor)).
		Foreground(lipgloss.Color(headerColor)).
		Padding(1, 2).
		Align(lipgloss.Center).
		Width(62).
		Bold(true).
		Render(fmt.Sprintf("%s (%d available)", headerText, itemCount))

	b.WriteString(headerBox + "\n\n")

	// Calculate overflow indicators using list's internal pagination
	totalItems := len(m.shared.AvailableDependencies)
	if totalItems > 0 {
		visibleItems := m.list.Paginator.ItemsOnPage(totalItems)
		currentPage := m.list.Paginator.Page
		itemsPerPage := m.list.Paginator.PerPage

		startIndex := currentPage * itemsPerPage
		endIndex := startIndex + visibleItems

		// Show overflow indicator at top with gradient
		if startIndex > 0 {
			overflowStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")). // Cyan
				Bold(true)
			overflowText := fmt.Sprintf("▲ %d more above ▲", startIndex)
			b.WriteString(overflowStyle.Render(overflowText) + "\n")
		}

		// Beautiful native list rendering
		b.WriteString(m.list.View())

		// Show overflow indicator at bottom with gradient
		if endIndex < totalItems {
			overflowStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")). // Cyan
				Bold(true)
			overflowText := fmt.Sprintf("▼ %d more below ▼", totalItems-endIndex)
			b.WriteString("\n" + overflowStyle.Render(overflowText))
		}
	} else {
		b.WriteString(m.list.View())
	}

	// Footer with selection info
	b.WriteString("\n\n")
	selectedIndices := m.delegate.getSelectedIndices()
	if len(selectedIndices) > 0 && !isSourceMode {
		// Show selected packages with beautiful styling
		selectedNames := []string{}
		for _, idx := range selectedIndices {
			if idx < len(m.shared.AvailableDependencies) {
				repo := m.shared.AvailableDependencies[idx]
				selectedNames = append(selectedNames, repo.Owner+"/"+repo.Name)
			}
		}
		if len(selectedNames) > 0 {
			selectionCountStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

			selectionText := fmt.Sprintf("Selected: %s", strings.Join(selectedNames, ", "))
			if len(selectionText) > 60 {
				selectionText = selectionCountStyle.Render(fmt.Sprintf("✓ %d packages selected", len(selectedIndices)))
			} else {
				selectionText = selectionCountStyle.Render("✓ " + selectionText)
			}
			b.WriteString(selectionText + "\n")
		}
	}

	// Footer instructions with colors
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#94A3B8")).
		Italic(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)

	if isSourceMode {
		b.WriteString(helpStyle.Render(keyStyle.Render("enter") + ": select project • " + keyStyle.Render("q") + ": back to menu"))
	} else {
		if len(selectedIndices) > 0 {
			b.WriteString(helpStyle.Render(keyStyle.Render("space") + ": toggle • " + keyStyle.Render("enter") + ": confirm selection • " + keyStyle.Render("q") + ": back"))
		} else {
			b.WriteString(helpStyle.Render(keyStyle.Render("space") + ": toggle packages • select at least 1 to continue • " + keyStyle.Render("q") + ": back"))
		}
	}

	return b.String()
}

// handleKeys handles keyboard input
func (m *RepoSelectionModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check if we're in SOURCE selection mode
	isSourceMode := len(m.shared.AvailableSourceRepos) > 0

	switch msg.String() {
	case "q", "ctrl+c":
		return m, TransitionToScreen(ScreenMainMenu)

	case " ":
		if isSourceMode {
			// SOURCE MODE: space does nothing (single-select only)
			return m, nil
		}
		// PACKAGE MODE: Multi-select - toggle selection using delegate
		currentIndex := m.list.Index()
		if currentIndex >= 0 && currentIndex < len(m.shared.AvailableDependencies) {
			m.delegate.toggleSelection(currentIndex)
		}
		return m, nil

	case "enter":
		if isSourceMode {
			// SOURCE MODE: Select single source and move to source configuration
			currentIndex := m.list.Index()
			if currentIndex >= 0 && currentIndex < len(m.shared.AvailableSourceRepos) {
				selectedRepo := m.shared.AvailableSourceRepos[currentIndex]
				m.shared.SourceProject = &core.Project{
					Name: selectedRepo.Name,
					Path: "",
				}
				m.logger.Info("source_selection", fmt.Sprintf("Selected source: %s/%s", selectedRepo.Owner, selectedRepo.Name))

				// Go to source config (save location editing)
				return m, TransitionToScreen(ScreenSourceConfig)
			}
			return m, nil
		}

		// PACKAGE MODE: Confirm multi-selection and move to configuration
		selectedIndices := m.delegate.getSelectedIndices()
		if len(selectedIndices) == 0 {
			// Don't allow proceeding without selecting packages
			return m, nil
		}
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

// setupListFromSource configures the list with source repositories (single-select mode)
func (m *RepoSelectionModel) setupListFromSource() {
	items := make([]list.Item, len(m.shared.AvailableSourceRepos))
	for i, repo := range m.shared.AvailableSourceRepos {
		items[i] = RepoItem{
			repo:  repo,
			index: i,
		}
	}
	m.list.SetItems(items)
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
	// Restore selection markers from shared state when returning to the screen
	if len(m.shared.SelectedDependencies) > 0 {
		selected := make(map[string]struct{}, len(m.shared.SelectedDependencies))
		for _, repo := range m.shared.SelectedDependencies {
			key := repo.Owner + "/" + repo.Name + "|" + repo.URL
			selected[key] = struct{}{}
		}

		for i, repo := range m.shared.AvailableDependencies {
			key := repo.Owner + "/" + repo.Name + "|" + repo.URL
			if _, ok := selected[key]; ok {
				m.delegate.selectedItems[i] = true
			}
		}
	}
	// Use default delegate for clean list-simple style with > indicator
}

// finalizeSelection saves the selected repositories to shared state
func (m *RepoSelectionModel) finalizeSelection() {
	selectedIndices := m.delegate.getSelectedIndices()
	selectedRepos := make([]core.RepoCandidate, 0, len(selectedIndices))
	for _, index := range selectedIndices {
		if index >= 0 && index < len(m.shared.AvailableDependencies) {
			selectedRepos = append(selectedRepos, m.shared.AvailableDependencies[index])
		} else {
			m.logger.Debug("repo_selection", fmt.Sprintf("Invalid repository index: %d", index))
		}
	}
	m.shared.SelectedDependencies = selectedRepos

	m.logger.Info("repo_selection", fmt.Sprintf("Selected %d repositories", len(selectedRepos)))
}

// isSelected checks if a repository at the given index is selected
func (m *RepoSelectionModel) isSelected(index int) bool {
	return m.delegate.selectedItems[index]
}

// Single-select mode - using native list rendering with beautiful styling
