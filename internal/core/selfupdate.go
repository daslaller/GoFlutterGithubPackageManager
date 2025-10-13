// Package core/selfupdate.go - Self-Update Functionality
//
// This file implements self-update logic that checks for newer versions on GitHub
// releases and downloads/installs the latest binary.

package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// CurrentVersion is the current version of flutter-pm
	CurrentVersion = "v1.0.0-alpha"

	// GitHubAPIBase is the base URL for GitHub API
	GitHubAPIBase = "https://api.github.com"

	// GitHubRepoPath is the repository path for flutter-pm
	GitHubRepoPath = "daslaller/GoFlutterGithubPackageManager"
)

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	Available      bool   // Whether an update is available
	CurrentVersion string // Current installed version
	LatestVersion  string // Latest version available
	DownloadURL    string // URL to download the update
	ReleaseNotes   string // Release notes for the update
	AssetName      string // Name of the asset file
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// CheckForUpdates checks if a new version is available on GitHub releases
func CheckForUpdates(logger *Logger) (UpdateInfo, error) {
	info := UpdateInfo{
		CurrentVersion: CurrentVersion,
		Available:      false,
	}

	// Fetch latest release from GitHub API
	url := fmt.Sprintf("%s/repos/%s/releases/latest", GitHubAPIBase, GitHubRepoPath)
	logger.Debug("selfupdate", fmt.Sprintf("Checking for updates at: %s", url))

	resp, err := http.Get(url)
	if err != nil {
		return info, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return info, fmt.Errorf("failed to parse release info: %w", err)
	}

	info.LatestVersion = release.TagName
	info.ReleaseNotes = release.Body

	// Compare versions (simple string comparison for now)
	if release.TagName != CurrentVersion {
		info.Available = true

		// Find the appropriate asset for this platform
		assetName := getAssetName()
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				info.DownloadURL = asset.BrowserDownloadURL
				info.AssetName = asset.Name
				logger.Debug("selfupdate", fmt.Sprintf("Found update: %s -> %s", CurrentVersion, release.TagName))
				break
			}
		}

		if info.DownloadURL == "" {
			return info, fmt.Errorf("no compatible binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	}

	return info, nil
}

// PerformUpdate downloads and installs the update
func PerformUpdate(info UpdateInfo, logger *Logger) error {
	if !info.Available {
		return fmt.Errorf("no update available")
	}

	logger.Info("selfupdate", fmt.Sprintf("Downloading update from: %s", info.DownloadURL))

	// Download the new binary
	resp, err := http.Get(info.DownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Get the current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create a backup of the current binary
	backupPath := exePath + ".backup"
	if err := copyFile(exePath, backupPath); err != nil {
		logger.Info("selfupdate", fmt.Sprintf("Failed to create backup: %v", err))
	} else {
		logger.Info("selfupdate", fmt.Sprintf("Created backup at: %s", backupPath))
	}

	// Create a temporary file for the new binary
	tmpPath := exePath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Download to temporary file
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write update: %w", err)
	}

	// Make it executable (Unix-like systems)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to make executable: %w", err)
		}
	}

	// Replace the current binary
	// On Windows, we need to rename the old file first
	if runtime.GOOS == "windows" {
		oldPath := exePath + ".old"
		os.Remove(oldPath) // Remove any existing .old file

		if err := os.Rename(exePath, oldPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to rename current binary: %w", err)
		}

		if err := os.Rename(tmpPath, exePath); err != nil {
			// Try to restore the old binary
			os.Rename(oldPath, exePath)
			return fmt.Errorf("failed to install update: %w", err)
		}

		// Schedule deletion of old binary on reboot (Windows)
		os.Remove(oldPath)
	} else {
		// Unix-like systems can replace the file directly
		if err := os.Rename(tmpPath, exePath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to install update: %w", err)
		}
	}

	logger.Info("selfupdate", fmt.Sprintf("Successfully updated to %s", info.LatestVersion))
	return nil
}

// getAssetName returns the asset name for the current platform
func getAssetName() string {
	var suffix string
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}

	return fmt.Sprintf("flutter-pm-%s-%s%s", runtime.GOOS, runtime.GOARCH, suffix)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// CompareVersions compares two version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Simple lexicographic comparison for now
	// This works for semantic versioning like "1.0.0", "1.0.1", etc.
	if v1 == v2 {
		return 0
	}
	if v1 < v2 {
		return -1
	}
	return 1
}

// GetExecutablePath returns the path of the current executable
func GetExecutablePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exePath)
}
