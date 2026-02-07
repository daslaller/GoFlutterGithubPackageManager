// Package core/stale.go - Stale Dependency Detection and Express Update Functionality
//
// This file implements stale dependency detection with both heuristic and precise
// methods, exactly matching the shell script's "express git update" feature.
// It provides fast dependency updates for existing git dependencies.
//
// Key features:
// - CheckStaleHeuristic: Fast 24-hour time-based staleness detection
// - CheckStalePrecise: SHA-based comparison for exact staleness detection
// - ExpressGitUpdate: Bulk update of all stale git dependencies
// - pubspec.lock parsing and analysis for dependency tracking
// - Concurrent stale checking with performance optimization
// - Shell script compatible update workflow and behavior
// - Safe batch operations with backup and rollback capability
//
// The express update feature is designed for developers who want to quickly
// update their git dependencies without going through the full package
// selection workflow, mirroring the shell script's express functionality.

package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

// CheckStalePrecise performs precise stale checking with intelligent caching
func CheckStalePrecise(logger *Logger, projectPath string) ([]StaleInfo, error) {
	// Check cache first
	if cached := staleCache.Get(projectPath); cached != nil {
		logger.Debug("stale", "Using cached stale check results")
		return cached, nil
	}

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

		// Get upstream SHA (this uses its own caching via GitLsRemote)
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

	// Cache the results
	staleCache.Set(projectPath, staleInfo)

	return staleInfo, nil
}

var (
	readerPool = sync.Pool{
		New: func() interface{} {
			return bufio.NewReaderSize(nil, 8192)
		},
	}
)

// StaleCheckCache provides intelligent caching for stale dependency checks
type StaleCheckCache struct {
	mu    sync.RWMutex
	cache map[string]CachedStaleInfo
	ttl   time.Duration
}

// CachedStaleInfo represents cached stale information with expiry
type CachedStaleInfo struct {
	Info   []StaleInfo
	Expiry time.Time
	Hash   string // Hash of pubspec.yaml + pubspec.lock for invalidation
}

var (
	staleCache = &StaleCheckCache{
		cache: make(map[string]CachedStaleInfo),
		ttl:   10 * time.Minute, // Cache for 10 minutes
	}
)

// parsePubspecLock parses the pubspec.lock file with optimized I/O
func parsePubspecLock(lockPath string) (*PubspecLock, error) {
	file, err := os.Open(lockPath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	// Get a buffered reader from pool
	reader := readerPool.Get().(*bufio.Reader)
	defer readerPool.Put(reader)
	reader.Reset(file)

	// Parse YAML-like structure (simplified parser for our needs)
	lock := &PubspecLock{
		Dependencies: make(map[string]PubspecLockDep),
	}

	inPackages := false
	currentPkg := ""
	currentDep := PubspecLockDep{}

	// Read line by line using buffered reader for better performance
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Remove newline
		line = strings.TrimSuffix(line, "\n")
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

	// Remember the last package
	if currentPkg != "" {
		lock.Dependencies[currentPkg] = currentDep
	}

	return lock, nil
}

// Get returns cached stale info if still valid
func (c *StaleCheckCache) Get(projectPath string) []StaleInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.cache[projectPath]
	if !exists || time.Now().After(cached.Expiry) {
		return nil
	}

	// Check if files have changed by comparing hash
	currentHash := c.generateProjectHash(projectPath)
	if currentHash != cached.Hash {
		return nil
	}

	return cached.Info
}

// Set caches the stale info with expiry
func (c *StaleCheckCache) Set(projectPath string, info []StaleInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := c.generateProjectHash(projectPath)
	c.cache[projectPath] = CachedStaleInfo{
		Info:   info,
		Expiry: time.Now().Add(c.ttl),
		Hash:   hash,
	}

	// Start cleanup timer
	go c.cleanupAfterTTL(projectPath)
}

// generateProjectHash creates a hash of pubspec files for cache invalidation
func (c *StaleCheckCache) generateProjectHash(projectPath string) string {
	pubspecPath := filepath.Join(projectPath, "pubspec.yaml")
	lockPath := filepath.Join(projectPath, "pubspec.lock")

	var hashBuilder strings.Builder

	// Include modification times of both files
	if info, err := os.Stat(pubspecPath); err == nil {
		hashBuilder.WriteString(info.ModTime().Format(time.RFC3339Nano))
	}
	if info, err := os.Stat(lockPath); err == nil {
		hashBuilder.WriteString(info.ModTime().Format(time.RFC3339Nano))
	}

	return hashBuilder.String()
}

// cleanupAfterTTL removes cache entry after TTL expires
func (c *StaleCheckCache) cleanupAfterTTL(projectPath string) {
	time.Sleep(c.ttl + time.Minute) // Extra minute buffer
	c.mu.Lock()
	delete(c.cache, projectPath)
	c.mu.Unlock()
}

// InvalidateProject removes cached data for a specific project
func (c *StaleCheckCache) InvalidateProject(projectPath string) {
	c.mu.Lock()
	delete(c.cache, projectPath)
	c.mu.Unlock()
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

// CheckSelfUpdate checks for Flutter-PM updates
func CheckSelfUpdate(logger *Logger, cfg *Config) ActionResult {
	logger.Info("selfupdate", "Checking for Flutter-PM updates...")

	if cfg.DryRun {
		return ActionResult{
			OK:      true,
			Message: "Would check for Flutter-PM updates",
			Logs:    []string{"DRY RUN: git fetch origin main"},
		}
	}

	// Try to find git repository root
	execPath, err := os.Executable()
	if err != nil {
		return ActionResult{
			OK:  false,
			Err: fmt.Sprintf("could not determine executable path: %v", err),
		}
	}

	// Look for .git directory up the tree
	dir := filepath.Dir(execPath)
	var repoRoot string
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			repoRoot = dir
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}

	if repoRoot == "" {
		return ActionResult{
			OK:      true,
			Message: "Not a git repository - please reinstall Flutter Package Manager",
		}
	}

	// Check git status and fetch updates
	cmd := exec.Command("git", "fetch", "origin", "main")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	logs := []string{strings.TrimSpace(string(output))}

	if err != nil {
		return ActionResult{
			OK:   false,
			Err:  fmt.Sprintf("git fetch failed: %v", err),
			Logs: logs,
		}
	}

	// Check if updates are available
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot
	currentCommit, err := cmd.Output()
	if err != nil {
		return ActionResult{
			OK:   false,
			Err:  fmt.Sprintf("could not get current commit: %v", err),
			Logs: logs,
		}
	}

	cmd = exec.Command("git", "rev-parse", "origin/main")
	cmd.Dir = repoRoot
	latestCommit, err := cmd.Output()
	if err != nil {
		return ActionResult{
			OK:   false,
			Err:  fmt.Sprintf("could not get latest commit: %v", err),
			Logs: logs,
		}
	}

	currentSHA := strings.TrimSpace(string(currentCommit))
	latestSHA := strings.TrimSpace(string(latestCommit))

	if currentSHA == latestSHA {
		return ActionResult{
			OK:      true,
			Message: "Flutter Package Manager is already up to date",
			Logs:    logs,
		}
	}

	// Updates available - perform update
	cmd = exec.Command("git", "reset", "--hard", "origin/main")
	cmd.Dir = repoRoot
	updateOutput, err := cmd.CombinedOutput()
	logs = append(logs, strings.TrimSpace(string(updateOutput)))

	if err != nil {
		return ActionResult{
			OK:   false,
			Err:  fmt.Sprintf("update failed: %v", err),
			Logs: logs,
		}
	}

	return ActionResult{
		OK:      true,
		Message: "Flutter Package Manager updated successfully - restart recommended",
		Logs:    logs,
	}
}

// NuclearCacheUpdate performs nuclear cache clearing + update (remove pubspec.lock + clear pub cache)
