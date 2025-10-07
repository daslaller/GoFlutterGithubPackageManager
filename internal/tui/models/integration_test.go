// Package models/integration_test.go - Integration tests for screen transitions
//
// This file tests the complete flow between screens to prevent crashes
// during normal user workflows.

package models

import (
	"testing"

	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// TestRepoSelectionToConfiguration tests the complete flow from repo selection to configuration
func TestRepoSelectionToConfiguration(t *testing.T) {
	cfg := core.Config{}
	logger := &core.Logger{}

	// Create test repos
	testRepos := []core.RepoCandidate{
		{Owner: "user1", Name: "repo1", URL: "https://github.com/user1/repo1"},
		{Owner: "user2", Name: "repo2", URL: "https://github.com/user2/repo2"},
	}

	shared := &AppState{
		AvailableDependencies: testRepos,
		SelectedDependencies:  []core.RepoCandidate{}, // Start empty
	}

	// Test repo selection
	repoModel := NewRepoSelectionModel(cfg, logger, shared)

	// Select a repository using the multiselect delegate
	repoModel.delegate.toggleSelection(0)
	repoModel.finalizeSelection()

	// Verify selection worked
	if len(shared.SelectedDependencies) != 1 {
		t.Fatalf("Expected 1 selected repo, got %d", len(shared.SelectedDependencies))
	}

	if shared.SelectedDependencies[0].Name != "repo1" {
		t.Errorf("Expected repo1, got %s", shared.SelectedDependencies[0].Name)
	}

	// Test configuration screen with selected repo
	configModel := NewConfigurationModel(cfg, logger, shared)
	configModel.Init()

	// Should not be marked complete
	if configModel.complete {
		t.Error("Configuration should not be complete with valid selection")
	}

	// Should have inputs for the selected repo
	expectedInputs := len(shared.SelectedDependencies) * 3
	if len(configModel.inputs) != expectedInputs {
		t.Errorf("Expected %d inputs, got %d", expectedInputs, len(configModel.inputs))
	}

	// Test input focusing - should not crash
	configModel.focusCurrentInput()

	// Generate package specs - should not crash
	configModel.generatePackageSpecs()

	// Should have generated specs
	if len(configModel.packageSpecs) != 1 {
		t.Errorf("Expected 1 package spec, got %d", len(configModel.packageSpecs))
	}
}

// TestEmptyRepoFlow tests the flow with no repositories selected
func TestEmptyRepoFlow(t *testing.T) {
	cfg := core.Config{}
	logger := &core.Logger{}

	shared := &AppState{
		AvailableDependencies: []core.RepoCandidate{},
		SelectedDependencies:  []core.RepoCandidate{},
	}

	// Test repo selection with empty repos
	repoModel := NewRepoSelectionModel(cfg, logger, shared)

	// Try to select invalid index - should not crash
	repoModel.delegate.toggleSelection(0)
	repoModel.finalizeSelection()

	// Should still have no selected repos
	if len(shared.SelectedDependencies) != 0 {
		t.Errorf("Expected 0 selected repos, got %d", len(shared.SelectedDependencies))
	}

	// Test configuration with no selected repos - should not crash
	configModel := NewConfigurationModel(cfg, logger, shared)
	configModel.Init()

	// Should be marked complete (skip configuration)
	if !configModel.complete {
		t.Error("Configuration should be complete with no selected repos")
	}

	// Should have no inputs
	if len(configModel.inputs) != 0 {
		t.Errorf("Expected 0 inputs, got %d", len(configModel.inputs))
	}

	// These operations should not crash
	configModel.focusCurrentInput()
	configModel.generatePackageSpecs()

	// Should have no package specs
	if len(configModel.packageSpecs) != 0 {
		t.Errorf("Expected 0 package specs, got %d", len(configModel.packageSpecs))
	}
}
