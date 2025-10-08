// Package models/repo_selection_test.go - Tests for Repository Selection Model
//
// This file contains tests to prevent crashes and ensure robust behavior
// during repository selection operations.

package models

import (
	"testing"

	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// TestRepoSelectionSafety tests edge cases that could cause crashes
func TestRepoSelectionSafety(t *testing.T) {
	cfg := core.Config{}
	logger := &core.Logger{}
	shared := &AppState{
		AvailableDependencies: []core.RepoCandidate{}, // Empty repos list
	}

	model := NewRepoSelectionModel(cfg, logger, shared)

	// Test selection with empty repos list - should not crash
	model.delegate.setSelected(0)
	model.finalizeSelection()

	// Should have no selected repos
	if len(shared.SelectedDependencies) != 0 {
		t.Errorf("Expected 0 selected repos, got %d", len(shared.SelectedDependencies))
	}

	// Test selection with invalid index - should not crash
	model.delegate.setSelected(-1)
	model.finalizeSelection()

	// Should still have no selected repos
	if len(shared.SelectedDependencies) != 0 {
		t.Errorf("Expected 0 selected repos after invalid selection, got %d", len(shared.SelectedDependencies))
	}
}

// TestRepoSelectionWithValidData tests normal operation
func TestRepoSelectionWithValidData(t *testing.T) {
	cfg := core.Config{}
	logger := &core.Logger{}

	// Create test repos
	testRepos := []core.RepoCandidate{
		{Owner: "user1", Name: "repo1", URL: "https://github.com/user1/repo1"},
		{Owner: "user2", Name: "repo2", URL: "https://github.com/user2/repo2"},
	}

	shared := &AppState{
		AvailableDependencies: testRepos,
	}

	model := NewRepoSelectionModel(cfg, logger, shared)

	// Test valid selection
	model.delegate.setSelected(0)
	model.finalizeSelection()

	// Should have exactly 1 selected repo
	if len(shared.SelectedDependencies) != 1 {
		t.Errorf("Expected 1 selected repo, got %d", len(shared.SelectedDependencies))
	}

	// Should be the correct repo
	if shared.SelectedDependencies[0].Name != "repo1" {
		t.Errorf("Expected repo1, got %s", shared.SelectedDependencies[0].Name)
	}

	// Test selecting different repo
	model.delegate.setSelected(1)
	model.finalizeSelection()

	// Should still have exactly 1 selected repo
	if len(shared.SelectedDependencies) != 1 {
		t.Errorf("Expected 1 selected repo, got %d", len(shared.SelectedDependencies))
	}

	// Should be the new repo
	if shared.SelectedDependencies[0].Name != "repo2" {
		t.Errorf("Expected repo2, got %s", shared.SelectedDependencies[0].Name)
	}
}
