// Package core/types.go - Core Data Structures and Type Definitions
//
// This file defines all the fundamental data structures used throughout the
// Flutter Package Manager application. These types provide a consistent
// interface for representing projects, repositories, package specifications,
// results, and workflow states.
//
// Key data structures:
// - Project: Represents a Flutter/Dart project with pubspec.yaml
// - RepoCandidate: GitHub repository that can be added as dependency
// - PkgSpec: Package specification for git dependencies
// - ActionResult: Standardized result format for all operations
// - Reco: Smart recommendations for improvements
// - Step: TUI workflow state enumeration
//
// These types ensure type safety and provide a clear contract between
// different modules (core business logic, TUI, CLI commands).

package core

import (
	"fmt"
	"time"
)

// Project represents a Flutter/Dart project
type Project struct {
	Path        string `json:"path"`
	PubspecPath string `json:"pubspec_path"`
	Name        string `json:"name,omitempty"`
}

// RepoCandidate represents a GitHub repository that can be added as a dependency
type RepoCandidate struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	Privacy string `json:"privacy"` // "public" or "private"
	Desc    string `json:"description,omitempty"`
}

// PkgSpec represents a package specification for adding as a dependency
type PkgSpec struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Ref    string `json:"ref"`
	Subdir string `json:"subdir,omitempty"`
}

// ActionResult represents the result of an operation
type ActionResult struct {
	OK      bool                   `json:"ok"`
	Message string                 `json:"message"`
	Err     string                 `json:"error,omitempty"`
	Logs    []string               `json:"logs,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// Reco represents a recommendation
type Reco struct {
	Message   string `json:"message"`
	Severity  string `json:"severity"` // "info", "warn", "error"
	Rationale string `json:"rationale"`
}

// Step represents the current step in the TUI workflow
type Step int

const (
	StepMainMenu Step = iota
	StepChooseSource
	StepSelectGitHubProject // Single-select GitHub repo to clone as project
	StepListRepos           // Multi-select GitHub repos as dependencies
	StepEditSpecs
	StepConfirm
	StepExecute
	StepSummary
)

// SourceMode represents how repositories are sourced
type SourceMode int

const (
	SourceLocalScan SourceMode = iota
	SourceGitHub
	SourceManualURL
)

// StaleInfo represents information about stale dependencies
type StaleInfo struct {
	PackageName string    `json:"package_name"`
	CurrentRef  string    `json:"current_ref"`
	CurrentSHA  string    `json:"current_sha"`
	UpstreamSHA string    `json:"upstream_sha"`
	IsStale     bool      `json:"is_stale"`
	LastChecked time.Time `json:"last_checked"`
}

// PubspecLock represents parsed pubspec.lock information
type PubspecLock struct {
	Dependencies map[string]PubspecLockDep `json:"dependencies"`
}

// PubspecLockDep represents a dependency in pubspec.lock
type PubspecLockDep struct {
	Source      string `json:"source"`
	ResolvedRef string `json:"resolved_ref,omitempty"`
	URL         string `json:"url,omitempty"`
}

// BackupInfo represents information about a backup
type BackupInfo struct {
	OriginalPath string    `json:"original_path"`
	BackupPath   string    `json:"backup_path"`
	Timestamp    time.Time `json:"timestamp"`
	Size         int64     `json:"size"`
}

// CommandExecution represents a command that was or will be executed
type CommandExecution struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	DryRun      bool              `json:"dry_run"`
	StartTime   time.Time         `json:"start_time,omitempty"`
	EndTime     time.Time         `json:"end_time,omitempty"`
	ExitCode    int               `json:"exit_code,omitempty"`
	Output      string            `json:"output,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// ErrViewNotFound represents an error when a view component is not found
type ErrViewNotFound struct {
	Name string
}

func (e ErrViewNotFound) Error() string {
	return fmt.Sprintf("view component '%s' not found", e.Name)
}
