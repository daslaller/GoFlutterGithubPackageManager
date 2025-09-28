package core

import (
	"errors"
	"os"
	"path/filepath"
)

// NearestPubspec walks up from startDir to root to find pubspec.yaml
func NearestPubspec(startDir string) (Project, error) {
	d := startDir
	for {
		p := filepath.Join(d, "pubspec.yaml")
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return Project{Path: d, PubspecPath: p}, nil
		}
		parent := filepath.Dir(d)
		if parent == d { // reached filesystem root
			break
		}
		d = parent
	}
	return Project{}, errors.New("no pubspec.yaml found in parent directories")
}

// CommonRoots returns common development roots (platform-aware)
func CommonRoots() []string {
	home, _ := os.UserHomeDir()
	var roots []string
	for _, sub := range []string{"Development", "Projects", "dev"} {
		roots = append(roots, filepath.Join(home, sub))
	}
	return roots
}
