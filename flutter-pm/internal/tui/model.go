package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daslaller/GoFlutterGithubPackageManager/flutter-pm/internal/core"
)

// Run temporarily provides a simple interactive text menu (non-TUI) so users
// always see choices even if no pubspec.yaml is found. This aligns better with
// the shell script which shows a menu outside a project too. When a project is
// detected, we also offer a quick "Sync deps" action.
func Run(cfg core.Config) error {
	cwd, _ := os.Getwd()
	proj, err := core.NearestPubspec(cwd)
	inProject := err == nil

	if cfg.Quiet {
		// In quiet mode, avoid interactive prompts. If in a project, just sync; else noop.
		if inProject {
			return core.Sync(proj.Path)
		}
		return nil
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		clearScreenHint()
		if inProject {
			fmt.Printf("flutter-pm — Project detected at: %s\n", proj.Path)
			fmt.Println("Choose an action:")
			fmt.Println("  1) Sync deps (pub get)")
			fmt.Println("  2) Validate & Fix project")
			fmt.Println("  3) Express Git Update")
			fmt.Println("  4) Recommendations (stub)")
			fmt.Println("  5) Add dependencies (stub)")
			fmt.Println("  q) Quit")
			fmt.Print("> ")
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(strings.ToLower(choice))
			switch choice {
			case "1":
				fmt.Println("Running pub get...")
				if err := core.Sync(proj.Path); err != nil {
					core.LogError(cfg, "sync", err)
				} else {
					fmt.Println("Done.")
				}
				pause(reader)
			case "2":
				fmt.Println("Validating project and applying safe fixes...")
				msgs, err := core.ValidateProject(proj.Path, true)
				for _, m := range msgs {
					fmt.Println("-", m)
				}
				if err != nil {
					core.LogError(cfg, "validate", err)
				}
				pause(reader)
			case "3":
				fmt.Println("Running Express Git Update...")
				if stale, lock, _ := core.LockIsStale(proj.Path); stale {
					fmt.Printf("Note: %s looks older than 24h; deps may be stale.\\n", lock)
				}
				if err := core.ExpressGitUpdate(proj.Path); err != nil {
					core.LogError(cfg, "express_update", err)
				} else {
					fmt.Println("Update complete.")
				}
				pause(reader)
			case "4":
				recos := core.SuggestPopularPkgs()
				fmt.Println("Recommendations:")
				for _, r := range recos {
					fmt.Printf("- %s (%s) — %s\n", r.Message, r.Severity, r.Rationale)
				}
				pause(reader)
			case "5":
				fmt.Println("Add dependencies flow not implemented yet in pre-alpha.")
				pause(reader)
			case "q", "quit", "exit":
				return nil
			default:
				// ignore and re-loop
			}
		} else {
			fmt.Println("flutter-pm — No pubspec.yaml found in current or parent directories.")
			if cfg.RootDir != "" {
				fmt.Printf("Hint: The provided --root path '%s' does not contain a Flutter/Dart project.\n", cfg.RootDir)
			}
			fmt.Println("You can still use these options:")
			fmt.Println("  1) Discover local project locations")
			fmt.Println("  2) Getting started help")
			fmt.Println("  q) Quit")
			fmt.Print("> ")
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(strings.ToLower(choice))
			switch choice {
			case "1":
				roots := core.CommonRoots()
				existing := make([]string, 0, len(roots))
				for _, r := range roots {
					if fi, err := os.Stat(r); err == nil && fi.IsDir() {
						existing = append(existing, r)
					}
				}
				if len(existing) == 0 {
					fmt.Println("No common development folders detected. Try checking your usual project directories.")
				} else {
					fmt.Println("Common places to check for projects on this machine:")
					for _, r := range existing {
						fmt.Printf("  - %s\n", r)
					}
					fmt.Printf("Example: cd %s && flutter-pm\n", filepath.Join(existing[0]))
				}
				pause(reader)
			case "2":
				fmt.Println("How to start:")
				fmt.Println("- Navigate to a Flutter/Dart project (directory with pubspec.yaml) and run flutter-pm again,")
				fmt.Println("  or pass --root PATH to point at a project directory.")
				fmt.Println("- Or choose 'Discover' to see common locations where your projects might live.")
				pause(reader)
			case "q", "quit", "exit":
				return nil
			default:
				// ignore and re-loop
			}
		}
	}
}

func pause(r *bufio.Reader) {
	fmt.Print("Press Enter to continue...")
	_, _ = r.ReadString('\n')
}

// clearScreenHint avoids complex terminal control; it just inserts spacing to separate screens.
func clearScreenHint() {
	fmt.Println()
}
