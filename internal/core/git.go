package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitLsRemote gets the SHA for a specific ref from a git repository
func GitLsRemote(url, ref string) (string, error) {
	cmd := exec.Command("git", "ls-remote", url, ref)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run git ls-remote: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 && (parts[1] == ref || parts[1] == "refs/heads/"+ref || parts[1] == "refs/tags/"+ref) {
			return parts[0], nil
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

// ListGitHubRepos uses gh CLI to list user repositories
// This mirrors the shell script's GitHub integration
func ListGitHubRepos(logger *Logger) ([]RepoCandidate, error) {
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

	// Get repositories as JSON
	cmd = exec.Command("gh", "repo", "list",
		"--json", "name,nameWithOwner,description,isPrivate,url,sshUrl,owner",
		"--limit", "100")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	var repos []GitHubRepo
	if err := json.Unmarshal(stdout.Bytes(), &repos); err != nil {
		return nil, fmt.Errorf("failed to parse repository JSON: %w", err)
	}

	var candidates []RepoCandidate
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

	logger.Info("github", fmt.Sprintf("Found %d repositories", len(candidates)))
	return candidates, nil
}

// GetRepoBranches gets available branches for a repository
func GetRepoBranches(repoURL string) ([]string, error) {
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
		return []string{"main"}, nil // Default fallback
	}

	return branches, nil
}

// GetRepoTags gets available tags for a repository
func GetRepoTags(repoURL string) ([]string, error) {
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

	return tags, nil
}

// ValidateGitURL checks if a Git URL is valid and accessible
func ValidateGitURL(url string) error {
	cmd := exec.Command("git", "ls-remote", "--exit-code", url, "HEAD")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("invalid or inaccessible git URL: %s", url)
	}
	return nil
}
