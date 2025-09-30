package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SuggestPopularPkgs provides recommendations for popular Flutter packages
// This is a starting point based on the shell script's recommendation system
func SuggestPopularPkgs() []Reco {
	return []Reco{
		{
			Message:   "Consider using http for network requests",
			Severity:  "info",
			Rationale: "The http package is the standard way to make HTTP requests in Flutter",
		},
		{
			Message:   "Consider using provider for state management",
			Severity:  "info",
			Rationale: "Provider is a popular and Flutter-team-recommended state management solution",
		},
		{
			Message:   "Consider using shared_preferences for local storage",
			Severity:  "info",
			Rationale: "SharedPreferences provides persistent storage for simple data",
		},
	}
}

// AnalyzeGitDependencies analyzes git dependencies and provides recommendations
func AnalyzeGitDependencies(logger *Logger, projectPath string) ([]Reco, error) {
	gitDeps, err := ListGitDependencies(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list git dependencies: %w", err)
	}

	var recommendations []Reco

	for _, dep := range gitDeps {
		// Check for floating branches
		if dep.Ref == "main" || dep.Ref == "master" || dep.Ref == "develop" {
			recommendations = append(recommendations, Reco{
				Message:   fmt.Sprintf("Pin %s to a specific tag or commit", dep.Name),
				Severity:  "warn",
				Rationale: fmt.Sprintf("Using floating branch '%s' can lead to unexpected changes. Consider pinning to a specific tag or commit SHA.", dep.Ref),
			})
		}

		// Check for GitHub URLs that could use SSH
		if strings.Contains(dep.URL, "https://github.com/") && !strings.Contains(dep.URL, "token") {
			sshURL := strings.Replace(dep.URL, "https://github.com/", "git@github.com:", 1)
			recommendations = append(recommendations, Reco{
				Message:   fmt.Sprintf("Consider using SSH URL for %s", dep.Name),
				Severity:  "info",
				Rationale: fmt.Sprintf("SSH URLs can be faster and more secure. Consider: %s", sshURL),
			})
		}

		// Check for subdirectory usage
		if dep.Subdir != "" {
			recommendations = append(recommendations, Reco{
				Message:   fmt.Sprintf("%s uses subdirectory - ensure it's necessary", dep.Name),
				Severity:  "info",
				Rationale: "Using subdirectories can slow down dependency resolution. Ensure this is the intended package structure.",
			})
		}
	}

	return recommendations, nil
}

// AnalyzePubspecStructure analyzes pubspec.yaml structure and provides recommendations
func AnalyzePubspecStructure(projectPath string) ([]Reco, error) {
	var recommendations []Reco

	// Check pubspec.yaml validation
	issues, err := ValidatePubspec(projectPath)
	if err != nil {
		return nil, err
	}

	for _, issue := range issues {
		recommendations = append(recommendations, Reco{
			Message:   "Pubspec validation issue: " + issue,
			Severity:  "warn",
			Rationale: "Fixing pubspec.yaml issues ensures proper dependency resolution",
		})
	}

	// Check for Flutter project structure
	libPath := filepath.Join(projectPath, "lib")
	testPath := filepath.Join(projectPath, "test")

	if !dirExists(libPath) {
		recommendations = append(recommendations, Reco{
			Message:   "Missing lib/ directory",
			Severity:  "error",
			Rationale: "Flutter projects require a lib/ directory containing Dart source code",
		})
	}

	if !dirExists(testPath) {
		recommendations = append(recommendations, Reco{
			Message:   "Consider adding tests to test/ directory",
			Severity:  "info",
			Rationale: "Having tests improves code quality and prevents regressions",
		})
	}

	return recommendations, nil
}

// AnalyzeGitRepository analyzes the git repository and provides recommendations
func AnalyzeGitRepository(projectPath string) ([]Reco, error) {
	var recommendations []Reco

	if !IsGitRepository(projectPath) {
		recommendations = append(recommendations, Reco{
			Message:   "Initialize Git repository",
			Severity:  "warn",
			Rationale: "Version control is essential for tracking changes and collaborating",
		})
		return recommendations, nil
	}

	// Check for gitignore
	gitignorePath := filepath.Join(projectPath, ".gitignore")
	if !fileExists(gitignorePath) {
		recommendations = append(recommendations, Reco{
			Message:   "Add .gitignore file",
			Severity:  "warn",
			Rationale: "Flutter projects generate build artifacts that shouldn't be committed",
		})
	}

	// Check for remotes
	remotes, err := GetGitRemotes(projectPath)
	if err != nil {
		return recommendations, nil // Continue without remote analysis
	}

	if len(remotes) == 0 {
		recommendations = append(recommendations, Reco{
			Message:   "Add remote repository",
			Severity:  "info",
			Rationale: "Adding a remote repository enables collaboration and backup",
		})
	}

	return recommendations, nil
}

// AnalyzePerformance analyzes for potential performance improvements
func AnalyzePerformance(projectPath string, gitDeps []PkgSpec) []Reco {
	var recommendations []Reco

	// Check for large number of git dependencies
	if len(gitDeps) > 10 {
		recommendations = append(recommendations, Reco{
			Message:   fmt.Sprintf("Consider consolidating %d git dependencies", len(gitDeps)),
			Severity:  "info",
			Rationale: "Many git dependencies can slow down pub operations. Consider using pub.dev packages when possible.",
		})
	}

	// Check for potential monorepo structure
	subdirCount := 0
	for _, dep := range gitDeps {
		if dep.Subdir != "" {
			subdirCount++
		}
	}

	if subdirCount > 3 {
		recommendations = append(recommendations, Reco{
			Message:   "Consider using pub workspaces for monorepo structure",
			Severity:  "info",
			Rationale: "Multiple subdirectory dependencies might benefit from pub workspace configuration",
		})
	}

	return recommendations
}

// GenerateFullRecommendations generates comprehensive recommendations for a project
func GenerateFullRecommendations(logger *Logger, projectPath string) ([]Reco, error) {
	var allRecommendations []Reco

	// Analyze pubspec structure
	structureRecos, err := AnalyzePubspecStructure(projectPath)
	if err != nil {
		logger.Error("reco", err)
	} else {
		allRecommendations = append(allRecommendations, structureRecos...)
	}

	// Analyze git repository
	gitRecos, err := AnalyzeGitRepository(projectPath)
	if err != nil {
		logger.Error("reco", err)
	} else {
		allRecommendations = append(allRecommendations, gitRecos...)
	}

	// Analyze git dependencies
	gitDepRecos, err := AnalyzeGitDependencies(logger, projectPath)
	if err != nil {
		logger.Error("reco", err)
	} else {
		allRecommendations = append(allRecommendations, gitDepRecos...)
	}

	// Analyze performance
	gitDeps, err := ListGitDependencies(projectPath)
	if err != nil {
		logger.Error("reco", err)
	} else {
		perfRecos := AnalyzePerformance(projectPath, gitDeps)
		allRecommendations = append(allRecommendations, perfRecos...)
	}

	// Add popular package suggestions if no specific issues found
	if len(allRecommendations) == 0 {
		allRecommendations = append(allRecommendations, SuggestPopularPkgs()...)
	}

	return allRecommendations, nil
}

// Helper functions
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
