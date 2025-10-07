// Package models/github_source_repo_selection_model.go - GitHub Repository Loading Screen Model
//
// This file implements the loading screen shown while fetching GitHub repositories.

package models

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// GitHubRepoModel handles the GitHub repository loading screen
type GitHubRepoModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState
}

// NewGitHubRepoModel creates a new GitHub repo loading model
func NewGitHubRepoModel(cfg core.Config, logger *core.Logger, shared *AppState) *GitHubRepoModel {
	return &GitHubRepoModel{
		cfg:    cfg,
		logger: logger,
		shared: shared,
	}
}

// Init initializes the GitHub repo loading screen
func (m *GitHubRepoModel) Init() tea.Cmd {
	// Automatically transition to repo selection
	return TransitionToScreen(ScreenSourceSelection)
}

// Update handles messages for GitHub repo loading
func (m *GitHubRepoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renders the GitHub repo loading screen
func (m *GitHubRepoModel) View() string {
	return "🔄 Loading GitHub repositories...\n\nThis will transition to repository selection."
}
