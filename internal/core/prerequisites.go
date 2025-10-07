// Package core/prerequisites.go - Prerequisites Checking and Auto-Installation
//
// This file provides comprehensive prerequisite checking and auto-installation
// functionality for the Flutter Package Manager. It validates that all required
// tools (git, dart/flutter, gh) are available and provides guidance for installation.

package core

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Prerequisite represents a required tool or dependency
type Prerequisite struct {
	Name        string
	Command     string
	CheckArgs   []string
	Required    bool
	Description string
	InstallURL  string
	InstallCmds map[string]string // OS -> command
}

// PrerequisiteResult represents the result of checking a prerequisite
type PrerequisiteResult struct {
	Name       string
	Available  bool
	Version    string
	Error      string
	InstallCmd string
}

// PrerequisiteCheck contains the overall prerequisite check results
type PrerequisiteCheck struct {
	AllMet   bool
	Results  []PrerequisiteResult
	Missing  []string
	Warnings []string
}

// GetPrerequisites returns the list of all prerequisites
func GetPrerequisites() []Prerequisite {
	return []Prerequisite{
		{
			Name:        "Git",
			Command:     "git",
			CheckArgs:   []string{"--version"},
			Required:    true,
			Description: "Git version control system for repository operations",
			InstallURL:  "https://git-scm.com/downloads",
			InstallCmds: map[string]string{
				"windows": "winget install Git.Git",
				"darwin":  "brew install git",
				"linux":   "sudo apt-get install git",
			},
		},
		{
			Name:        "Flutter",
			Command:     "flutter",
			CheckArgs:   []string{"--version"},
			Required:    false, // Either Flutter or Dart is sufficient
			Description: "Flutter SDK for Flutter project management",
			InstallURL:  "https://flutter.dev/docs/get-started/install",
			InstallCmds: map[string]string{
				"windows": "winget install Google.Flutter",
				"darwin":  "brew install --cask flutter",
				"linux":   "snap install flutter --classic",
			},
		},
		{
			Name:        "Dart",
			Command:     "dart",
			CheckArgs:   []string{"--version"},
			Required:    false, // Either Flutter or Dart is sufficient
			Description: "Dart SDK for Dart project management",
			InstallURL:  "https://dart.dev/get-dart",
			InstallCmds: map[string]string{
				"windows": "winget install Dart.DartSDK",
				"darwin":  "brew install dart",
				"linux":   "sudo apt-get install dart",
			},
		},
		{
			Name:        "GitHub CLI",
			Command:     "gh",
			CheckArgs:   []string{"--version"},
			Required:    false, // Optional but recommended
			Description: "GitHub CLI for repository browsing and authentication",
			InstallURL:  "https://cli.github.com/",
			InstallCmds: map[string]string{
				"windows": "winget install GitHub.cli",
				"darwin":  "brew install gh",
				"linux":   "sudo apt-get install gh",
			},
		},
	}
}

// CheckPrerequisites validates all prerequisites and returns detailed results
func CheckPrerequisites(logger *Logger) PrerequisiteCheck {
	prerequisites := GetPrerequisites()
	results := make([]PrerequisiteResult, 0, len(prerequisites))
	missing := make([]string, 0)
	warnings := make([]string, 0)

	logger.Debug("prerequisites", "Checking all prerequisites")

	// Check each prerequisite
	for _, prereq := range prerequisites {
		result := checkSinglePrerequisite(prereq)
		results = append(results, result)

		if !result.Available {
			if prereq.Required {
				missing = append(missing, prereq.Name)
			} else {
				warnings = append(warnings, fmt.Sprintf("%s not available (optional)", prereq.Name))
			}
		}

		logger.Debug("prerequisites", fmt.Sprintf("%s: %t (%s)", prereq.Name, result.Available, result.Version))
	}

	// Special case: Either Flutter or Dart must be available
	dartResult := findResult(results, "Dart")
	flutterResult := findResult(results, "Flutter")

	if dartResult != nil && flutterResult != nil {
		if !dartResult.Available && !flutterResult.Available {
			missing = append(missing, "Flutter or Dart")
		} else if dartResult.Available || flutterResult.Available {
			// Remove from missing if either is available
			missing = removeFromSlice(missing, "Dart")
			missing = removeFromSlice(missing, "Flutter")
		}
	}

	allMet := len(missing) == 0

	logger.Info("prerequisites", fmt.Sprintf("Prerequisites check complete. All met: %t, Missing: %d, Warnings: %d",
		allMet, len(missing), len(warnings)))

	return PrerequisiteCheck{
		AllMet:   allMet,
		Results:  results,
		Missing:  missing,
		Warnings: warnings,
	}
}

// checkSinglePrerequisite checks if a single prerequisite is available
func checkSinglePrerequisite(prereq Prerequisite) PrerequisiteResult {
	result := PrerequisiteResult{
		Name:      prereq.Name,
		Available: false,
		Version:   "",
		Error:     "",
	}

	// Get install command for current OS
	osName := runtime.GOOS
	if installCmd, exists := prereq.InstallCmds[osName]; exists {
		result.InstallCmd = installCmd
	} else {
		result.InstallCmd = fmt.Sprintf("Please visit: %s", prereq.InstallURL)
	}

	// Check if command exists
	_, err := exec.LookPath(prereq.Command)
	if err != nil {
		result.Error = fmt.Sprintf("Command '%s' not found in PATH", prereq.Command)
		return result
	}

	// Run version check
	cmd := exec.Command(prereq.Command, prereq.CheckArgs...)
	output, err := cmd.Output()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to run '%s %s': %v", prereq.Command, strings.Join(prereq.CheckArgs, " "), err)
		return result
	}

	result.Available = true
	result.Version = strings.TrimSpace(string(output))

	return result
}

// AutoInstallPrerequisites attempts to auto-install missing prerequisites (where possible)
func AutoInstallPrerequisites(logger *Logger, missing []string, dryRun bool) ActionResult {
	if len(missing) == 0 {
		return ActionResult{
			OK:      true,
			Message: "All prerequisites are already available",
		}
	}

	logs := []string{}
	osName := runtime.GOOS

	// For now, just provide installation guidance
	// Auto-installation would require elevated privileges

	if dryRun {
		for _, name := range missing {
			prereq := findPrerequisite(name)
			if prereq != nil {
				if installCmd, exists := prereq.InstallCmds[osName]; exists {
					logs = append(logs, fmt.Sprintf("Would run: %s", installCmd))
				}
			}
		}

		return ActionResult{
			OK:      true,
			Message: fmt.Sprintf("Would attempt to install %d missing prerequisites", len(missing)),
			Logs:    logs,
		}
	}

	// Provide installation guidance
	logger.Info("prerequisites", "Providing installation guidance for missing prerequisites")

	for _, name := range missing {
		prereq := findPrerequisite(name)
		if prereq != nil {
			if installCmd, exists := prereq.InstallCmds[osName]; exists {
				logs = append(logs, fmt.Sprintf("To install %s: %s", name, installCmd))
			} else {
				logs = append(logs, fmt.Sprintf("To install %s: Visit %s", name, prereq.InstallURL))
			}
		}
	}

	return ActionResult{
		OK:      false,
		Message: fmt.Sprintf("Cannot auto-install prerequisites. Manual installation required for: %s", strings.Join(missing, ", ")),
		Logs:    logs,
		Err:     "Manual installation required",
	}
}

// GetInstallationGuidance returns formatted installation guidance
func GetInstallationGuidance(check PrerequisiteCheck) []string {
	if check.AllMet {
		return []string{"✅ All prerequisites are available!"}
	}

	guidance := []string{}
	osName := runtime.GOOS
	osDisplay := map[string]string{
		"windows": "Windows",
		"darwin":  "macOS",
		"linux":   "Linux",
	}[osName]

	if len(check.Missing) > 0 {
		guidance = append(guidance, fmt.Sprintf("❌ Missing required prerequisites (%s):", osDisplay))
		guidance = append(guidance, "")

		for _, name := range check.Missing {
			prereq := findPrerequisite(name)
			if prereq != nil {
				guidance = append(guidance, fmt.Sprintf("📦 %s - %s", name, prereq.Description))
				if installCmd, exists := prereq.InstallCmds[osName]; exists {
					guidance = append(guidance, fmt.Sprintf("   Install: %s", installCmd))
				} else {
					guidance = append(guidance, fmt.Sprintf("   Install: Visit %s", prereq.InstallURL))
				}
				guidance = append(guidance, "")
			}
		}
	}

	if len(check.Warnings) > 0 {
		guidance = append(guidance, "⚠️ Optional tools not available:")
		for _, warning := range check.Warnings {
			guidance = append(guidance, fmt.Sprintf("   %s", warning))
		}
		guidance = append(guidance, "")
	}

	guidance = append(guidance, "💡 After installation, restart your terminal and try again.")

	return guidance
}

// Utility functions

func findResult(results []PrerequisiteResult, name string) *PrerequisiteResult {
	for i := range results {
		if results[i].Name == name {
			return &results[i]
		}
	}
	return nil
}

func findPrerequisite(name string) *Prerequisite {
	prereqs := GetPrerequisites()
	for i := range prereqs {
		if prereqs[i].Name == name {
			return &prereqs[i]
		}
	}
	return nil
}

func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
