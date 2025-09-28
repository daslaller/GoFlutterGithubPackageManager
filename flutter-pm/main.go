package main

import (
	"fmt"
	"os"

	"github.com/daslaller/GoFlutterGithubPackageManager/flutter-pm/internal/core"
	"github.com/daslaller/GoFlutterGithubPackageManager/flutter-pm/internal/tui"
)

func main() {
	cfg := core.ParseEnv()
	if cfg.ShowVersion {
		fmt.Println("flutter-pm pre-alpha")
		return
	}

	// Dependency checks (git, dart|flutter, gh optional)
	if err := core.EnsureCoreTools(cfg); err != nil {
		core.LogError(cfg, "deps", err)
		os.Exit(1)
	}

	if cfg.CLICommand != "" {
		// rudimentary CLI handling (stubs except sync)
		switch cfg.CLICommand {
		case "sync":
			if err := core.Sync(cmdRoot(cfg)); err != nil {
				core.LogError(cfg, "sync", err)
				os.Exit(1)
			}
			return
		case "add", "status", "reco":
			fmt.Println("Command not implemented yet in pre-alpha; launch TUI by running just 'flutter-pm'.")
			return
		default:
			fmt.Printf("Unknown command: %s\n", cfg.CLICommand)
			os.Exit(2)
		}
	}

	// Default: launch TUI
	if err := tui.Run(cfg); err != nil {
		core.LogError(cfg, "tui", err)
		os.Exit(1)
	}
}

func cmdRoot(cfg core.Config) string {
	if cfg.RootDir != "" {
		return cfg.RootDir
	}
	cwd, _ := os.Getwd()
	return cwd
}
