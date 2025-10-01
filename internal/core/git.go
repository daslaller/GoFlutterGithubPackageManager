package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
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

// GitInit initializes a new git repository
func GitInit(projectPath string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectPath
	return cmd.Run()
}

// IsGitRepository checks if a directory is a git repository
func IsGitRepository(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	return cmd.Run() == nil
}

// GetGitRemotes gets the remote URLs for a git repository
func GetGitRemotes(repoPath string) (map[string]string, error) {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git remotes: %w", err)
	}

	remotes := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.HasSuffix(line, "(fetch)") {
			remotes[parts[0]] = parts[1]
		}
	}

	return remotes, nil
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

	if time.Now().Before(c.expiry) && len(c.repos) > 0 {
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
func GetRepoBranches(repoURL string) ([]string, error) {
	cacheKey := "branches:" + repoURL

	// Try cache first (branches don't change frequently)
	gitLsRemoteCache.mu.RLock()
	if cached, exists := gitLsRemoteCache.cache[cacheKey]; exists {
		gitLsRemoteCache.mu.RUnlock()
		return strings.Split(cached, ","), nil
	}
	gitLsRemoteCache.mu.RUnlock()

	cmd := exec.Command("git", "ls-remote", "--heads", repoURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ref := parts[1]
			if strings.HasPrefix(ref, "refs/heads/") {
				branch := strings.TrimPrefix(ref, "refs/heads/")
				branches = append(branches, branch)
			}
		}
	}

	if len(branches) == 0 {
		branches = []string{"main"} // Default fallback
	}

	// Cache the results
	gitLsRemoteCache.mu.Lock()
	gitLsRemoteCache.cache[cacheKey] = strings.Join(branches, ",")
	gitLsRemoteCache.mu.Unlock()
	go gitLsRemoteCache.cleanupAfterTTL(cacheKey)

	return branches, nil
}

// GetRepoTags gets available tags for a repository with caching
func GetRepoTags(repoURL string) ([]string, error) {
	cacheKey := "tags:" + repoURL

	// Try cache first (tags are immutable once created)
	gitLsRemoteCache.mu.RLock()
	if cached, exists := gitLsRemoteCache.cache[cacheKey]; exists {
		gitLsRemoteCache.mu.RUnlock()
		if cached == "" {
			return []string{}, nil
		}
		return strings.Split(cached, ","), nil
	}
	gitLsRemoteCache.mu.RUnlock()

	cmd := exec.Command("git", "ls-remote", "--tags", repoURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	var tags []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ref := parts[1]
			if strings.HasPrefix(ref, "refs/tags/") && !strings.HasSuffix(ref, "^{}") {
				tag := strings.TrimPrefix(ref, "refs/tags/")
				tags = append(tags, tag)
			}
		}
	}

	// Cache the results
	cacheValue := strings.Join(tags, ",")
	gitLsRemoteCache.mu.Lock()
	gitLsRemoteCache.cache[cacheKey] = cacheValue
	gitLsRemoteCache.mu.Unlock()
	go gitLsRemoteCache.cleanupAfterTTL(cacheKey)

	return tags, nil
}

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

// ValidateGitURL checks if a Git URL is valid and accessible with enhanced security
func ValidateGitURL(url string) error {
	// Input validation to prevent command injection
	if url == "" {
		return fmt.Errorf("empty git URL")
	}

	// Enhanced validation to prevent command injection
	if strings.ContainsAny(url, ";|&$`\n\r\t'\"\\") {
		return fmt.Errorf("invalid characters in git URL")
	}

	// Check for common protocol patterns (more restrictive)
	validPrefixes := []string{"https://", "http://", "git@", "ssh://"}
	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(strings.ToLower(url), prefix) {
			hasValidPrefix = true
			break
		}
	}
	if !hasValidPrefix {
		return fmt.Errorf("invalid git URL protocol (must start with https://, http://, git@, or ssh://)")
	}

	// Limit URL length to prevent buffer overflow attacks
	if len(url) > 2048 {
		return fmt.Errorf("git URL too long (max 2048 characters)")
	}

	// Use timeout and secure environment for git command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--exit-code", url, "HEAD")
	cmd.Env = []string{
		"GIT_TERMINAL_PROMPT=0", // Disable interactive prompts
		"GIT_ASKPASS=true",      // Disable password prompts
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("invalid or inaccessible git URL: %s", url)
	}
	return nil
}
