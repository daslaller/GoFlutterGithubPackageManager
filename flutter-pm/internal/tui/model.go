package tui

import (
	"fmt"
	"os"

	"github.com/daslaller/GoFlutterGithubPackageManager/flutter-pm/internal/core"
)

// Run is a temporary placeholder until Bubble Tea wiring is added.
// It detects the project and runs pub get, emitting verbose logs unless --quiet.
func Run(cfg core.Config) error {
	cwd, _ := os.Getwd()
	proj, err := core.NearestPubspec(cwd)
	if err != nil {
		return err
	}
	if !cfg.Quiet {
		fmt.Printf("Detected Flutter/Dart project at %s\n", proj.Path)
		fmt.Println("Synchronizing dependencies (pub get)...")
	}
	return core.Sync(proj.Path)
}
