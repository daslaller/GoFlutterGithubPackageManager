// Package models/configuration_test.go - Tests for Configuration Model
//
// This file contains tests to prevent crashes during package configuration.

package models

import (
	"testing"

	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// TestConfigurationSafety tests edge cases that could cause crashes
func TestConfigurationSafety(t *testing.T) {
	cfg := core.Config{}
	logger := &core.Logger{}

	// Test with empty selected dependencies - should not crash
	shared := &AppState{
		SelectedDependencies: []core.RepoCandidate{}, // Empty
	}

	model := NewConfigurationModel(cfg, logger, shared)

	// Init should not crash with empty repos
	model.Init()

	// Should be marked as complete
	if !model.complete {
		t.Error("Expected model to be marked complete with empty repos")
	}

	// focusCurrentInput should not crash
	model.focusCurrentInput()

	// generatePackageSpecs should not crash
	model.generatePackageSpecs()

	// Should have no package specs
	if len(model.packageSpecs) != 0 {
		t.Errorf("Expected 0 package specs, got %d", len(model.packageSpecs))
	}
}

// TestConfigurationWithValidData tests normal operation
func TestConfigurationWithValidData(t *testing.T) {
	cfg := core.Config{}
	logger := &core.Logger{}

	// Create test repos
	testRepos := []core.RepoCandidate{
		{Owner: "user1", Name: "repo1", URL: "https://github.com/user1/repo1"},
	}

	shared := &AppState{
		SelectedDependencies: testRepos,
	}

	model := NewConfigurationModel(cfg, logger, shared)

	// Init should create inputs
	model.Init()

	// Should have 3 inputs (name, ref, subdir)
	expectedInputs := len(testRepos) * 3
	if len(model.inputs) != expectedInputs {
		t.Errorf("Expected %d inputs, got %d", expectedInputs, len(model.inputs))
	}

	// Should not be marked as complete
	if model.complete {
		t.Error("Expected model to not be marked complete with valid repos")
	}

	// generatePackageSpecs should work
	model.generatePackageSpecs()

	// Should have 1 package spec
	if len(model.packageSpecs) != 1 {
		t.Errorf("Expected 1 package spec, got %d", len(model.packageSpecs))
	}

	// Should have correct default values
	if model.packageSpecs[0].Name != "repo1" {
		t.Errorf("Expected package name 'repo1', got '%s'", model.packageSpecs[0].Name)
	}

	if model.packageSpecs[0].Ref != "main" {
		t.Errorf("Expected ref 'main', got '%s'", model.packageSpecs[0].Ref)
	}
}

// TestConfigurationInputHandling tests input manipulation
func TestConfigurationInputHandling(t *testing.T) {
	cfg := core.Config{}
	logger := &core.Logger{}

	testRepos := []core.RepoCandidate{
		{Owner: "user1", Name: "repo1", URL: "https://github.com/user1/repo1"},
	}

	shared := &AppState{
		SelectedDependencies: testRepos,
	}

	model := NewConfigurationModel(cfg, logger, shared)
	model.Init()

	// Test focusing inputs - should not crash
	model.currentRepo = 0
	model.currentField = 0
	model.focusCurrentInput()

	// Test with invalid indices - should not crash
	model.currentRepo = 10 // Invalid
	model.focusCurrentInput()

	model.currentRepo = 0
	model.currentField = 10 // Invalid
	model.focusCurrentInput()
}
