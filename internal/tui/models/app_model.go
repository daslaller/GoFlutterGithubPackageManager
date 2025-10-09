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
	ScreenSplash        AppScreen = iota // NEW: Initial splash screen with prerequisites check
	ScreenMainMenu
	ScreenPrerequisites           // NEW: Check prerequisites and provide guidance
	ScreenScanDirectories
	ScreenGitHubRepo
	ScreenSourceSelection     // NEW: Select which Flutter project to work on
	ScreenSourceConfig        // NEW: Configure download location and name
	ScreenSourceDownload      // NEW: Download/clone the source project
	ScreenDependencySelection // RENAMED: Multi-select dependencies to add (was ScreenRepoSelection)
	ScreenConfiguration
	ScreenConfirmation
	ScreenExecution
	ScreenResults
	ScreenSearchConfig // NEW: Configure repository search filters
	ScreenForceUpdate  // NEW: Force update stale packages
	ScreenSelfUpdate   // NEW: Update Flutter-PM itself
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
	splash              tea.Model // NEW: Splash screen with prerequisites check
	mainMenu            tea.Model
	prerequisites       tea.Model // NEW: Prerequisites checking
	scanDirectories     tea.Model
	gitHubRepo          tea.Model
	sourceSelection     tea.Model // NEW: Select source Flutter project
	sourceConfig        tea.Model // NEW: Configure download location/name
	sourceDownload      tea.Model // NEW: Download source project
	dependencySelection tea.Model // RENAMED: Multi-select dependencies (was repoSelection)
	configuration       tea.Model
	confirmation        tea.Model
	execution           tea.Model
	results             tea.Model
	searchConfig        tea.Model // NEW: Configure search filters
	forceUpdate         tea.Model // NEW: Force update packages
	selfUpdate          tea.Model // NEW: Self-update Flutter-PM
	errorScreen         tea.Model

	// Shared application state
	SharedState *AppState

	// Performance optimization: cache warmer
	cacheWarmer *core.CacheWarmer
}

// AppState holds data that needs to be shared between screens
type AppState struct {
	// Source project information (the Flutter project being worked ON)
	SourceProject         *core.Project // The Flutter project we're modifying
	SourceProjectPath     string        // Path to the source project
	DetectedPubspecPath   string        // Detected local pubspec path
	DetectedProject       string        // Detected local project name
	LocalPubspecAvailable bool          // Whether local pubspec was found
	HasGitDeps            bool          // Whether project has git dependencies

	// Available source projects (for selection)
	AvailableSourceRepos []core.RepoCandidate // Available Flutter projects to work on

	// Dependencies (packages to ADD to the source project)
	AvailableDependencies []core.RepoCandidate // Available packages to add as dependencies
	SelectedDependencies  []core.RepoCandidate // Selected packages to add to pubspec

	// Package specifications (for dependency installation)
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
		currentScreen: ScreenSplash, // Start with splash screen
		SharedState:   sharedState,
		cacheWarmer:   cacheWarmer,
	}
}

// Init initializes the app model
func (m *AppModel) Init() tea.Cmd {
	// Start background cache warming for better performance
	m.cacheWarmer.Start()

	// Initialize the first screen (Splash Screen)
	m.splash = NewSplashScreenModel(m.cfg, m.logger, m.SharedState)
	return m.splash.Init()
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
	case ScreenSplash:
		if m.splash != nil {
			return m.splash.View()
		}
	case ScreenMainMenu:
		if m.mainMenu != nil {
			return m.mainMenu.View()
		}
	case ScreenPrerequisites:
		if m.prerequisites != nil {
			return m.prerequisites.View()
		}
	case ScreenScanDirectories:
		if m.scanDirectories != nil {
			return m.scanDirectories.View()
		}
	case ScreenGitHubRepo:
		if m.gitHubRepo != nil {
			return m.gitHubRepo.View()
		}
	case ScreenSourceSelection:
		if m.sourceSelection != nil {
			return m.sourceSelection.View()
		}
	case ScreenSourceConfig:
		if m.sourceConfig != nil {
			return m.sourceConfig.View()
		}
	case ScreenSourceDownload:
		if m.sourceDownload != nil {
			return m.sourceDownload.View()
		}
	case ScreenDependencySelection:
		if m.dependencySelection != nil {
			return m.dependencySelection.View()
		}
	case ScreenSearchConfig:
		if m.searchConfig != nil {
			return m.searchConfig.View()
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
	case ScreenForceUpdate:
		if m.forceUpdate != nil {
			return m.forceUpdate.View()
		}
	case ScreenSelfUpdate:
		if m.selfUpdate != nil {
			return m.selfUpdate.View()
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
	case ScreenSplash:
		if m.splash != nil {
			m.splash, cmd = m.splash.Update(msg)
		}
	case ScreenMainMenu:
		if m.mainMenu != nil {
			m.mainMenu, cmd = m.mainMenu.Update(msg)
		}
	case ScreenPrerequisites:
		if m.prerequisites != nil {
			m.prerequisites, cmd = m.prerequisites.Update(msg)
		}
	case ScreenScanDirectories:
		if m.scanDirectories != nil {
			m.scanDirectories, cmd = m.scanDirectories.Update(msg)
		}
	case ScreenGitHubRepo:
		if m.gitHubRepo != nil {
			m.gitHubRepo, cmd = m.gitHubRepo.Update(msg)
		}
	case ScreenSourceSelection:
		if m.sourceSelection != nil {
			m.sourceSelection, cmd = m.sourceSelection.Update(msg)
		}
	case ScreenSourceConfig:
		if m.sourceConfig != nil {
			m.sourceConfig, cmd = m.sourceConfig.Update(msg)
		}
	case ScreenSourceDownload:
		if m.sourceDownload != nil {
			m.sourceDownload, cmd = m.sourceDownload.Update(msg)
		}
	case ScreenDependencySelection:
		if m.dependencySelection != nil {
			m.dependencySelection, cmd = m.dependencySelection.Update(msg)
		}
	case ScreenSearchConfig:
		if m.searchConfig != nil {
			m.searchConfig, cmd = m.searchConfig.Update(msg)
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
	case ScreenForceUpdate:
		if m.forceUpdate != nil {
			m.forceUpdate, cmd = m.forceUpdate.Update(msg)
		}
	case ScreenSelfUpdate:
		if m.selfUpdate != nil {
			m.selfUpdate, cmd = m.selfUpdate.Update(msg)
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
	case ScreenSplash:
		if m.splash == nil {
			m.splash = NewSplashScreenModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.splash.Init()

	case ScreenMainMenu:
		if m.mainMenu == nil {
			m.mainMenu = NewMainMenuModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.mainMenu.Init()

	case ScreenPrerequisites:
		if m.prerequisites == nil {
			// Route to scan directories model for now (building on existing foundation)
			m.prerequisites = NewScanDirectoriesModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.prerequisites.Init()

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

	case ScreenSourceSelection:
		if m.sourceSelection == nil {
			m.sourceSelection = NewRepoSelectionModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.sourceSelection.Init()
	case ScreenSourceConfig:
		if m.sourceConfig == nil {
			m.sourceConfig = NewSourceConfigModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.sourceConfig.Init()
	case ScreenSourceDownload:
		if m.sourceDownload == nil {
			// Route to scan directories model for now (building on existing foundation)
			m.sourceDownload = NewScanDirectoriesModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.sourceDownload.Init()
	case ScreenDependencySelection:
		if m.dependencySelection == nil {
			m.dependencySelection = NewRepoSelectionModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.dependencySelection.Init()

	case ScreenSearchConfig:
		if m.searchConfig == nil {
			m.searchConfig = NewSearchConfigModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.searchConfig.Init()

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

	case ScreenForceUpdate:
		if m.forceUpdate == nil {
			// Route to execution model for now (building on existing foundation)
			m.forceUpdate = NewExecutionModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.forceUpdate.Init()

	case ScreenSelfUpdate:
		if m.selfUpdate == nil {
			// Route to execution model for now (building on existing foundation)
			m.selfUpdate = NewExecutionModel(m.cfg, m.logger, m.SharedState)
		}
		return m, m.selfUpdate.Init()

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
