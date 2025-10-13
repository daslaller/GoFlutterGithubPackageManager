// Package models/self_update_model.go - Self-Update Screen
//
// This file implements the self-update screen that checks for updates,
// shows available updates, and performs the update process.

package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// SelfUpdateModel handles the self-update screen
type SelfUpdateModel struct {
	cfg    core.Config
	logger *core.Logger
	shared *AppState

	// Update state
	checking     bool
	updateInfo   core.UpdateInfo
	updating     bool
	updateDone   bool
	updateError  error
	progress     string

	// Styles
	titleStyle   lipgloss.Style
	successStyle lipgloss.Style
	errorStyle   lipgloss.Style
	infoStyle    lipgloss.Style
	helpStyle    lipgloss.Style
}

// updateCheckMsg is sent when update check completes
type updateCheckMsg struct {
	info core.UpdateInfo
	err  error
}

// updateCompleteMsg is sent when update completes
type updateCompleteMsg struct {
	err error
}

// NewSelfUpdateModel creates a new self-update model
func NewSelfUpdateModel(cfg core.Config, logger *core.Logger, shared *AppState) *SelfUpdateModel {
	return &SelfUpdateModel{
		cfg:      cfg,
		logger:   logger,
		shared:   shared,
		checking: true,

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0EA5E9")).
			Bold(true),

		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),

		infoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			Italic(true),
	}
}

// Init initializes the self-update model
func (m *SelfUpdateModel) Init() tea.Cmd {
	return m.checkForUpdates()
}

// Update handles messages for the self-update model
func (m *SelfUpdateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			if m.checking || m.updating {
				// Don't allow quitting during update
				return m, nil
			}
			return m, TransitionToScreen(ScreenMainMenu)

		case "y", "Y":
			// User confirmed update
			if !m.checking && m.updateInfo.Available && !m.updating && !m.updateDone {
				m.updating = true
				m.progress = "Downloading update..."
				return m, m.performUpdate()
			}
			return m, nil

		case "n", "N":
			// User declined update
			if !m.checking && !m.updating {
				return m, TransitionToScreen(ScreenMainMenu)
			}
			return m, nil

		case "enter":
			// After update completes, return to menu
			if m.updateDone {
				return m, TransitionToScreen(ScreenMainMenu)
			}
			return m, nil
		}

	case updateCheckMsg:
		m.checking = false
		m.updateInfo = msg.info
		m.updateError = msg.err

		if msg.err != nil {
			m.logger.Error("selfupdate", msg.err)
		} else if msg.info.Available {
			m.logger.Info("selfupdate", fmt.Sprintf("Update available: %s -> %s", msg.info.CurrentVersion, msg.info.LatestVersion))
		} else {
			m.logger.Info("selfupdate", "Already on latest version")
		}
		return m, nil

	case updateCompleteMsg:
		m.updating = false
		m.updateDone = true
		m.updateError = msg.err

		if msg.err != nil {
			m.logger.Error("selfupdate", msg.err)
		} else {
			m.logger.Info("selfupdate", "Update completed successfully")
		}
		return m, nil

	case ScreenTransitionMsg:
		// Forward transition message
		return m, func() tea.Msg { return msg }

	case tea.WindowSizeMsg:
		// Handle window resize gracefully
		return m, nil
	}

	return m, nil
}

// View renders the self-update screen
func (m *SelfUpdateModel) View() string {
	var b strings.Builder

	// Header with warm amber theme
	header := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F59E0B")).
		Foreground(lipgloss.Color("#F59E0B")).
		Padding(1, 2).
		Align(lipgloss.Center).
		Width(62).
		Bold(true).
		Render("🔄 Flutter-PM Self-Update")

	b.WriteString(header + "\n\n")

	if m.checking {
		// Checking for updates
		b.WriteString(m.infoStyle.Render("🔍 Checking for updates...") + "\n\n")
	} else if m.updateError != nil && !m.updating && !m.updateDone {
		// Error checking for updates
		b.WriteString(m.errorStyle.Render(fmt.Sprintf("❌ Error: %s", m.updateError.Error())) + "\n\n")
		b.WriteString(m.helpStyle.Render("Press 'q' to return to main menu") + "\n")
	} else if m.updating {
		// Performing update
		b.WriteString(m.titleStyle.Render("📥 Updating...") + "\n\n")
		b.WriteString(m.infoStyle.Render(m.progress) + "\n\n")
		b.WriteString(m.helpStyle.Render("Please wait, do not close this window...") + "\n")
	} else if m.updateDone {
		// Update complete
		if m.updateError != nil {
			b.WriteString(m.errorStyle.Render(fmt.Sprintf("❌ Update failed: %s", m.updateError.Error())) + "\n\n")
		} else {
			b.WriteString(m.successStyle.Render("✅ Update completed successfully!") + "\n\n")
			b.WriteString(m.infoStyle.Render(fmt.Sprintf("Updated to version: %s", m.updateInfo.LatestVersion)) + "\n\n")
			b.WriteString(m.titleStyle.Render("⚠️  Please restart flutter-pm to use the new version") + "\n\n")
		}
		b.WriteString(m.helpStyle.Render("Press 'enter' to return to main menu") + "\n")
	} else if m.updateInfo.Available {
		// Update available - ask user
		b.WriteString(m.titleStyle.Render("✨ Update Available!") + "\n\n")
		b.WriteString(m.infoStyle.Render(fmt.Sprintf("Current version: %s", m.updateInfo.CurrentVersion)) + "\n")
		b.WriteString(m.infoStyle.Render(fmt.Sprintf("Latest version:  %s", m.updateInfo.LatestVersion)) + "\n\n")

		// Show release notes if available
		if m.updateInfo.ReleaseNotes != "" {
			notes := m.updateInfo.ReleaseNotes
			if len(notes) > 300 {
				notes = notes[:300] + "..."
			}
			b.WriteString(m.titleStyle.Render("📝 Release Notes:") + "\n")
			b.WriteString(m.infoStyle.Render(notes) + "\n\n")
		}

		b.WriteString(m.successStyle.Render("Would you like to update now?") + "\n\n")
		b.WriteString(m.helpStyle.Render("y: yes, update • n: no, return to menu") + "\n")
	} else {
		// Already up to date
		b.WriteString(m.successStyle.Render("✅ You're already on the latest version!") + "\n\n")
		b.WriteString(m.infoStyle.Render(fmt.Sprintf("Current version: %s", m.updateInfo.CurrentVersion)) + "\n\n")
		b.WriteString(m.helpStyle.Render("Press 'q' to return to main menu") + "\n")
	}

	return b.String()
}

// checkForUpdates runs the update check in the background
func (m *SelfUpdateModel) checkForUpdates() tea.Cmd {
	return func() tea.Msg {
		info, err := core.CheckForUpdates(m.logger)
		return updateCheckMsg{info: info, err: err}
	}
}

// performUpdate performs the update in the background
func (m *SelfUpdateModel) performUpdate() tea.Cmd {
	return func() tea.Msg {
		err := core.PerformUpdate(m.updateInfo, m.logger)
		return updateCompleteMsg{err: err}
	}
}
