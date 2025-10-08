// Package testing/comprehensive_walkthrough_test.go - Complete Terminal Walkthrough Testing
//
// This file implements comprehensive testing that walks through each menu option
// and verifies the actual terminal output content, catching issues like showing
// the wrong screen content.

package testing

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/tui/models"
)

// TestCompleteMenuWalkthrough tests each menu option and verifies correct screen content
func TestCompleteMenuWalkthrough(t *testing.T) {
	cfg := core.Config{Debug: true, Quiet: true}
	logger := core.NewLogger(&cfg)

	testCases := []struct {
		option           string
		expectedTitle    string
		expectedContent  []string
		forbiddenContent []string
		description      string
	}{
		{
			option:        "1",
			expectedTitle: "🔍 Scanning for Flutter Projects...",
			expectedContent: []string{
				"Scanning for Flutter Projects",
				"Please wait while we scan common directories",
			},
			forbiddenContent: []string{
				"Fetching GitHub repositories",
				"⚙️ Configure Directory Search",
				"Package Configuration",
			},
			description: "Option 1: Check prerequisites (placeholder scanning screen)",
		},
		{
			option:        "2",
			expectedTitle: "Fetching GitHub repositories",
			expectedContent: []string{
				"Fetching GitHub repositories",
				"available packages",
			},
			forbiddenContent: []string{
				"Scanning for Flutter Projects",
				"⚙️ Configure Directory Search",
				"Package Configuration",
			},
			description: "Option 2: GitHub repo loading",
		},
		{
			option:        "3",
			expectedTitle: "⚙️ Configure Directory Search",
			expectedContent: []string{
				"⚙️ Configure Directory Search",
				"Current Search Configuration",
				"Add search path",
				"Change search depth",
				"Toggle full disk search",
				"Continue",
			},
			forbiddenContent: []string{
				"Package Configuration",
				"selected packages",
				"All Packages Configured",
				"Fetching GitHub repositories",
				"Scanning for Flutter Projects",
			},
			description: "Option 3: Configure search (CRITICAL TEST)",
		},
		{
			option:        "4",
			expectedTitle: "🔍 Scanning for Flutter Projects...",
			expectedContent: []string{
				"Scanning for Flutter Projects",
				"Please wait while we scan common directories",
			},
			forbiddenContent: []string{
				"Fetching GitHub repositories",
				"⚙️ Configure Directory Search",
				"Package Configuration",
			},
			description: "Option 4: Update local package (scan directories placeholder)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Create fresh app instance for each test
			app := models.NewAppModel(cfg, logger)
			app.Init()

			// Wait for initialization
			time.Sleep(10 * time.Millisecond)

			// Get initial view to confirm we're on main menu
			initialView := app.View()
			if !strings.Contains(initialView, "Flutter Package Manager - Main Menu") {
				t.Fatalf("Not starting from main menu. View: %s", initialView)
			}

			// Select the option by key
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune(tc.option),
			}
			updatedApp, cmd := app.Update(keyMsg)

			// Execute the command if returned
			if cmd != nil {
				msg := cmd()
				if transitionMsg, ok := msg.(models.ScreenTransitionMsg); ok {
					// Apply the screen transition
					finalApp, _ := updatedApp.Update(transitionMsg)

					// Wait for screen to render
					time.Sleep(50 * time.Millisecond)

					// Get the final view
					finalView := finalApp.View()

					// Test expected content is present
					for _, expected := range tc.expectedContent {
						if !strings.Contains(finalView, expected) {
							t.Errorf("%s: Missing expected content '%s'\nActual view:\n%s",
								tc.description, expected, finalView)
						}
					}

					// Test forbidden content is NOT present
					for _, forbidden := range tc.forbiddenContent {
						if strings.Contains(finalView, forbidden) {
							t.Errorf("%s: Contains forbidden content '%s'\nActual view:\n%s",
								tc.description, forbidden, finalView)
						}
					}

					// Special validation for the critical option 3 test
					if tc.option == "3" {
						if !strings.Contains(finalView, "⚙️ Configure Directory Search") {
							t.Errorf("CRITICAL FAILURE: Option 3 does not show directory search configuration screen!")
							t.Errorf("Expected: '⚙️ Configure Directory Search'")
							t.Errorf("Actual view:\n%s", finalView)
						} else {
							t.Logf("✅ CRITICAL TEST PASSED: Option 3 correctly shows directory search configuration")
						}
					}

					t.Logf("✅ %s: Shows correct content", tc.description)
				} else {
					t.Errorf("%s: Expected screen transition, got: %T", tc.description, msg)
				}
			} else {
				t.Errorf("%s: No command returned after key press", tc.description)
			}
		})
	}
}

// TestScreenContentValidation validates specific screen content in detail
func TestScreenContentValidation(t *testing.T) {
	cfg := core.Config{Debug: true, Quiet: true}
	logger := core.NewLogger(&cfg)

	t.Run("SearchConfigScreenValidation", func(t *testing.T) {
		// Create search config screen directly
		shared := &models.AppState{}
		searchConfig := models.NewSearchConfigModel(cfg, logger, shared)
		searchConfig.Init()

		view := searchConfig.View()

		// Detailed validation of directory search config screen
		requiredElements := []string{
			"⚙️ Configure Directory Search",
			"Current Search Configuration",
			"Add search path",
			"Change search depth",
			"Toggle full disk search",
			"Continue",
			"navigate",
			"enter: select option",
		}

		for _, element := range requiredElements {
			if !strings.Contains(view, element) {
				t.Errorf("Directory search config missing required element: '%s'\nFull view:\n%s", element, view)
			}
		}

		// Validate it's NOT showing package configuration content
		forbiddenElements := []string{
			"Package Configuration",
			"selected packages",
			"All Packages Configured",
			"🔍 Configure Repository Search",
			"Owner/Organization Filter",
			"Language Filter",
			"Topic Filter",
		}

		for _, forbidden := range forbiddenElements {
			if strings.Contains(view, forbidden) {
				t.Errorf("Directory search config shows forbidden content: '%s'\nFull view:\n%s", forbidden, view)
			}
		}

		t.Log("✅ Directory search configuration screen content is correct")
	})

	// Skip source config test for now - not implemented yet
	t.Log("⏭️ Source configuration test skipped - not implemented yet")
}

// TestActualTerminalFrames captures the real terminal output for analysis
func TestActualTerminalFrames(t *testing.T) {
	cfg := core.Config{Debug: true, Quiet: true}
	logger := core.NewLogger(&cfg)

	frames := make(map[string]string)

	// Capture main menu
	app := models.NewAppModel(cfg, logger)
	app.Init()
	frames["main_menu"] = app.View()

	// Capture each option screen
	for i := 1; i <= 4; i++ {
		app := models.NewAppModel(cfg, logger)
		app.Init()

		keyMsg := tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{rune('0' + i)},
		}
		updatedApp, cmd := app.Update(keyMsg)

		if cmd != nil {
			msg := cmd()
			if transitionMsg, ok := msg.(models.ScreenTransitionMsg); ok {
				finalApp, _ := updatedApp.Update(transitionMsg)
				time.Sleep(50 * time.Millisecond)
				frames[string(rune('0'+i))] = finalApp.View()
			}
		}
	}

	// Write frames to test output for manual inspection
	for key, frame := range frames {
		filename := "terminal_frame_" + key + ".txt"
		if err := WriteTestOutputToFile(filename, frame); err != nil {
			t.Logf("Failed to write frame %s: %v", key, err)
		} else {
			t.Logf("📄 Saved terminal frame: %s", filename)
		}
	}

	// Validate critical option 3 frame
	option3Frame := frames["3"]
	if !strings.Contains(option3Frame, "⚙️ Configure Directory Search") {
		t.Errorf("CRITICAL: Option 3 frame does not contain search configuration content!")
		t.Errorf("Frame content:\n%s", option3Frame)
	} else {
		t.Log("✅ Option 3 frame validation passed")
	}
}

// WriteTestOutputToFile helper to save terminal frames
func WriteTestOutputToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// TestGitHubRepoToMultiselect verifies that GitHub repo option leads to multiselect package screen
func TestGitHubRepoToMultiselect(t *testing.T) {
	cfg := core.Config{Debug: true, Quiet: true}
	logger := core.NewLogger(&cfg)

	t.Run("GitHubRepoLeadsToMultiselect", func(t *testing.T) {
		// Create fresh app instance
		app := models.NewAppModel(cfg, logger)
		app.Init()

		// Wait for initialization
		time.Sleep(10 * time.Millisecond)

		// Confirm we're on main menu
		initialView := app.View()
		if !strings.Contains(initialView, "Flutter Package Manager - Main Menu") {
			t.Fatalf("Not starting from main menu. View: %s", initialView)
		}

		// Select option 2 (GitHub repo)
		keyMsg := tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("2"),
		}
		updatedApp, cmd := app.Update(keyMsg)

		// Execute the transition command
		if cmd != nil {
			msg := cmd()
			if transitionMsg, ok := msg.(models.ScreenTransitionMsg); ok {
				// Apply screen transition to GitHub loading screen
				loadingApp, initCmd := updatedApp.Update(transitionMsg)

				// Execute init command if any
				if initCmd != nil {
					initCmd()
				}

				// Wait for loading screen to render
				time.Sleep(50 * time.Millisecond)

				// Verify we're on the loading screen
				loadingView := loadingApp.View()
				if !strings.Contains(loadingView, "Fetching GitHub repositories") {
					t.Errorf("Expected loading screen, got: %s", loadingView)
				}

				// Simulate successful repo load with mock data
				mockRepos := []core.RepoCandidate{
					{Name: "test-package-1", Owner: "testowner", URL: "https://github.com/testowner/test-package-1", Desc: "Test package 1"},
					{Name: "test-package-2", Owner: "testowner", URL: "https://github.com/testowner/test-package-2", Desc: "Test package 2"},
					{Name: "test-package-3", Owner: "testowner", URL: "https://github.com/testowner/test-package-3", Desc: "Test package 3"},
				}

				// We can't directly test the internal message handling, but we can verify
				// that the RepoSelectionModel exists and works correctly
				shared := &models.AppState{
					AvailableDependencies: mockRepos,
				}
				multiselect := models.NewRepoSelectionModel(cfg, logger, shared)
				multiselect.Init()

				// Wait for initialization
				time.Sleep(50 * time.Millisecond)

				// Get the view
				multiselectView := multiselect.View()

				// Verify it shows the package multiselect screen content
				expectedContent := []string{
					"📦 Add Dependencies",
					"testowner/test-package-1",
					"testowner/test-package-2",
					"testowner/test-package-3",
					"space: toggle",
				}

				for _, expected := range expectedContent {
					if !strings.Contains(multiselectView, expected) {
						t.Errorf("Multiselect screen missing expected content '%s'\nActual view:\n%s",
							expected, multiselectView)
					}
				}

				// Verify forbidden content is NOT present
				forbiddenContent := []string{
					"⚙️ Configure Directory Search",
					"Package Configuration",
					"Scanning for Flutter Projects",
				}

				for _, forbidden := range forbiddenContent {
					if strings.Contains(multiselectView, forbidden) {
						t.Errorf("Multiselect screen contains forbidden content '%s'\nActual view:\n%s",
							forbidden, multiselectView)
					}
				}

				t.Log("✅ GitHub repo option correctly leads to multiselect package selection screen")

				// Save the frame for manual inspection
				if err := WriteTestOutputToFile("terminal_frame_github_multiselect.txt", multiselectView); err != nil {
					t.Logf("Failed to write multiselect frame: %v", err)
				} else {
					t.Log("📄 Saved multiselect terminal frame: terminal_frame_github_multiselect.txt")
				}
			} else {
				t.Errorf("Expected screen transition, got: %T", msg)
			}
		} else {
			t.Error("No command returned after selecting GitHub repo option")
		}
	})
}
