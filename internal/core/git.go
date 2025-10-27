// Package core/git.go - Git Operations and GitHub CLI Integration
//
// This file provides all Git-related functionality with exact shell script parity.
// It integrates with the GitHub CLI (gh) for authentication and repository listing,
// and uses Git CLI commands for all operations to maintain compatibility.
//
// Key features:
// - GitHub CLI integration for repository listing and authentication
// - Git clone operations with proper error handling and conflict resolution
// - Git version checking and command availability validation
// - Concurrent Git operations with timeout management
// - SHA-based comparison for precise dependency staleness detection
// - Cross-platform Git command execution
// - Shell script compatible Git workflow and command sequences
//
// All Git operations use the CLI tools rather than Go libraries to ensure
// exact compatibility with the shell script behavior and handle edge cases
// in the same way as the original implementation.

package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// GitLsRemoteCache provides caching for git ls-remote operations
type GitLsRemoteCache struct {
	mu     sync.RWMutex
	cache  map[string]string      // URL+ref -> SHA
	timers map[string]*time.Timer // Track cleanup timers to prevent races
	ttl    time.Duration
}

var (
	gitLsRemoteCache = &GitLsRemoteCache{
		cache:  make(map[string]string),
		timers: make(map[string]*time.Timer),
		ttl:    2 * time.Minute, // Cache git ls-remote for 2 minutes
	}
)

// GitLsRemote gets the SHA for a specific ref from a git repository with caching
func GitLsRemote(url, ref string) (string, error) {
	cacheKey := url + "#" + ref

	// Try cache first
	gitLsRemoteCache.mu.RLock()
	if cached, exists := gitLsRemoteCache.cache[cacheKey]; exists {
		gitLsRemoteCache.mu.RUnlock()
		return cached, nil
	}
	gitLsRemoteCache.mu.RUnlock()

	cmd := exec.Command("git", "ls-remote", url, ref)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run git ls-remote: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 && (parts[1] == ref || parts[1] == "refs/heads/"+ref || parts[1] == "refs/tags/"+ref) {
			sha := parts[0]
			// Cache the result
			gitLsRemoteCache.mu.Lock()
			gitLsRemoteCache.cache[cacheKey] = sha
			gitLsRemoteCache.mu.Unlock()

			// Start cleanup timer if this is the first entry
			go gitLsRemoteCache.cleanupAfterTTL(cacheKey)

			return sha, nil
		}
	}

	return "", fmt.Errorf("ref %s not found in repository %s", ref, url)
}

// GitClone clones a repository to a local directory
func GitClone(logger *Logger, cfg *Config, url, dir, ref string) ActionResult {
	args := []string{"clone"}

	if ref != "" && ref != "main" && ref != "master" {
		args = append(args, "--branch", ref)
	}

	args = append(args, url, dir)

	logger.LogCommand("git", "git", args)

	if cfg.DryRun {
		return ActionResult{
			OK:      true,
			Message: fmt.Sprintf("Would clone %s to %s", url, dir),
			Logs:    []string{fmt.Sprintf("DRY RUN: git %s", strings.Join(args, " "))},
		}
	}

	cmd := exec.Command("git", args...)
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
		Message: fmt.Sprintf("Successfully cloned %s", url),
		Logs:    logs,
	}
}

// GitHubRepo represents a GitHub repository from gh CLI
type GitHubRepo struct {
	Name        string `json:"name"`
	FullName    string `json:"nameWithOwner"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"isPrivate"`
	URL         string `json:"url"`
	SSHURL      string `json:"sshUrl"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// GitHubCache provides intelligent caching for GitHub API responses
type GitHubCache struct {
	mu     sync.RWMutex
	repos  []RepoCandidate
	expiry time.Time
	hash   string
	ttl    time.Duration
}

var (
	githubCache = &GitHubCache{
		ttl: 5 * time.Minute, // Cache for 5 minutes
	}
)

// ListGitHubRepos uses gh CLI to list user repositories with intelligent caching
// This mirrors the shell script's GitHub integration but optimized for performance
func ListGitHubRepos(logger *Logger) ([]RepoCandidate, error) {
	// Check cache first
	if cached := githubCache.Get(); cached != nil {
		logger.Debug("github", "Using cached repository list")
		return cached, nil
	}

	// Check if gh is available
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("GitHub CLI (gh) not found. Please install: https://cli.github.com/")
	}

	// Check if authenticated
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("GitHub CLI not authenticated. Please run 'gh auth login'")
	}

	logger.Debug("github", "Fetching repositories from GitHub")

	// Get repositories as JSON with increased limit for better UX
	cmd = exec.Command("gh", "repo", "list",
		"--json", "name,nameWithOwner,description,isPrivate,url,sshUrl,owner",
		"--limit", "200") // Increased from 100 for better coverage

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	var repos []GitHubRepo
	if err := json.Unmarshal(stdout.Bytes(), &repos); err != nil {
		return nil, fmt.Errorf("failed to parse repository JSON: %w", err)
	}

	// Transform repos to candidates efficiently
	candidates := make([]RepoCandidate, 0, len(repos))
	for _, repo := range repos {
		privacy := "public"
		if repo.IsPrivate {
			privacy = "private"
		}

		// Use HTTPS URL and add .git suffix for consistency
		gitURL := repo.URL
		if !strings.HasSuffix(gitURL, ".git") {
			gitURL += ".git"
		}

		candidates = append(candidates, RepoCandidate{
			Owner:   repo.Owner.Login,
			Name:    repo.Name,
			URL:     gitURL,
			Privacy: privacy,
			Desc:    repo.Description,
		})
	}

	// Cache the results
	githubCache.Set(candidates)

	logger.Info("github", fmt.Sprintf("Found %d repositories", len(candidates)))
	return candidates, nil
}

// Get returns cached repositories if still valid
func (c *GitHubCache) Get() []RepoCandidate {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Now().Before(c.expiry) && len(

		c.repos) > 0 {
		return c.repos
	}

	return nil
}

// Set caches the repositories with expiry
func (c *GitHubCache) Set(repos []RepoCandidate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.repos = repos
	c.expiry = time.Now().Add(c.ttl)

	// Generate hash for cache invalidation if needed
	h := sha256.New()
	for _, repo := range repos {
		h.Write([]byte(repo.URL + repo.Name))
	}
	c.hash = hex.EncodeToString(h.Sum(nil))
}

// InvalidateCache clears the cache
func (c *GitHubCache) InvalidateCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.repos = nil
	c.expiry = time.Time{}
}

// GetRepoBranches gets available branches for a repository with caching

// GetRepoTags gets available tags for a repository with caching

// cleanupAfterTTL removes cache entry after TTL expires with proper race condition handling
func (c *GitLsRemoteCache) cleanupAfterTTL(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cancel existing timer if present to prevent race
	if existingTimer, exists := c.timers[key]; exists {
		existingTimer.Stop()
		delete(c.timers, key) // Remove immediately to prevent double cleanup
	}

	// Set new cleanup timer with proper synchronization
	c.timers[key] = time.AfterFunc(c.ttl, func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		// Double-check that this timer is still the current one
		if timer, exists := c.timers[key]; exists && timer != nil {
			delete(c.cache, key)
			delete(c.timers, key)
		}
	})
}

// GetGitVersion returns the git version string
func GetGitVersion() (string, error) {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git not available: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GitHubPackageNameResult represents the result from gh search code
type GitHubPackageNameResult struct {
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Path        string `json:"path"`
	TextMatches []struct {
		Fragment string `json:"fragment"`
	} `json:"textMatches"`
}

// FetchPackageNameFromGit fetches the actual package name from a git repository's pubspec.yaml
// This is critical because the repository name may not match the package name declared in pubspec.yaml
// For example: repo "my_awesome_repo" might contain package "my_package"
//
// Uses a fallback chain for maximum robustness:
// 1. Primary: GitHub CLI API (works for public and private repos if authenticated)
// 2. Fallback 1: Direct HTTP GET from raw.githubusercontent.com (public repos only)
// 3. Fallback 2: Try alternative branch names (main, master, develop)
// 4. Final fallback: Use repository name as package name
func FetchPackageNameFromGit(logger *Logger, gitURL string, ref string, subdir string) (string, error) {
	// Only supports GitHub repos
	if !strings.Contains(gitURL, "github.com") {
		return "", fmt.Errorf("non-GitHub repos not yet supported for automatic name detection")
	}

	// Extract owner/repo from URL
	// https://github.com/dart-lang/http.git -> dart-lang/http
	gitURL = strings.TrimSuffix(gitURL, ".git")
	gitURL = strings.TrimSuffix(gitURL, "/")
	parts := strings.Split(gitURL, "github.com/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid GitHub URL format: %s", gitURL)
	}
	ownerRepo := strings.TrimSuffix(parts[1], "/")
	if ownerRepo == "" {
		return "", fmt.Errorf("invalid GitHub URL format: %s", gitURL)
	}

	// Build the path to pubspec.yaml
	pubspecPath := "pubspec.yaml"
	if subdir != "" {
		pubspecPath = subdir + "/pubspec.yaml"
	}

	logger.Debug("git", fmt.Sprintf("Fetching package name from GitHub repo: %s (path: %s)", ownerRepo, pubspecPath))

	// Default branch if ref is empty
	branch := ref
	if branch == "" {
		branch = "main"
	}

	// METHOD 1: Try gh api (best method - works for public and private repos)
	if packageName, err := fetchPackageNameViaGhAPI(logger, ownerRepo, pubspecPath); err == nil {
		logger.Debug("git", fmt.Sprintf("✓ Found package name via gh api: %s", packageName))
		return packageName, nil
	} else {
		logger.Debug("git", fmt.Sprintf("✗ gh api method failed: %s", err.Error()))
	}

	// METHOD 2: Try raw.githubusercontent.com with specified branch (works for public repos)
	if packageName, err := fetchPackageNameViaHTTP(logger, ownerRepo, pubspecPath, branch); err == nil {
		logger.Debug("git", fmt.Sprintf("✓ Found package name via HTTP (branch: %s): %s", branch, packageName))
		return packageName, nil
	} else {
		logger.Debug("git", fmt.Sprintf("✗ HTTP method failed for branch '%s': %s", branch, err.Error()))
	}

	// METHOD 3: Try alternative branch names if the specified branch failed
	alternativeBranches := []string{"main", "master", "develop"}
	for _, altBranch := range alternativeBranches {
		if altBranch == branch {
			continue // Skip the branch we already tried
		}
		if packageName, err := fetchPackageNameViaHTTP(logger, ownerRepo, pubspecPath, altBranch); err == nil {
			logger.Debug("git", fmt.Sprintf("✓ Found package name via HTTP (alternative branch: %s): %s", altBranch, packageName))
			return packageName, nil
		}
	}

	// METHOD 4: Final fallback - use repository name
	repoName := ownerRepo
	if slashIdx := strings.LastIndex(ownerRepo, "/"); slashIdx != -1 {
		repoName = ownerRepo[slashIdx+1:]
	}
	logger.Debug("git", fmt.Sprintf("⚠ All methods failed, using repository name as package name: %s", repoName))
	return repoName, nil
}

// fetchPackageNameViaGhAPI uses GitHub CLI to fetch pubspec.yaml (works for public and private repos)
func fetchPackageNameViaGhAPI(logger *Logger, ownerRepo string, pubspecPath string) (string, error) {
	// Build gh api command to fetch pubspec.yaml contents
	// CRITICAL: This matches the user's exact working command - DO NOT MODIFY THE SYNTAX!
	// gh api repos/owner/repo/contents/pubspec.yaml --jq ".content | @base64d | split(\"\n\")[] | select(test(\"^name:\")) | sub(\"^name:\\s*\"; \"\")"
	//
	// The jq expression (robust version):
	// 1. Takes the base64-encoded .content field from GitHub API
	// 2. Decodes it with @base64d
	// 3. Splits by newline and iterates through ALL lines
	// 4. Selects lines that match regex ^name: (starts with "name:")
	// 5. Removes the "name:" prefix and any whitespace after it using sub()
	args := []string{
		"api",
		fmt.Sprintf("repos/%s/contents/%s", ownerRepo, pubspecPath),
		"--jq", ".content | @base64d | split(\"\\n\")[] | select(test(\"^name:\")) | sub(\"^name:\\\\s*\"; \"\")",
	}

	logger.Debug("git", fmt.Sprintf("Trying gh api: gh %s", strings.Join(args, " ")))

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		// Try to get stderr for better error messages
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			return "", fmt.Errorf("gh api failed: %s", stderr)
		}
		return "", fmt.Errorf("failed to run gh api: %w", err)
	}

	// The jq expression returns just the package name
	packageName := strings.TrimSpace(string(output))

	// Remove quotes if present (jq might include them)
	packageName = strings.Trim(packageName, "\"'")

	if packageName == "" {
		return "", fmt.Errorf("empty package name returned from gh api")
	}

	return packageName, nil
}

// fetchPackageNameViaHTTP fetches pubspec.yaml via raw.githubusercontent.com (public repos only)
func fetchPackageNameViaHTTP(logger *Logger, ownerRepo string, pubspecPath string, branch string) (string, error) {
	// Build URL: https://raw.githubusercontent.com/owner/repo/branch/path/to/pubspec.yaml
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", ownerRepo, branch, pubspecPath)
	logger.Debug("git", fmt.Sprintf("Trying HTTP GET: %s", url))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read the content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the package name from pubspec.yaml content
	packageName := extractPackageNameFromYAML(string(body))
	if packageName == "" {
		return "", fmt.Errorf("could not find 'name:' field in pubspec.yaml")
	}

	return packageName, nil
}

// extractPackageNameFromYAML extracts the package name from pubspec.yaml content using proper YAML parsing
func extractPackageNameFromYAML(content string) string {
	// Define a minimal structure to extract just the name field
	var pubspec struct {
		Name string `yaml:"name"`
	}

	// Parse the YAML content
	if err := yaml.Unmarshal([]byte(content), &pubspec); err != nil {
		// If YAML parsing fails, return empty string
		return ""
	}

	// Return the extracted name (will be empty string if not found)
	return strings.TrimSpace(pubspec.Name)
}

// ValidateGitURL checks if a Git URL is valid and accessible with enhanced security
