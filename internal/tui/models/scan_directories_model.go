// Package models/scan_directories_model.go - Directory Scanning Screen Model
//
// This file implements the directory scanning screen where the application
// scans for Flutter projects in common directories.

package models

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ScanDirectoriesModel handles directory scanning for Flutter projects
type ScanDirectoriesModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// State
	scanning      bool
	projects      []core.Project
	complete      bool
	quitting      bool
	selectedIndex int // Currently selected project index

	// Styles
	headerStyle   lipgloss.Style
	successStyle  lipgloss.Style
	errorStyle    lipgloss.Style
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
}

// NewScanDirectoriesModel creates a new scan directories model
func NewScanDirectoriesModel(cfg core.Config, logger *core.Logger, shared *AppState) *ScanDirectoriesModel {
	return &ScanDirectoriesModel{
		cfg:           cfg,
		logger:        logger,
		shared:        shared,
		scanning:      true,
		selectedIndex: 0,

		// Styles
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).
			Bold(true),

		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0EA5E9")).
			Padding(0, 1),

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
}

// scanCompleteMsg is sent when scanning is complete
type scanCompleteMsg struct {
	projects []core.Project
	err      error
}

// Init initializes the scan directories screen
func (m *ScanDirectoriesModel) Init() tea.Cmd {
	return m.scanForProjects()
}

// Update handles messages for directory scanning
func (m *ScanDirectoriesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.scanning {
			// Don't handle keys while scanning
			return m, nil
		}
		return m.handleKeys(msg)

	case scanCompleteMsg:
		m.scanning = false
		if msg.err != nil {
			m.logger.Error("scan_directories", fmt.Errorf("failed to scan directories: %w", msg.err))
			// Show error and allow return to main menu
			m.complete = true
			return m, nil
		}

		m.projects = msg.projects
		m.shared.SourceProject = nil // Will be set if user selects one
		m.complete = true

		m.logger.Info("scan_directories", fmt.Sprintf("Found %d Flutter projects", len(msg.projects)))
		return m, nil

	case tea.WindowSizeMsg:
		// Handle window resize gracefully
		return m, nil
	}

	return m, nil
}

// View renders the scan directories screen
func (m *ScanDirectoriesModel) View() string {
	if m.quitting {
		return ""
	}

	if m.scanning {
		return m.headerStyle.Render("🔍 Scanning for Flutter Projects...") + "\n\nPlease wait while we scan common directories for Flutter projects.\n\n"
	}

	if len(m.projects) == 0 {
		return m.errorStyle.Render("❌ No Flutter Projects Found") + "\n\nNo Flutter projects were found in common directories.\n\nPress Enter to return to main menu or Q to quit."
	}

	var content string

	// Single project - auto-select it
	if len(m.projects) == 1 {
		content = m.successStyle.Render("✅ Found 1 Flutter Project") + "\n\n"
		content += fmt.Sprintf("  %s\n\n", m.projects[0].Path)
		content += "Press Enter to use this project or Q to return to main menu"
		return content
	}

	// Multiple projects - let user select
	content = m.successStyle.Render(fmt.Sprintf("✅ Found %d Flutter Projects", len(m.projects))) + "\n\n"

	for i, project := range m.projects {
		projectText := fmt.Sprintf("%d. %s", i+1, project.Path)
		if i == m.selectedIndex {
			content += m.selectedStyle.Render("▶ "+projectText) + "\n"
		} else {
			content += m.normalStyle.Render("  "+projectText) + "\n"
		}

		if i >= 9 { // Limit display to first 10
			content += fmt.Sprintf("... and %d more\n", len(m.projects)-10)
			break
		}
	}

	content += "\n↑/↓ or j/k: navigate • Enter: select project • Q: back to menu"
	return content
}

// handleKeys handles keyboard input
func (m *ScanDirectoriesModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, TransitionToScreen(ScreenMainMenu)

	case "up", "k":
		if len(m.projects) > 1 {
			m.selectedIndex--
			if m.selectedIndex < 0 {
				m.selectedIndex = len(m.projects) - 1
			}
		}
		return m, nil

	case "down", "j":
		if len(m.projects) > 1 {
			m.selectedIndex++
			if m.selectedIndex >= len(m.projects) {
				m.selectedIndex = 0
			}
		}
		return m, nil

	case "enter":
		// Select the current project and save to shared state
		if len(m.projects) > 0 && m.selectedIndex >= 0 && m.selectedIndex < len(m.projects) {
			selectedProject := m.projects[m.selectedIndex]
			m.shared.SourceProject = &selectedProject
			m.shared.SourceProjectPath = selectedProject.Path
			m.shared.DetectedPubspecPath = selectedProject.PubspecPath
			m.shared.DetectedProject = selectedProject.Name
			m.shared.LocalPubspecAvailable = true

			m.logger.Info("scan_directories", fmt.Sprintf("Selected project: %s at %s", selectedProject.Name, selectedProject.Path))

			// Return to main menu with selected project
			return m, TransitionToScreen(ScreenMainMenu)
		}
		return m, nil
	}

	return m, nil
}

// scanForProjects scans for Flutter projects in common directories
func (m *ScanDirectoriesModel) scanForProjects() tea.Cmd {
	return func() tea.Msg {
		m.logger.Info("scan_directories", "Starting directory scan for Flutter projects")

		// Check if local project was detected - use that first
		if m.shared.LocalPubspecAvailable && m.shared.SourceProjectPath != "" {
			m.logger.Info("scan_directories", fmt.Sprintf("Using detected local project: %s", m.shared.DetectedProject))
			project := core.Project{
				Path:        m.shared.SourceProjectPath,
				Name:        m.shared.DetectedProject,
				PubspecPath: m.shared.DetectedPubspecPath,
			}
			return scanCompleteMsg{
				projects: []core.Project{project},
				err:      nil,
			}
		}

		// Otherwise, scan for projects within +-3 levels from current directory
		if project, err := core.FindPubspecNearCurrent(); err == nil {
			m.logger.Info("scan_directories", fmt.Sprintf("Found project within +-3 levels: %s", project.Name))
			return scanCompleteMsg{
				projects: []core.Project{*project},
				err:      nil,
			}
		}

		// If nothing found nearby, scan common roots
		m.logger.Info("scan_directories", "Scanning common development directories")
		projects, err := core.ScanCommonRoots()
		if err != nil {
			m.logger.Error("scan_directories", fmt.Errorf("scan failed: %w", err))
			return scanCompleteMsg{
				projects: nil,
				err:      err,
			}
		}

		m.logger.Info("scan_directories", fmt.Sprintf("Scan complete: found %d projects", len(projects)))
		return scanCompleteMsg{
			projects: projects,
			err:      nil,
		}
	}
}
