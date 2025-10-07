// Package tui - New Multi-Model Architecture Entry Point
//
// This file provides the main entry point for the new multimodel TUI architecture
// where each screen is its own model, coordinated by the AppModel.

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/tui/models"
)

// RunNew starts the new multimodel TUI application
func RunNew(cfg core.Config, logger *core.Logger) error {
	// Create the main app coordinator
	app := models.NewAppModel(cfg, logger)

	// Start the bubbletea program
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunLegacy runs the old single-model implementation as fallback
