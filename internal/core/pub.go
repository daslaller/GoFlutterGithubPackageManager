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

	// Build command arguments
	args := []string{"pub", "add", spec.Name, "--git-url", spec.URL}

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

	// Execute the command
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
		Message: fmt.Sprintf("Successfully added %s", spec.Name),
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
