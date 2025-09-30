package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
func RestoreBackup(backupInfo BackupInfo) error {
	content, err := os.ReadFile(backupInfo.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	if err := os.WriteFile(backupInfo.OriginalPath, content, 0644); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// ValidatePubspec performs basic validation on pubspec.yaml
func ValidatePubspec(projectPath string) ([]string, error) {
	var issues []string
	pubspecPath := filepath.Join(projectPath, "pubspec.yaml")

	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		return issues, fmt.Errorf("failed to read pubspec.yaml: %w", err)
	}

	lines := strings.Split(string(content), "\\n")
	hasName := false
	hasFlutter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "name:") {
			hasName = true
		}

		if strings.Contains(trimmed, "flutter:") {
			hasFlutter = true
		}
	}

	if !hasName {
		issues = append(issues, "Missing 'name' field in pubspec.yaml")
	}

	if !hasFlutter {
		issues = append(issues, "Missing Flutter dependency - this might not be a Flutter project")
	}

	return issues, nil
}

// ListGitDependencies extracts git dependencies from pubspec.yaml
func ListGitDependencies(projectPath string) ([]PkgSpec, error) {
	pubspecPath := filepath.Join(projectPath, "pubspec.yaml")
	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pubspec.yaml: %w", err)
	}

	var deps []PkgSpec
	lines := strings.Split(string(content), "\\n")
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
		if inDependencies && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\\t") && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
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

		// Extract git details
		if inGitDep {
			if strings.HasPrefix(trimmed, "url:") {
				currentPkg.URL = strings.TrimSpace(strings.TrimPrefix(trimmed, "url:"))
				currentPkg.URL = strings.Trim(currentPkg.URL, "\"'")
			} else if strings.HasPrefix(trimmed, "ref:") {
				currentPkg.Ref = strings.TrimSpace(strings.TrimPrefix(trimmed, "ref:"))
				currentPkg.Ref = strings.Trim(currentPkg.Ref, "\"'")
			} else if strings.HasPrefix(trimmed, "path:") {
				currentPkg.Subdir = strings.TrimSpace(strings.TrimPrefix(trimmed, "path:"))
				currentPkg.Subdir = strings.Trim(currentPkg.Subdir, "\"'")
			}
		}
	}

	// Don't forget the last dependency
	if inGitDep && currentPkg.Name != "" && currentPkg.URL != "" {
		deps = append(deps, currentPkg)
	}

	return deps, nil
}
