// Package models/github_source_repo_selection_model.go - GitHub Repository Loading Screen Model
//
// This file implements the loading screen shown while fetching GitHub repositories.

package models

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// gitHubReposLoadedMsg is emitted when repositories have been fetched
type gitHubReposLoadedMsg struct {
	repos []core.RepoCandidate
	err   error
}

// GitHubRepoModel handles the GitHub repository loading screen
type GitHubRepoModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	spinner spinner.Model
	loading bool
}

// NewGitHubRepoModel creates a new GitHub repo loading model
func NewGitHubRepoModel(cfg core.Config, logger *core.Logger, shared *AppState) *GitHubRepoModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#0EA5E9"))

	return &GitHubRepoModel{
		cfg:     cfg,
		logger:  logger,
		shared:  shared,
		spinner: s,
		loading: true,
	}
}

// Init initializes the GitHub repo loading screen
func (m *GitHubRepoModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(
		m.spinner.Tick,
		m.loadRepositories(),
	)
}

// Update handles messages for GitHub repo loading
func (m *GitHubRepoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case gitHubReposLoadedMsg:
		m.loading = false
		if msg.err != nil {
			wrappedErr := fmt.Errorf("failed to list GitHub repositories: %w", msg.err)
			m.logger.Error("github_repo_loader", wrappedErr)
			errorData := ErrorData{
				Title:        "GitHub repository fetch failed",
				Message:      "We couldn't list your GitHub repositories. Please check your network connection and GitHub authentication (gh auth status).",
				Error:        wrappedErr,
				ReturnScreen: ScreenMainMenu,
			}
			return m, func() tea.Msg {
				return ScreenTransitionMsg{Screen: ScreenError, Data: errorData}
			}
		}

		// Store repositories for the selection screen and reset any previous picks
		m.shared.AvailableDependencies = msg.repos
		m.shared.SelectedDependencies = nil

		return m, TransitionToScreen(ScreenDependencySelection)
	}

	return m, nil
}

// View renders the GitHub repo loading screen
func (m *GitHubRepoModel) View() string {
	if !m.loading {
		return "Preparing repository list..."
	}

	return fmt.Sprintf("\n%s Fetching GitHub repositories...\n\nPlease wait while we gather available packages.\n", m.spinner.View())
}

// loadRepositories fetches repositories from GitHub
func (m *GitHubRepoModel) loadRepositories() tea.Cmd {
	return func() tea.Msg {
		repos, err := core.ListGitHubRepos(m.logger)
		return gitHubReposLoadedMsg{repos: repos, err: err}
	}
}
