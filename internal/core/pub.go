// Package core/pub.go - Dart/Flutter Pub Command Integration and pubspec.yaml Management
//
// This file provides integration with Dart and Flutter pub commands, offering the same
// functionality as the shell script but with improved error handling and cross-platform
// support. It manages pubspec.yaml files safely and executes pub operations.
//
// Key features:
// - FindPubTool: Auto-detect available dart/flutter commands (shell script parity)
// - AddGitDependency: Add git dependencies using pub commands (not direct YAML editing)
// - Sync: Execute pub get/flutter packages get operations
// - CreateBackup: Safe backup creation before modifying pubspec.yaml
// - Cross-platform pub command execution with proper error handling
// - Concurrent operation support with timeout management
// - Shell script compatible dependency addition workflow
//
// The pub integration maintains the exact same behavior as the shell script while
// providing better error messages and safer operation handling.

package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// FindPubTool finds the first available pub tool (dart or flutter)
// This mirrors the shell script's tool detection logic
func FindPubTool() (string, error) {
	tools := []string{"dart", "flutter"}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			return tool, nil
		}
	}

	return "", fmt.Errorf("neither 'dart' nor 'flutter' found in PATH")
}

// AddGitDependency adds a git dependency using pub add
// This follows Junie's plan to use dart/flutter pub add instead of YAML surgery
func AddGitDependency(logger *Logger, cfg *Config, projectPath string, spec PkgSpec) ActionResult {
	tool, err := FindPubTool()
	if err != nil {
		return ActionResult{
			OK:  false,
			Err: err.Error(),
		}
	}

	// CRITICAL FIX: Fetch the actual package name from the git repository's pubspec.yaml
	// This prevents "name field doesn't match" errors (exit code 65)
	// dart pub add requires the package name to match the name in the repo's pubspec.yaml
	actualName, err := FetchPackageNameFromGit(logger, spec.URL, spec.Ref, spec.Subdir)
	if err != nil {
		logger.Debug("pub", fmt.Sprintf("Failed to fetch package name from git: %s", err.Error()))
		logger.Debug("pub", fmt.Sprintf("Falling back to user-provided name: %s", spec.Name))
		actualName = spec.Name // Fallback to user-provided name if fetch fails
	} else {
		if actualName != spec.Name {
			// Show this at Info level so users can see the automatic correction working
			logger.Info("pub", fmt.Sprintf("📝 Auto-corrected package name: '%s' → '%s'", spec.Name, actualName))
		}
	}

	// Build command arguments using the actual package name from git
	args := []string{"pub", "add", actualName, "--git-url", spec.URL}

	if spec.Ref != "" && spec.Ref != "main" {
		args = append(args, "--git-ref", spec.Ref)
	}

	if spec.Subdir != "" {
		args = append(args, "--git-path", spec.Subdir)
	}

	logger.LogCommand("pub", tool, args)

	// Log the working directory for debugging
	absPath, _ := filepath.Abs(projectPath)
	logger.Debug("pub", fmt.Sprintf("Working directory: %s", absPath))
	logger.Debug("pub", fmt.Sprintf("Full command: %s %s", tool, strings.Join(args, " ")))

	if cfg.DryRun {
		return ActionResult{
			OK:      true,
			Message: fmt.Sprintf("Would execute: %s %s", tool, strings.Join(args, " ")),
			Logs:    []string{fmt.Sprintf("DRY RUN: %s %s", tool, strings.Join(args, " "))},
		}
	}

	// Verify pubspec.yaml exists before attempting to add
	pubspecPath := filepath.Join(projectPath, "pubspec.yaml")
	if _, err := os.Stat(pubspecPath); err != nil {
		errMsg := fmt.Sprintf("pubspec.yaml not found at %s before running pub add", pubspecPath)
		logger.Debug("pub", errMsg)
		return ActionResult{
			OK:   false,
			Err:  errMsg,
			Logs: []string{errMsg},
		}
	}
	logger.Debug("pub", fmt.Sprintf("Verified pubspec.yaml exists at: %s", pubspecPath))

	// INSTRUMENTATION: Capture pubspec.yaml state BEFORE command
	beforeContent, beforeErr := os.ReadFile(pubspecPath)
	if beforeErr == nil {
		logger.Debug("pub", "=== PUBSPEC.YAML STATE BEFORE COMMAND ===")
		logger.Debug("pub", fmt.Sprintf("Size: %d bytes", len(beforeContent)))
		logger.Debug("pub", fmt.Sprintf("First 500 chars:\n%s", string(beforeContent[:min(500, len(beforeContent))])))
	} else {
		logger.Debug("pub", fmt.Sprintf("WARNING: Could not read pubspec.yaml before command: %s", beforeErr))
	}

	// INSTRUMENTATION: Check for lock files
	lockPath := filepath.Join(projectPath, "pubspec.lock")
	dartToolPath := filepath.Join(projectPath, ".dart_tool")
	if lockInfo, err := os.Stat(lockPath); err == nil {
		logger.Debug("pub", fmt.Sprintf("pubspec.lock exists: size=%d, modified=%s", lockInfo.Size(), lockInfo.ModTime()))
	} else {
		logger.Debug("pub", fmt.Sprintf("pubspec.lock does not exist: %s", err))
	}
	if dartToolInfo, err := os.Stat(dartToolPath); err == nil {
		logger.Debug("pub", fmt.Sprintf(".dart_tool exists: modified=%s", dartToolInfo.ModTime()))
	} else {
		logger.Debug("pub", fmt.Sprintf(".dart_tool does not exist: %s", err))
	}

	// INSTRUMENTATION: Record start time
	startTime := time.Now()
	logger.Debug("pub", fmt.Sprintf("=== EXECUTING COMMAND at %s ===", startTime.Format("15:04:05.000")))

	// Execute the command
	cmd := exec.Command(tool, args...)
	cmd.Dir = projectPath

	// Ensure stdin is closed so the command doesn't wait for input
	cmd.Stdin = nil

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	logs := []string{outputStr}

	// INSTRUMENTATION: Record end time
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	logger.Debug("pub", fmt.Sprintf("=== COMMAND COMPLETED at %s (duration: %s) ===", endTime.Format("15:04:05.000"), duration))

	// INSTRUMENTATION: Capture pubspec.yaml state AFTER command
	afterContent, afterErr := os.ReadFile(pubspecPath)
	if afterErr == nil {
		logger.Debug("pub", "=== PUBSPEC.YAML STATE AFTER COMMAND ===")
		logger.Debug("pub", fmt.Sprintf("Size: %d bytes (before: %d)", len(afterContent), len(beforeContent)))
		logger.Debug("pub", fmt.Sprintf("Changed: %t", !bytesEqual(beforeContent, afterContent)))
		logger.Debug("pub", fmt.Sprintf("First 500 chars:\n%s", string(afterContent[:min(500, len(afterContent))])))

		// Check if the file is valid YAML by looking for basic structure
		hasName := strings.Contains(string(afterContent), "name:")
		hasDependencies := strings.Contains(string(afterContent), "dependencies:")
		logger.Debug("pub", fmt.Sprintf("Validation: has 'name:' = %t, has 'dependencies:' = %t", hasName, hasDependencies))
	} else {
		logger.Debug("pub", fmt.Sprintf("ERROR: Could not read pubspec.yaml after command: %s", afterErr))
	}

	// INSTRUMENTATION: Check lock files after command
	if lockInfo, err := os.Stat(lockPath); err == nil {
		logger.Debug("pub", fmt.Sprintf("pubspec.lock after: size=%d, modified=%s", lockInfo.Size(), lockInfo.ModTime()))
	} else {
		logger.Debug("pub", fmt.Sprintf("pubspec.lock after: does not exist: %s", err))
	}
	if dartToolInfo, err := os.Stat(dartToolPath); err == nil {
		logger.Debug("pub", fmt.Sprintf(".dart_tool after: modified=%s", dartToolInfo.ModTime()))
	} else {
		logger.Debug("pub", fmt.Sprintf(".dart_tool after: does not exist: %s", err))
	}

	if err != nil {
		logger.Debug("pub", fmt.Sprintf("Command failed: %s", err.Error()))
		logger.Debug("pub", fmt.Sprintf("Command output: %s", outputStr))

		// Analyze the error and attempt intelligent recovery
		conflictAnalysis := analyzeDependencyConflict(outputStr, err)

		// If this is a recoverable conflict, try resolution strategies
		if conflictAnalysis.IsRecoverable {
			// Notify user about the conflict and that we're working on it
			logger.Info("pub", fmt.Sprintf("⚠️  Dependency conflict detected while adding %s", actualName))
			logger.Info("pub", fmt.Sprintf("🔧 Issue: %s", conflictAnalysis.UserMessage))
			logger.Info("pub", "🔄 Working on automatic resolution... please wait")

			// Show what we're trying to do
			if conflictAnalysis.ConflictingPkg != "" {
				logger.Info("pub", fmt.Sprintf("📋 Resolving conflict with package: %s", conflictAnalysis.ConflictingPkg))
			}

			// Attempt resolution
			if resolvedResult := attemptConflictResolution(logger, cfg, projectPath, spec, conflictAnalysis); resolvedResult.OK {
				// Success - add detailed resolution info to result
				resolvedResult.Data = map[string]interface{}{
					"conflict_resolved": true,
					"conflict_type":     conflictAnalysis.ConflictType,
					"conflicting_pkg":   conflictAnalysis.ConflictingPkg,
					"resolution_method": "inline_dependency_override",
					"user_message":      fmt.Sprintf("Successfully resolved %s conflict with %s", conflictAnalysis.ConflictType, conflictAnalysis.ConflictingPkg),
				}
				logger.Info("pub", fmt.Sprintf("✅ Conflict resolved! %s has been successfully added", actualName))
				logger.Info("pub", fmt.Sprintf("🛠️  Resolution: Used dependency override for %s", conflictAnalysis.ConflictingPkg))
				return resolvedResult
			}

			logger.Info("pub", "❌ Automatic conflict resolution failed - manual intervention may be required")
		}

		// Enhanced error reporting with conflict details
		errDetail := fmt.Sprintf("%s (working dir: %s)", err.Error(), absPath)
		if conflictAnalysis.ConflictType != "unknown" {
			errDetail = fmt.Sprintf("Dependency conflict (%s): %s", conflictAnalysis.ConflictType, conflictAnalysis.UserMessage)
		}

		return ActionResult{
			OK:   false,
			Err:  errDetail,
			Logs: logs,
			Data: map[string]interface{}{
				"conflict_type":  conflictAnalysis.ConflictType,
				"is_recoverable": conflictAnalysis.IsRecoverable,
				"suggested_fix":  conflictAnalysis.SuggestedFix,
				"user_message":   conflictAnalysis.UserMessage,
			},
		}
	}

	logger.Debug("pub", fmt.Sprintf("Command succeeded: %s", outputStr))

	// CRITICAL FIX: Wait for dart pub to fully release file locks
	// dart pub add creates/updates pubspec.lock and .dart_tool/package_config.json
	// On Windows, these files may remain locked briefly after the process exits
	// A small delay ensures subsequent dart pub add commands don't fail with exit 65
	time.Sleep(500 * time.Millisecond)
	logger.Debug("pub", "Waited 500ms for file locks to release")

	return ActionResult{
		OK:      true,
		Message: fmt.Sprintf("Successfully added %s", spec.Name),
		Logs:    logs,
	}
}

// Helper function to compare byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ConflictAnalysis holds information about a dependency conflict
type ConflictAnalysis struct {
	ConflictType    string   // Type of conflict: "version", "sdk", "circular", "platform", "git_vs_hosted", "transitive", "unknown"
	SubType         string   // More specific conflict classification
	IsRecoverable   bool     // Whether we can attempt automatic resolution
	SuggestedFix    string   // Human-readable description of fix strategy
	UserMessage     string   // Clear message for the user
	ConflictingPkg  string   // Name of conflicting package
	SourceConflict  string   // Details about source conflict (git vs hosted)
	ResolutionSteps []string // Step-by-step resolution approach
}

// analyzeDependencyConflict analyzes pub add output to identify conflict types with enhanced classification
func analyzeDependencyConflict(output string, err error) ConflictAnalysis {
	lowerOutput := strings.ToLower(output)
	originalOutput := output

	// Extract conflicting package name if possible
	conflictingPkg := extractConflictingPackageName(originalOutput)

	// SDK constraint violations (check first, as they often include "version solving failed")
	if (strings.Contains(lowerOutput, "sdk constraint") ||
		strings.Contains(lowerOutput, "requires sdk") ||
		strings.Contains(lowerOutput, "dart sdk")) ||
		(strings.Contains(lowerOutput, "current dart sdk version") && strings.Contains(lowerOutput, "requires sdk")) {
		return ConflictAnalysis{
			ConflictType:    "sdk",
			SubType:         "constraint_violation",
			IsRecoverable:   false,
			SuggestedFix:    "Update Dart/Flutter SDK or choose compatible package version",
			UserMessage:     "Package requires newer SDK version than project supports",
			ConflictingPkg:  conflictingPkg,
			ResolutionSteps: []string{"Check package SDK requirements", "Update Flutter/Dart SDK", "Or use older package version"},
		}
	}

	// Platform constraints (check before general version conflicts)
	if (strings.Contains(lowerOutput, "platform") &&
		(strings.Contains(lowerOutput, "not supported") || strings.Contains(lowerOutput, "doesn't support"))) ||
		(strings.Contains(lowerOutput, "doesn't support platform") || strings.Contains(lowerOutput, "unsupported platform")) {
		return ConflictAnalysis{
			ConflictType:    "platform",
			SubType:         "incompatible_platform",
			IsRecoverable:   false,
			SuggestedFix:    "Choose package compatible with target platforms",
			UserMessage:     "Package not compatible with project's target platforms",
			ConflictingPkg:  conflictingPkg,
			ResolutionSteps: []string{"Check package platform support", "Update pubspec.yaml platforms", "Or choose alternative package"},
		}
	}

	// Circular dependency detection
	if strings.Contains(lowerOutput, "circular") ||
		strings.Contains(lowerOutput, "cycle") ||
		strings.Contains(lowerOutput, "circular dependency") {
		return ConflictAnalysis{
			ConflictType:    "circular",
			SubType:         "dependency_cycle",
			IsRecoverable:   false,
			SuggestedFix:    "Remove circular dependency or choose different packages",
			UserMessage:     "Circular dependency detected between packages",
			ConflictingPkg:  conflictingPkg,
			ResolutionSteps: []string{"Identify circular dependency chain", "Remove one dependency", "Or restructure dependencies"},
		}
	}

	// Git vs Hosted conflict detection (CRITICAL for the reported issue)
	if detectGitVsHostedConflict(originalOutput) {
		conflictDetails := analyzeGitVsHostedConflict(originalOutput)
		return ConflictAnalysis{
			ConflictType:    "git_vs_hosted",
			SubType:         "source_conflict",
			IsRecoverable:   true,
			SuggestedFix:    "Use dependency_overrides to force git source",
			UserMessage:     fmt.Sprintf("Package source conflict: %s", conflictDetails),
			ConflictingPkg:  conflictingPkg,
			SourceConflict:  conflictDetails,
			ResolutionSteps: []string{"Add dependency_overrides section", "Force git source for conflicting package", "Retry package addition"},
		}
	}

	// Transitive dependency conflicts (check before general version conflicts)
	if strings.Contains(lowerOutput, "transitive") ||
		strings.Contains(lowerOutput, "indirect dependency") ||
		strings.Contains(lowerOutput, "transitive dependency") {
		return ConflictAnalysis{
			ConflictType:    "transitive",
			SubType:         "indirect_conflict",
			IsRecoverable:   true,
			SuggestedFix:    "Resolve transitive dependency with dependency overrides",
			UserMessage:     "Conflict in transitive dependencies - attempting resolution",
			ConflictingPkg:  conflictingPkg,
			ResolutionSteps: []string{"Run pub deps to analyze", "Add dependency_overrides if needed", "Retry package addition"},
		}
	}

	// Version conflict patterns (check last, as this is most general)
	if strings.Contains(lowerOutput, "version solving failed") ||
		strings.Contains(lowerOutput, "version conflict") ||
		strings.Contains(lowerOutput, "incompatible version") ||
		(strings.Contains(lowerOutput, "depends on") && strings.Contains(lowerOutput, "version solving failed")) {
		return ConflictAnalysis{
			ConflictType:    "version",
			SubType:         "version_mismatch",
			IsRecoverable:   true,
			SuggestedFix:    "Run pub get to resolve version constraints",
			UserMessage:     "Version conflict detected - attempting automatic resolution",
			ConflictingPkg:  conflictingPkg,
			ResolutionSteps: []string{"Run pub get to resolve versions", "Check for constraint conflicts", "Retry package addition"},
		}
	}

	// Default case
	return ConflictAnalysis{
		ConflictType:    "unknown",
		SubType:         "unclassified",
		IsRecoverable:   false,
		SuggestedFix:    "Manual intervention required",
		UserMessage:     "Unknown dependency conflict - check pub output for details",
		ConflictingPkg:  conflictingPkg,
		ResolutionSteps: []string{"Review full error output", "Check package compatibility", "Manual resolution required"},
	}
}

// attemptConflictResolution tries to resolve dependency conflicts automatically with enhanced strategies
func attemptConflictResolution(logger *Logger, cfg *Config, projectPath string, spec PkgSpec, analysis ConflictAnalysis) ActionResult {
	logger.Info("pub", fmt.Sprintf("🔧 Starting resolution for %s conflict (subtype: %s)", analysis.ConflictType, analysis.SubType))

	// For all recoverable conflicts, try using inline dependency overrides in the dart pub add command
	if analysis.IsRecoverable && analysis.ConflictingPkg != "" {
		logger.Info("pub", fmt.Sprintf("🔄 Attempting conflict resolution using inline dependency override for %s", analysis.ConflictingPkg))
		return resolveWithInlineOverride(logger, cfg, projectPath, spec, analysis)
	}

	// Log resolution steps for debugging
	for i, step := range analysis.ResolutionSteps {
		logger.Debug("pub", fmt.Sprintf("Resolution step %d: %s", i+1, step))
	}

	// If we get here, fallback to the general error message
	logger.Debug("pub", fmt.Sprintf("No automatic resolution available for conflict type: %s", analysis.ConflictType))
	return ActionResult{
		OK:  false,
		Err: fmt.Sprintf("No resolution strategy available for %s conflict", analysis.ConflictType),
		Data: map[string]interface{}{
			"conflict_analysis": analysis,
			"manual_steps":      analysis.ResolutionSteps,
		},
	}
}

// resolveWithInlineOverride uses the dart pub add inline override syntax to resolve conflicts
// Based on user discovery: dart pub add package:"{git: url}" override:conflicting_package:any
func resolveWithInlineOverride(logger *Logger, cfg *Config, projectPath string, spec PkgSpec, analysis ConflictAnalysis) ActionResult {
	tool, err := FindPubTool()
	if err != nil {
		return ActionResult{
			OK:  false,
			Err: err.Error(),
		}
	}

	// Fetch the actual package name
	actualName, err := FetchPackageNameFromGit(logger, spec.URL, spec.Ref, spec.Subdir)
	if err != nil {
		logger.Debug("pub", fmt.Sprintf("Failed to fetch package name from git: %s", err.Error()))
		actualName = spec.Name
	} else {
		if actualName != spec.Name {
			logger.Info("pub", fmt.Sprintf("📝 Auto-corrected package name: '%s' → '%s'", spec.Name, actualName))
		}
	}

	// Build git URL specification for inline syntax
	gitSpec := fmt.Sprintf("{git: %s", spec.URL)
	if spec.Ref != "" {
		gitSpec += fmt.Sprintf(", ref: %s", spec.Ref)
	}
	if spec.Subdir != "" {
		gitSpec += fmt.Sprintf(", path: %s", spec.Subdir)
	}
	gitSpec += "}"

	// Build command with inline dependency override
	// Format: dart pub add package_name:"git_spec" override:conflicting_package:any
	args := []string{"pub", "add", fmt.Sprintf("%s:\"%s\"", actualName, gitSpec)}

	// Add dependency override for the conflicting package
	if analysis.ConflictingPkg != "" {
		overrideArg := fmt.Sprintf("override:%s:any", analysis.ConflictingPkg)
		args = append(args, overrideArg)
		logger.Info("pub", fmt.Sprintf("📋 Adding dependency override: %s", overrideArg))
	}

	logger.LogCommand("pub", tool, args)
	if analysis.ConflictingPkg != "" {
		logger.Info("pub", fmt.Sprintf("🔧 Applying conflict resolution for %s (conflicting with %s)", actualName, analysis.ConflictingPkg))
	} else {
		logger.Info("pub", fmt.Sprintf("🔧 Applying enhanced installation method for %s", actualName))
	}

	if cfg.DryRun {
		return ActionResult{
			OK:      true,
			Message: fmt.Sprintf("Would execute with inline override: %s %s", tool, strings.Join(args, " ")),
			Logs:    []string{fmt.Sprintf("DRY RUN: %s %s", tool, strings.Join(args, " "))},
		}
	}

	// Execute the command with inline override
	cmd := exec.Command(tool, args...)
	cmd.Dir = projectPath
	cmd.Stdin = nil

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	logs := []string{outputStr}

	if err != nil {
		logger.Debug("pub", fmt.Sprintf("Inline override resolution failed: %s", err.Error()))
		logger.Debug("pub", fmt.Sprintf("Command output: %s", outputStr))
		return ActionResult{
			OK:   false,
			Err:  fmt.Sprintf("Inline override resolution failed: %s", err.Error()),
			Logs: logs,
		}
	}

	logger.Info("pub", fmt.Sprintf("✅ Package %s successfully installed with conflict resolution", actualName))

	// Wait for file locks and return success with detailed resolution info
	time.Sleep(500 * time.Millisecond)
	return ActionResult{
		OK:      true,
		Message: fmt.Sprintf("Successfully added %s with dependency override", actualName),
		Logs:    logs,
		Data: map[string]interface{}{
			"conflict_resolved": true,
			"conflicting_pkg":   analysis.ConflictingPkg,
			"resolution_method": "inline_dependency_override",
			"package_name":      actualName,
		},
	}
}

// addGitDependencyWithoutConflictResolution adds a git dependency without conflict resolution (to avoid recursion)
func addGitDependencyWithoutConflictResolution(logger *Logger, cfg *Config, projectPath string, spec PkgSpec) ActionResult {
	tool, err := FindPubTool()
	if err != nil {
		return ActionResult{
			OK:  false,
			Err: err.Error(),
		}
	}

	// Fetch the actual package name (same as main function)
	actualName, err := FetchPackageNameFromGit(logger, spec.URL, spec.Ref, spec.Subdir)
	if err != nil {
		logger.Debug("pub", fmt.Sprintf("Failed to fetch package name from git: %s", err.Error()))
		actualName = spec.Name
	} else {
		if actualName != spec.Name {
			logger.Info("pub", fmt.Sprintf("📝 Auto-corrected package name: '%s' → '%s'", spec.Name, actualName))
		}
	}

	// Build command arguments
	args := []string{"pub", "add", actualName, "--git-url", spec.URL}
	if spec.Ref != "" && spec.Ref != "main" {
		args = append(args, "--git-ref", spec.Ref)
	}
	if spec.Subdir != "" {
		args = append(args, "--git-path", spec.Subdir)
	}

	logger.LogCommand("pub", tool, args)

	if cfg.DryRun {
		return ActionResult{
			OK:      true,
			Message: fmt.Sprintf("Would execute: %s %s", tool, strings.Join(args, " ")),
			Logs:    []string{fmt.Sprintf("DRY RUN: %s %s", tool, strings.Join(args, " "))},
		}
	}

	// Execute the command (no conflict resolution on retry)
	cmd := exec.Command(tool, args...)
	cmd.Dir = projectPath
	cmd.Stdin = nil

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	logs := []string{outputStr}

	if err != nil {
		// No conflict resolution on retry - just return the error
		return ActionResult{
			OK:   false,
			Err:  fmt.Sprintf("Retry failed: %s", err.Error()),
			Logs: logs,
		}
	}

	// Wait for file locks and return success
	time.Sleep(500 * time.Millisecond)
	return ActionResult{
		OK:      true,
		Message: fmt.Sprintf("Successfully added %s", actualName),
		Logs:    logs,
	}
}

// AddGitDependenciesBatch adds multiple git dependencies efficiently in a single operation

// AddGitDependenciesIndividual adds dependency one by one (fallback method)

// Sync runs pub get to synchronize dependencies
func Sync(logger *Logger, cfg *Config, projectPath string) ActionResult {
	tool, err := FindPubTool()
	if err != nil {
		return ActionResult{
			OK:  false,
			Err: err.Error(),
		}
	}

	args := []string{"pub", "get"}
	logger.LogCommand("sync", tool, args)

	if cfg.DryRun {
		return ActionResult{
			OK:      true,
			Message: "Would run pub get",
			Logs:    []string{fmt.Sprintf("DRY RUN: %s %s", tool, strings.Join(args, " "))},
		}
	}

	cmd := exec.Command(tool, args...)
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	logs := []string{strings.TrimSpace(string(output))}

	if err != nil {
		return ActionResult{
			OK:   false,
			Err:  err.Error(),
			Logs: logs,
		}
	}

	return ActionResult{
		OK:      true,
		Message: "Dependencies synchronized",
		Logs:    logs,
	}
}

// CreateBackup creates a timestamped backup of pubspec.yaml
// This mirrors the shell script's backup strategy
func CreateBackup(projectPath string) (BackupInfo, error) {
	pubspecPath := filepath.Join(projectPath, "pubspec.yaml")

	// Check if pubspec.yaml exists
	info, err := os.Stat(pubspecPath)
	if err != nil {
		return BackupInfo{}, fmt.Errorf("pubspec.yaml not found: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now()
	backupName := fmt.Sprintf("pubspec.yaml.backup.%s", timestamp.Format("20060102_150405"))
	backupPath := filepath.Join(projectPath, backupName)

	// Copy the file
	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		return BackupInfo{}, fmt.Errorf("failed to read pubspec.yaml: %w", err)
	}

	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return BackupInfo{}, fmt.Errorf("failed to create backup: %w", err)
	}

	return BackupInfo{
		OriginalPath: pubspecPath,
		BackupPath:   backupPath,
		Timestamp:    timestamp,
		Size:         info.Size(),
	}, nil
}

// RestoreBackup restores a backup file

// ValidatePubspec performs basic validation on pubspec.yaml

// Compiled regex patterns for efficient parsing
var (
	pubspecParseOnce sync.Once
	// Git dependency pattern: captures package name, URL, ref, and path
	gitDepPattern *regexp.Regexp
	// General YAML value extraction patterns
	_           *regexp.Regexp
	urlPattern  *regexp.Regexp
	refPattern  *regexp.Regexp
	pathPattern *regexp.Regexp
)

// initPubspecRegex initializes regex patterns for pubspec parsing
func initPubspecRegex() {
	pubspecParseOnce.Do(func() {
		// Pattern to match git dependencies in pubspec.yaml
		// This matches the entire git dependency block
		gitDepPattern = regexp.MustCompile(`(?s)(\s+\w+):\s*\n?\s*git:\s*\n?(?:\s*url:\s*['"]?([^'"\n]+)['"]?\s*\n?)?(?:\s*ref:\s*['"]?([^'"\n]+)['"]?\s*\n?)?(?:\s*path:\s*['"]?([^'"\n]+)['"]?\s*\n?)?`)

		// Individual value patterns for fallback parsing
		_ = regexp.MustCompile(`^\s*name:\s*['"]?([^'"\n]+)['"]?\s*$`)
		urlPattern = regexp.MustCompile(`^\s*url:\s*['"]?([^'"\n]+)['"]?\s*$`)
		refPattern = regexp.MustCompile(`^\s*ref:\s*['"]?([^'"\n]+)['"]?\s*$`)
		pathPattern = regexp.MustCompile(`^\s*path:\s*['"]?([^'"\n]+)['"]?\s*$`)
	})
}

// ListGitDependencies extracts git dependencies from pubspec.yaml with optimized parsing
func ListGitDependencies(projectPath string) ([]PkgSpec, error) {
	initPubspecRegex() // Initialize regex patterns

	pubspecPath := filepath.Join(projectPath, "pubspec.yaml")
	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pubspec.yaml: %w", err)
	}

	// Try fast regex-based parsing first
	if deps := parseGitDepsWithRegex(string(content)); len(deps) > 0 {
		return deps, nil
	}

	// Fallback to line-by-line parsing for complex cases
	return parseGitDepsLineByLine(string(content))
}

// parseGitDepsWithRegex uses compiled regex for fast parsing
func parseGitDepsWithRegex(content string) []PkgSpec {
	var deps []PkgSpec

	// Find dependencies section first
	depsStart := strings.Index(content, "dependencies:")
	if depsStart == -1 {
		return deps
	}

	// Find next top-level section to limit scope
	depsContent := content[depsStart:]
	nextSection := regexp.MustCompile(`\n\w+:`).FindStringIndex(depsContent[12:]) // Skip "dependencies:"
	if nextSection != nil {
		depsContent = depsContent[:12+nextSection[0]]
	}

	// Find all git dependencies in the dependencies section
	matches := gitDepPattern.FindAllStringSubmatch(depsContent, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			pkg := PkgSpec{
				Name: strings.TrimSpace(match[1]),
			}

			// Extract URL (match[2])
			if len(match) > 2 && match[2] != "" {
				pkg.URL = strings.TrimSpace(match[2])
			}

			// Extract ref (match[3])
			if len(match) > 3 && match[3] != "" {
				pkg.Ref = strings.TrimSpace(match[3])
			}

			// Extract path (match[4])
			if len(match) > 4 && match[4] != "" {
				pkg.Subdir = strings.TrimSpace(match[4])
			}

			// Only add if we have a name and URL
			if pkg.Name != "" && pkg.URL != "" {
				deps = append(deps, pkg)
			}
		}
	}

	return deps
}

// parseGitDepsLineByLine fallback parser for complex YAML cases
func parseGitDepsLineByLine(content string) ([]PkgSpec, error) {
	var deps []PkgSpec
	lines := strings.Split(content, "\n")
	inDependencies := false
	inGitDep := false
	currentPkg := PkgSpec{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for dependencies section
		if trimmed == "dependencies:" {
			inDependencies = true
			continue
		}

		// Exit dependencies if we hit another top-level section
		if inDependencies && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			inDependencies = false
		}

		if !inDependencies {
			continue
		}

		// Check for package name (indented but not double-indented)
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.Contains(line, ":") && !strings.HasPrefix(trimmed, "#") {
			// Save previous git dependency if we had one
			if inGitDep && currentPkg.Name != "" && currentPkg.URL != "" {
				deps = append(deps, currentPkg)
			}

			// Start new package
			parts := strings.SplitN(trimmed, ":", 2)
			currentPkg = PkgSpec{Name: strings.TrimSpace(parts[0])}
			inGitDep = false
		}

		// Check for git dependency
		if trimmed == "git:" {
			inGitDep = true
		}

		// Extract git details using regex for consistent parsing
		if inGitDep {
			if match := urlPattern.FindStringSubmatch(line); len(match) > 1 {
				currentPkg.URL = match[1]
			} else if match := refPattern.FindStringSubmatch(line); len(match) > 1 {
				currentPkg.Ref = match[1]
			} else if match := pathPattern.FindStringSubmatch(line); len(match) > 1 {
				currentPkg.Subdir = match[1]
			}
		}
	}

	// Don't forget the last dependency
	if inGitDep && currentPkg.Name != "" && currentPkg.URL != "" {
		deps = append(deps, currentPkg)
	}

	return deps, nil
}

// extractConflictingPackageName attempts to extract the conflicting package name from error output
func extractConflictingPackageName(output string) string {
	// Look for patterns like "because project_name depends on package_name"
	patterns := []string{
		// Git vs hosted specific patterns - prioritize these
		`(\w+) from hosted is required`,
		`depends on (\w+) from git`,
		`depends on (\w+) from hosted`,
		`So, because \w+ depends on (\w+) from`,
		// General patterns
		`depends on (\w+) [\^\~]?[\d\.]+`,
		`package (\w+) from`,
		`(\w+) from git`,
		`(\w+) from hosted`,
		// Fallback patterns
		`depends on (\w+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(output); len(matches) > 1 {
			pkg := matches[1]
			// Skip common project-type words
			if pkg != "flutter" && pkg != "sdk" && pkg != "dart" {
				return pkg
			}
		}
	}

	return ""
}

// detectGitVsHostedConflict detects conflicts between git and hosted package sources
func detectGitVsHostedConflict(output string) bool {
	lowerOutput := strings.ToLower(output)

	// Look for the classic git vs hosted conflict pattern
	hasGitSource := strings.Contains(lowerOutput, "from git")
	hasHostedSource := strings.Contains(lowerOutput, "from hosted")
	hasVersionSolvingFailed := strings.Contains(lowerOutput, "version solving failed")

	return hasGitSource && hasHostedSource && hasVersionSolvingFailed
}

// analyzeGitVsHostedConflict provides detailed analysis of git vs hosted conflicts
func analyzeGitVsHostedConflict(output string) string {
	lines := strings.Split(output, "\n")
	var conflictLines []string

	for _, line := range lines {
		lowerLine := strings.ToLower(line)
		if (strings.Contains(lowerLine, "from git") || strings.Contains(lowerLine, "from hosted")) &&
			(strings.Contains(lowerLine, "depends on") || strings.Contains(lowerLine, "requires")) {
			conflictLines = append(conflictLines, strings.TrimSpace(line))
		}
	}

	if len(conflictLines) > 0 {
		return strings.Join(conflictLines, " | ")
	}

	return "Git vs hosted source conflict detected"
}
