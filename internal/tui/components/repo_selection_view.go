// Package components/repo_selection_view.go - Repository Selection View Component
//
// This file implements the repository selection view component following the bubbles
// view component pattern. It handles multi-selection of GitHub repositories with
// proper pagination and shell script parity.

package components

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// RepoSelectionView represents the repository selection view component
type RepoSelectionView struct {
	cfg             core.Config
	logger          *core.Logger
	list            list.Model
	repos           []core.RepoCandidate
	selectedIndices []int
	complete        bool

	// Styling
	headerStyle    lipgloss.Style
	selectionStyle lipgloss.Style
	selectedStyle  lipgloss.Style
	normalStyle    lipgloss.Style
}

// RepoSelectionData represents data passed to the repository selection view
type RepoSelectionData struct {
	Repos []core.RepoCandidate
}

// RepoSelectionResult represents the result from repository selection
type RepoSelectionResult struct {
	SelectedRepos   []core.RepoCandidate
	SelectedIndices []int
}

// NewRepoSelectionView creates a new repository selection view component
func NewRepoSelectionView(cfg core.Config, logger *core.Logger) *RepoSelectionView {
	// Create empty list initially with default delegate
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return &RepoSelectionView{
		cfg:    cfg,
		logger: logger,
		list:   l,

		// Initialize styles
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Background(lipgloss.Color("#F0FDF4")).
			Padding(0, 1).
			Bold(true).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#10B981")),

		selectionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#8B5CF6")).
			Padding(0, 1).
			Bold(true).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#0EA5E9")),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#10B981")).
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#0EA5E9")).
			Padding(1, 2).
			Margin(0, 1).
			Bold(true),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151")).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#D1D5DB")).
			Padding(1, 2).
			Margin(0, 1),
	}
}

// RepoItem represents a repository item for the list
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

// RepoSelectionDelegate handles rendering with selection checkmarks
type RepoSelectionDelegate struct {
	list.DefaultDelegate
	view *RepoSelectionView
}

// NewRepoSelectionDelegate creates a delegate for repository selection
func NewRepoSelectionDelegate(view *RepoSelectionView) *RepoSelectionDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = view.selectionStyle
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

// Render implements custom rendering with selection checkmarks
func (d RepoSelectionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	if repoItem, ok := item.(RepoItem); ok {
		// Check if this item is selected
		isSelected := contains(d.view.selectedIndices, index)

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

		// Render with width constraints
		width := 70
		renderedTitle := titleStyle.Width(width).Render(title)
		renderedDesc := descStyle.Width(width).Render(desc)

		fmt.Fprint(w, renderedTitle+"\n"+renderedDesc)
		return
	}

	// Fallback to default rendering
	d.DefaultDelegate.Render(w, m, index, item)
}

// SetData sets the repositories for this view component
func (v *RepoSelectionView) SetData(data interface{}) {
	if repoData, ok := data.(RepoSelectionData); ok {
		v.repos = repoData.Repos
		v.setupList()
	}
}

// GetResult returns the selected repositories
func (v *RepoSelectionView) GetResult() interface{} {
	selectedRepos := make([]core.RepoCandidate, len(v.selectedIndices))
	for i, idx := range v.selectedIndices {
		selectedRepos[i] = v.repos[idx]
	}

	return RepoSelectionResult{
		SelectedRepos:   selectedRepos,
		SelectedIndices: v.selectedIndices,
	}
}

// IsComplete returns whether selection is complete
func (v *RepoSelectionView) IsComplete() bool {
	return v.complete
}

// Init initializes the view component
func (v *RepoSelectionView) Init() tea.Cmd {
	return v.list.StartSpinner()
}

// Update handles messages for this view component
func (v *RepoSelectionView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return v.handleKeys(msg)

	default:
		var cmd tea.Cmd
		v.list, cmd = v.list.Update(msg)
		return v, cmd
	}
}

// View renders this view component
func (v *RepoSelectionView) View() string {
	var b strings.Builder

	// Header
	headerText := fmt.Sprintf("🎯 Repository Selection - Found %d packages", len(v.repos))
	b.WriteString(v.headerStyle.Render(headerText) + "\n\n")

	// Repository list with pagination
	b.WriteString(v.list.View() + "\n")

	// Selection summary
	if len(v.selectedIndices) > 0 {
		summaryText := fmt.Sprintf("✨ %d repositories selected for installation", len(v.selectedIndices))
		summaryStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)
		b.WriteString(summaryStyle.Render(summaryText) + "\n")
	}

	// Help text
	helpText := "space toggle • enter confirm • up/down navigate • q quit"
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// handleKeys handles key input for repository selection
func (v *RepoSelectionView) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return v, tea.Quit

	case "up", "k":
		v.list.CursorUp()
		return v, nil

	case "down", "j":
		v.list.CursorDown()
		return v, nil

	case " ":
		// Toggle selection (shell script behavior)
		currentIndex := v.list.Index()
		if contains(v.selectedIndices, currentIndex) {
			// Remove from selection
			for i, idx := range v.selectedIndices {
				if idx == currentIndex {
					v.selectedIndices = append(v.selectedIndices[:i], v.selectedIndices[i+1:]...)
					break
				}
			}
		} else {
			// Add to selection
			v.selectedIndices = append(v.selectedIndices, currentIndex)
		}

		// Update delegate to reflect selection changes
		v.updateDelegate()
		return v, nil

	case "enter":
		// Confirm selection
		if len(v.selectedIndices) == 0 {
			// No selection, stay in this view
			return v, nil
		}
		v.complete = true
		return v, nil
	}

	return v, nil
}

// setupList configures the list with repository items
func (v *RepoSelectionView) setupList() {
	items := make([]list.Item, len(v.repos))
	for i, repo := range v.repos {
		items[i] = RepoItem{
			repo:  repo,
			index: i,
		}
	}

	v.list.SetItems(items)
	v.list.Title = fmt.Sprintf("📋 Found %d repositories - Select packages to add", len(v.repos))
	v.selectedIndices = []int{} // Reset selection

	// Set up delegate
	v.updateDelegate()
}

// updateDelegate updates the delegate to reflect current selection state
func (v *RepoSelectionView) updateDelegate() {
	delegate := NewRepoSelectionDelegate(v)
	v.list.SetDelegate(delegate)
}

// contains checks if a slice contains a value
func contains(slice []int, value int) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
