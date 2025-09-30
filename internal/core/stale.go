package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CheckStaleHeuristic performs a quick heuristic check for stale dependencies
// If pubspec.lock is older than 24 hours, mark git deps as potentially stale
func CheckStaleHeuristic(projectPath string) (bool, string, error) {
	lockPath := filepath.Join(projectPath, "pubspec.lock")

	info, err := os.Stat(lockPath)
	if os.IsNotExist(err) {
		return false, "", nil // No lock file, nothing to check
	}
	if err != nil {
		return false, "", fmt.Errorf("failed to stat pubspec.lock: %w", err)
	}

	age := time.Since(info.ModTime())
	staleThreshold := 24 * time.Hour

	return age > staleThreshold, lockPath, nil
}

// CheckStalePrecise performs precise stale checking by comparing lock file SHAs with upstream
func CheckStalePrecise(logger *Logger, projectPath string) ([]StaleInfo, error) {
	lockPath := filepath.Join(projectPath, "pubspec.lock")

	// Parse pubspec.lock
	lockDeps, err := parsePubspecLock(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pubspec.lock: %w", err)
	}

	// Get git dependencies from pubspec.yaml for URL mapping
	gitDeps, err := ListGitDependencies(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list git dependencies: %w", err)
	}

	// Create URL mapping
	urlMap := make(map[string]string)
	refMap := make(map[string]string)
	for _, dep := range gitDeps {
		urlMap[dep.Name] = dep.URL
		refMap[dep.Name] = dep.Ref
		if refMap[dep.Name] == "" {
			refMap[dep.Name] = "main"
		}
	}

	var staleInfo []StaleInfo

	// Check each git dependency in the lock file
	for name, lockDep := range lockDeps.Dependencies {
		if lockDep.Source != "git" {
			continue
		}

		url := urlMap[name]
		if url == "" {
			url = lockDep.URL
		}

		ref := refMap[name]
		if ref == "" {
			ref = "main"
		}

		logger.Debug("stale", fmt.Sprintf("Checking %s at %s#%s", name, url, ref))

		// Get upstream SHA
		upstreamSHA, err := GitLsRemote(url, ref)
		if err != nil {
			logger.Debug("stale", fmt.Sprintf("Failed to get upstream SHA for %s: %v", name, err))
			continue
		}

		// Truncate to 7 characters for comparison (like git does)
		if len(upstreamSHA) > 7 {
			upstreamSHA = upstreamSHA[:7]
		}

		currentSHA := lockDep.ResolvedRef
		if len(currentSHA) > 7 {
			currentSHA = currentSHA[:7]
		}

		isStale := currentSHA != upstreamSHA

		staleInfo = append(staleInfo, StaleInfo{
			PackageName: name,
			CurrentRef:  ref,
			CurrentSHA:  currentSHA,
			UpstreamSHA: upstreamSHA,
			IsStale:     isStale,
			LastChecked: time.Now(),
		})

		if isStale {
			logger.Info("stale", fmt.Sprintf("%s is stale: %s -> %s", name, currentSHA, upstreamSHA))
		}
	}

	return staleInfo, nil
}

// parsePubspecLock parses the pubspec.lock file and extracts dependency information
func parsePubspecLock(lockPath string) (*PubspecLock, error) {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}

	// Parse YAML-like structure (simplified parser for our needs)
	lock := &PubspecLock{
		Dependencies: make(map[string]PubspecLockDep),
	}

	lines := strings.Split(string(content), "\\n")
	inPackages := false
	currentPkg := ""
	currentDep := PubspecLockDep{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "packages:" {
			inPackages = true
			continue
		}

		if !inPackages {
			continue
		}

		// Check for package name (two spaces indentation)
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(line, ":") {
			// Save previous package
			if currentPkg != "" {
				lock.Dependencies[currentPkg] = currentDep
			}

			// Start new package
			currentPkg = strings.TrimSpace(strings.TrimSuffix(line, ":"))
			currentDep = PubspecLockDep{}
		}

		// Parse dependency fields (four spaces indentation)
		if strings.HasPrefix(line, "    ") {
			if strings.HasPrefix(trimmed, "source:") {
				currentDep.Source = extractValue(trimmed, "source:")
			} else if strings.HasPrefix(trimmed, "resolved-ref:") {
				currentDep.ResolvedRef = extractValue(trimmed, "resolved-ref:")
			} else if strings.HasPrefix(trimmed, "url:") {
				currentDep.URL = extractValue(trimmed, "url:")
			}
		}
	}

	// Don't forget the last package
	if currentPkg != "" {
		lock.Dependencies[currentPkg] = currentDep
	}

	return lock, nil
}

// extractValue extracts the value from a YAML line like "key: value"
func extractValue(line, key string) string {
	value := strings.TrimSpace(strings.TrimPrefix(line, key))
	value = strings.Trim(value, "\"'")
	return value
}

// UpdateStaleDependencies updates stale git dependencies
func UpdateStaleDependencies(logger *Logger, cfg *Config, projectPath string, stalePackages []string) ActionResult {
	if len(stalePackages) == 0 {
		return ActionResult{
			OK:      true,
			Message: "No stale packages to update",
		}
	}

	tool, err := FindPubTool()
	if err != nil {
		return ActionResult{
			OK:  false,
			Err: err.Error(),
		}
	}

	// Create backup before updating
	backupInfo, err := CreateBackup(projectPath)
	if err != nil {
		logger.Error("backup", err)
	} else {
		logger.Info("backup", fmt.Sprintf("Created backup: %s", backupInfo.BackupPath))
	}

	args := []string{"pub", "upgrade"}
	logger.LogCommand("update", tool, args)

	if cfg.DryRun {
		return ActionResult{
			OK:      true,
			Message: fmt.Sprintf("Would update %d stale packages", len(stalePackages)),
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
		Message: fmt.Sprintf("Updated %d stale packages", len(stalePackages)),
		Logs:    logs,
		Data: map[string]interface{}{
			"updated_packages": stalePackages,
		},
	}
}

// ExpressGitUpdate performs the "express git update" feature from the shell script
func ExpressGitUpdate(logger *Logger, cfg *Config, projectPath string) ActionResult {
	logger.Info("express", "Starting express Git package update")

	// Check if there are any git dependencies
	gitDeps, err := ListGitDependencies(projectPath)
	if err != nil {
		return ActionResult{
			OK:  false,
			Err: err.Error(),
		}
	}

	if len(gitDeps) == 0 {
		return ActionResult{
			OK:      true,
			Message: "No git dependencies found",
		}
	}

	// Perform stale check
	staleInfo, err := CheckStalePrecise(logger, projectPath)
	if err != nil {
		logger.Error("stale", err)
		// Continue with heuristic approach
		isStale, lockPath, _ := CheckStaleHeuristic(projectPath)
		if isStale {
			logger.Info("express", fmt.Sprintf("Lock file %s is older than 24h, updating all git packages", lockPath))
		}
	}

	// Count stale packages
	var stalePackages []string
	for _, info := range staleInfo {
		if info.IsStale {
			stalePackages = append(stalePackages, info.PackageName)
		}
	}

	if len(stalePackages) == 0 {
		return ActionResult{
			OK:      true,
			Message: "All git packages are up to date",
		}
	}

	// Update stale packages
	return UpdateStaleDependencies(logger, cfg, projectPath, stalePackages)
}
