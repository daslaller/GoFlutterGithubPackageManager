package cmd

import (
	"fmt"
	"os"

	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/tui"
)

// Execute runs the root command
func Execute() error {
	cfg := core.ParseEnv()
	logger := core.NewLogger(&cfg)

	// Handle version flag
	if cfg.ShowVersion {
		fmt.Println("flutter-pm v1.0.0-alpha")
		fmt.Println("AI-Powered Flutter Package Manager")
		return nil
	}

	// Handle CLI commands
	if cfg.CLICommand != "" {
		return handleCLICommand(cfg, logger)
	}

	// Default: launch TUI
	return tui.Run(cfg, logger)
}

// handleCLICommand handles non-interactive CLI commands
func handleCLICommand(cfg core.Config, logger *core.Logger) error {
	rootDir := cfg.RootDir
	if rootDir == "" {
		var err error
		rootDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	switch cfg.CLICommand {
	case "sync":
		return cmdSync(cfg, logger, rootDir)
	case "add":
		return cmdAdd(cfg, logger, rootDir)
	case "status":
		return cmdStatus(cfg, logger, rootDir)
	case "reco":
		return cmdReco(cfg, logger, rootDir)
	case "autotest":
		return cmdAutoTest(cfg, logger)
	default:
		return fmt.Errorf("unknown command: %s", cfg.CLICommand)
	}
}

// cmdSync handles the sync command
func cmdSync(cfg core.Config, logger *core.Logger, rootDir string) error {
	// Find project
	project, err := core.NearestPubspec(rootDir)
	if err != nil {
		return fmt.Errorf("no Flutter project found: %w", err)
	}

	logger.Info("sync", fmt.Sprintf("Syncing dependencies for %s", project.Path))

	result := core.Sync(logger, &cfg, project.Path)
	if !result.OK {
		return fmt.Errorf("sync failed: %s", result.Err)
	}

	logger.Info("sync", result.Message)
	return nil
}

// cmdAdd handles the add command (stub - would need additional CLI parsing)
func cmdAdd(cfg core.Config, logger *core.Logger, rootDir string) error {
	return fmt.Errorf("add command not implemented in CLI mode yet - use TUI")
}

// cmdStatus handles the status command
func cmdStatus(cfg core.Config, logger *core.Logger, rootDir string) error {
	// Find project
	project, err := core.NearestPubspec(rootDir)
	if err != nil {
		return fmt.Errorf("no Flutter project found: %w", err)
	}

	logger.Info("status", fmt.Sprintf("Checking status for %s", project.Path))

	// Check for git dependencies
	gitDeps, err := core.ListGitDependencies(project.Path)
	if err != nil {
		return fmt.Errorf("failed to list git dependencies: %w", err)
	}

	logger.Info("status", fmt.Sprintf("Found %d git dependencies", len(gitDeps)))

	// Check stale status
	staleInfo, err := core.CheckStalePrecise(logger, project.Path)
	if err != nil {
		logger.Error("status", err)
		// Fall back to heuristic
		isStale, lockPath, _ := core.CheckStaleHeuristic(project.Path)
		if isStale {
			logger.Info("status", fmt.Sprintf("Lock file %s appears stale (>24h old)", lockPath))
		}
	} else {
		staleCount := 0
		for _, info := range staleInfo {
			if info.IsStale {
				staleCount++
				logger.Info("status", fmt.Sprintf("%s is stale: %s -> %s",
					info.PackageName, info.CurrentSHA, info.UpstreamSHA))
			}
		}
		if staleCount == 0 {
			logger.Info("status", "All git dependencies are up to date")
		}
	}

	return nil
}

// cmdReco handles the recommendations command
func cmdReco(cfg core.Config, logger *core.Logger, rootDir string) error {
	// Find project
	project, err := core.NearestPubspec(rootDir)
	if err != nil {
		return fmt.Errorf("no Flutter project found: %w", err)
	}

	logger.Info("reco", fmt.Sprintf("Generating recommendations for %s", project.Path))

	recommendations, err := core.GenerateFullRecommendations(logger, project.Path)
	if err != nil {
		return fmt.Errorf("failed to generate recommendations: %w", err)
	}

	if len(recommendations) == 0 {
		logger.Info("reco", "No recommendations - project looks good!")
		return nil
	}

	for _, reco := range recommendations {
		severity := reco.Severity
		if severity == "warn" {
			severity = "⚠️"
		} else if severity == "error" {
			severity = "❌"
		} else {
			severity = "ℹ️"
		}

		fmt.Printf("%s %s\n", severity, reco.Message)
		if reco.Rationale != "" {
			fmt.Printf("   %s\n", reco.Rationale)
		}
		fmt.Println()
	}

	return nil
}

// cmdAutoTest handles the autotest command
func cmdAutoTest(cfg core.Config, logger *core.Logger) error {
	return tui.RunParityAutoTest(cfg, logger)
}
