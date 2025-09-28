package core

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Sync runs pub get in the project root (nearest pubspec)
func Sync(start string) error {
	proj, err := NearestPubspec(start)
	if err != nil {
		return err
	}
	tool := findFirstOnPath("dart", "flutter")
	if tool == "" {
		return errors.New("neither 'dart' nor 'flutter' found on PATH; install Dart SDK or Flutter. Hints: https://dart.dev/get-dart or https://docs.flutter.dev/get-started/install")
	}
	args := []string{"pub", "get"}
	if filepath.Base(tool) == "flutter" || filepath.Base(tool) == "flutter.exe" {
		// for flutter, args are the same: flutter pub get
	}
	cmd := exec.Command(tool, args...)
	cmd.Dir = proj.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Running %s %v in %s\n", tool, args, proj.Path)
	return cmd.Run()
}

func findFirstOnPath(names ...string) string {
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	return ""
}
