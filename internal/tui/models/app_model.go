// Package models/app_model.go - Main Application Coordinator
//
// This file implements the main application model that coordinates between
// different screen models. It manages the overall application state and
// handles transitions between screens while preserving shared data.

package models

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// AppScreen represents the current screen/model being displayed
type AppScreen int

const (
	ScreenMainMenu AppScreen = iota
	ScreenScanDirectories
	ScreenGitHubRepo
	ScreenRepoSelection
	ScreenConfiguration
	ScreenConfirmation
	ScreenExecution
	ScreenResults
	ScreenError
)

// AppModel is the main coordinator that manages screen transitions and shared state
type AppModel struct {
	// Core configuration
	cfg    core.Config
	logger *core.Logger

	// Screen management
	currentScreen AppScreen
	width         int
	height        int

	// Screen models
	mainMenu        tea.Model
	scanDirectories tea.Model
	gitHubRepo      tea.Model
	repoSelection   tea.Model
	configuration   tea.Model
	confirmation    tea.Model
	execution       tea.Model
	results         tea.Model
	errorScreen     tea.Model

	// Shared application state
	SharedState *AppState

	// Performance optimization: cache warmer
	cacheWarmer *core.CacheWarmer
}

// AppState holds data that needs to be shared between screens
type AppState struct {
	// Project information
	SelectedProject       *core.Project
	DetectedPubspecPath   string
	DetectedProject       string
	LocalPubspecAvailable bool
	HasGitDeps            bool

	// Repository data
	AvailableRepos []core.RepoCandidate
	SelectedRepos  []core.RepoCandidate

	// Package specifications
	PackageSpecs []core.PkgSpec

	// Operation results
	Results []core.ActionResult

	// User choices
	ProjectSourceChoice int // 1-6 from shell script menu
}

// ScreenTransitionMsg is sent when we need to change screens
type ScreenTransitionMsg struct {
	Screen AppScreen
	Data   interface{} // Optional data to pass to the new screen
}

// NewAppModel creates a new main application coordinator
func NewAppModel(cfg core.Config, logger *core.Logger) *AppModel {
	sharedState := &AppState{}
	cacheWarmer := core.NewCacheWarmer(logger, &cfg)

	return &AppModel{
		cfg:           cfg,
		logger:        logger,
		currentScreen: ScreenMainMenu,
		SharedState:   sharedState,
		cacheWarmer:   cacheWarmer,
	}
}

// Init initializes the app model
func (m *AppModel) Init() tea.Cmd {
	// Start background cache warming for better performance
	m.cacheWarmer.Start()

	// Initialize the first screen (MainMenu)
	m.mainMenu = NewMainMenuModel(m.cfg, m.logger, m.SharedState)
	return m.mainMenu.Init()
}

// Update handles messages and coordinates between screens
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Pass size to current screen
		return m.updateCurrentScreen(msg)

	case ScreenTransitionMsg:
		return m.transitionToScreen(msg.Screen, msg.Data)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			// Stop cache warmer before quitting
			m.cacheWarmer.Stop()
			return m, tea.Quit
		}
		// Pass other keys to current screen
		return m.updateCurrentScreen(msg)

	default:
		// Pass message to current screen
		return m.updateCurrentScreen(msg)
	}
}

// View renders the current screen
func (m *AppModel) View() string {
	switch m.currentScreen {
	case ScreenMainMenu:
		if m.mainMenu != nil {
			return m.mainMenu.View()
		}
	case ScreenScanDirectories:
		if m.scanDirectories != nil {
			return m.scanDirectories.View()
		}
	case ScreenGitHubRepo:
		if m.gitHubRepo != nil {
			return m.gitHubRepo.View()
		}
	case ScreenRepoSelection:
		if m.repoSelection != nil {
			return m.repoSelection.View()
		}
	case ScreenConfiguration:
		if m.configuration != nil {
			return m.configuration.View()
		}
	case ScreenConfirmation:
		if m.confirmation != nil {
			return m.confirmation.View()
		}
	case ScreenExecution:
		if m.execution != nil {
			return m.execution.View()
		}
	case ScreenResults:
		if m.results != nil {
			return m.results.View()
		}
	case ScreenError:
		if m.errorScreen != nil {
			return m.errorScreen.View()
		}
	}

	return "Loading..."
}

// updateCurrentScreen passes messages to the current screen model
func (m *AppModel) updateCurrentScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.currentScreen {
	case ScreenMainMenu:
		if m.mainMenu != nil {
			m.mainMenu, cmd = m.mainMenu.Update(msg)
		}
	case ScreenScanDirectories:
		if m.scanDirectories != nil {
			m.scanDirectories, cmd = m.scanDirectories.Update(msg)
		}
	case ScreenGitHubRepo:
		if m.gitHubRepo != nil {
			m.gitHubRepo, cmd = m.gitHubRepo.Update(msg)
		}
	case ScreenRepoSelection:
		if m.repoSelection != nil {
			m.repoSelection, cmd = m.repoSelection.Update(msg)
		}
	case ScreenConfiguration:
		if m.configuration != nil {
			m.configuration, cmd = m.configuration.Update(msg)
		}
	case ScreenConfirmation:
		if m.confirmation != nil {
			m.confirmation, cmd = m.confirmation.Update(msg)
		}
	case ScreenExecution:
		if m.execution != nil {
			m.execution, cmd = m.execution.Update(msg)
		}
	case ScreenResults:
		if m.results != nil {
			m.results, cmd = m.results.Update(msg)
		}
	case ScreenError:
		if m.errorScreen != nil {
			m.errorScreen, cmd = m.errorScreen.Update(msg)
		}
	}

	return m, cmd
}

// transitionToScreen handles switching between screens
func (m *AppModel) transitionToScreen(screen AppScreen, data interface{}) (tea.Model, tea.Cmd) {
	m.currentScreen = screen

	switch screen {
	case ScreenMainMenu:
		if m.mainMenu == nil {
			m.mainMenu = NewMainMenuModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.mainMenu.Init()

	case ScreenScanDirectories:
		if m.scanDirectories == nil {
			m.scanDirectories = NewScanDirectoriesModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.scanDirectories.Init()

	case ScreenGitHubRepo:
		if m.gitHubRepo == nil {
			m.gitHubRepo = NewGitHubRepoModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.gitHubRepo.Init()

	case ScreenRepoSelection:
		if m.repoSelection == nil {
			m.repoSelection = NewRepoSelectionModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.repoSelection.Init()

	case ScreenConfiguration:
		if m.configuration == nil {
			m.configuration = NewConfigurationModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.configuration.Init()

	case ScreenConfirmation:
		if m.confirmation == nil {
			m.confirmation = NewConfirmationModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.confirmation.Init()

	case ScreenExecution:
		if m.execution == nil {
			m.execution = NewExecutionModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.execution.Init()

	case ScreenResults:
		if m.results == nil {
			m.results = NewResultsModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.results.Init()

	case ScreenError:
		if m.errorScreen == nil {
			m.errorScreen = NewErrorModel(m.cfg, m.logger, m.SharedState)
		}
		// Set error data if provided
		if errorData, ok := data.(ErrorData); ok {
			m.errorScreen.(*ErrorModel).SetError(errorData)
		}
		return m, m.errorScreen.Init()
	}

	return m, nil
}

// Helper function to send screen transition commands
func TransitionToScreen(screen AppScreen) tea.Cmd {
	return func() tea.Msg {
		return ScreenTransitionMsg{Screen: screen}
	}
}
