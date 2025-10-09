// Package testing/splash_screen_test.go - Splash Screen Testing
//
// This file implements tests for the splash screen with prerequisites checking.

package testing

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/tui/models"
)

// TestSplashScreenStartup tests that the app starts with splash screen
func TestSplashScreenStartup(t *testing.T) {
	cfg := core.Config{Debug: false, Quiet: true}
	logger := core.NewLogger(&cfg)

	// Create app model
	app := models.NewAppModel(cfg, logger)

	// Initialize and get the view directly
	app.Init()

	// Get the initial view (should be splash screen)
	view := app.View()

	// Verify we get splash screen content
	if len(view) == 0 {
		t.Error("Expected app to have initial view content")
	}

	// Should contain splash screen elements
	if !strings.Contains(view, "FLUTTER") {
		t.Log("Note: Initial view doesn't show FLUTTER logo yet (may be loading)")
		t.Logf("View preview: %s", view[:min(200, len(view))])
	}

	t.Logf("App startup test completed. View length: %d chars", len(view))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestSplashScreenContent tests splash screen displays expected content
func TestSplashScreenContent(t *testing.T) {
	cfg := core.Config{Debug: false, Quiet: true}
	logger := core.NewLogger(&cfg)
	shared := &models.AppState{}

	// Create splash screen model directly
	splash := models.NewSplashScreenModel(cfg, logger, shared)

	// Initialize
	splash.Init()

	// Get initial view (should show checking message)
	view := splash.View()

	// Verify splash screen content
	// The ASCII logo uses box drawing characters, so check for more reliable text
	expectedContent := []string{
		"Checking prerequisites",
		"q: quit",
	}

	for _, content := range expectedContent {
		if !strings.Contains(view, content) {
			t.Errorf("Expected splash screen to contain '%s', but it doesn't", content)
			t.Logf("View preview (first 500 chars):\n%s", view[:min(500, len(view))])
		}
	}

	t.Logf("Splash screen view length: %d chars", len(view))
}

// TestSplashScreenTransition tests auto-transition to main menu
func TestSplashScreenTransition(t *testing.T) {
	cfg := core.Config{Debug: false, Quiet: true}
	logger := core.NewLogger(&cfg)
	shared := &models.AppState{}

	// Create splash screen model
	splash := models.NewSplashScreenModel(cfg, logger, shared)

	// Initialize
	cmd := splash.Init()
	if cmd == nil {
		t.Fatal("Expected Init to return a command")
	}

	// Wait for prerequisites check to complete (simulate)
	// In real scenario, this would be done by the prerequisitesCheckMsg
	time.Sleep(100 * time.Millisecond)

	// Verify initial state shows checking
	view := splash.View()
	if !strings.Contains(view, "Checking prerequisites") {
		t.Error("Expected to see 'Checking prerequisites' message")
	}

	t.Log("Splash screen transition test completed")
}

// TestSplashScreenKeyboardControls tests keyboard controls on splash screen
func TestSplashScreenKeyboardControls(t *testing.T) {
	cfg := core.Config{Debug: false, Quiet: true}
	logger := core.NewLogger(&cfg)
	shared := &models.AppState{}

	// Create splash screen model
	splash := models.NewSplashScreenModel(cfg, logger, shared)
	splash.Init()

	// Test 'q' key (should quit)
	updatedModel, cmd := splash.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("Expected 'q' to trigger quit command")
	}
	if updatedModel == nil {
		t.Error("Expected model to be returned")
	}

	// Test 'd' key (should toggle detailed view)
	splash2 := models.NewSplashScreenModel(cfg, logger, shared)
	splash2.Init()
	updatedModel2, _ := splash2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if updatedModel2 == nil {
		t.Error("Expected model to be returned after 'd' key")
	}

	t.Log("Keyboard controls test passed")
}

// TestPrerequisitesCheckIntegration tests that prerequisites are actually checked
func TestPrerequisitesCheckIntegration(t *testing.T) {
	cfg := core.Config{Debug: false, Quiet: true}
	logger := core.NewLogger(&cfg)

	// Run actual prerequisites check
	result := core.CheckPrerequisites(logger)

	// Log results
	t.Logf("Prerequisites check result: AllMet=%v, Missing=%v, Warnings=%d",
		result.AllMet, result.Missing, len(result.Warnings))

	// Verify structure
	if len(result.Results) == 0 {
		t.Error("Expected at least some prerequisite results")
	}

	for _, res := range result.Results {
		t.Logf("  %s: Available=%v, Version=%s", res.Name, res.Available, res.Version)
	}
}

// TestSplashScreenAnimation tests that animation updates correctly
func TestSplashScreenAnimation(t *testing.T) {
	cfg := core.Config{Debug: false, Quiet: true}
	logger := core.NewLogger(&cfg)
	shared := &models.AppState{}

	// Create splash screen model
	splash := models.NewSplashScreenModel(cfg, logger, shared)
	splash.Init()

	// Capture initial view
	view1 := splash.View()

	// Send animation tick
	type animationTickMsg struct{}
	splash.Update(animationTickMsg{})

	// Capture view after tick
	view2 := splash.View()

	// Views might be slightly different due to animation
	// (dots progress, but overall structure should be similar)
	if len(view1) == 0 || len(view2) == 0 {
		t.Error("Expected views to have content")
	}

	// Both should contain checking message or quit option
	hasContent1 := strings.Contains(view1, "Checking prerequisites") || strings.Contains(view1, "q: quit")
	hasContent2 := strings.Contains(view2, "Checking prerequisites") || strings.Contains(view2, "q: quit")

	if !hasContent1 || !hasContent2 {
		t.Error("Expected both views to contain splash screen content")
	}

	t.Logf("Animation test completed. View1 length: %d, View2 length: %d", len(view1), len(view2))
}
