package core

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ValidateProject checks for common issues and optionally fixes them.
// Returns a list of messages describing checks and fixes performed.
func ValidateProject(root string, autoFix bool) ([]string, error) {
	proj, err := NearestPubspec(root)
	if err != nil {
		return nil, err
	}
	msgs := []string{}
	// 1) Ensure lib/main.dart exists
	libDir := filepath.Join(proj.Path, "lib")
	mainDart := filepath.Join(libDir, "main.dart")
	if _, err := os.Stat(mainDart); err != nil {
		msgs = append(msgs, "Missing lib/main.dart")
		if autoFix {
			if err := os.MkdirAll(libDir, 0o755); err != nil {
				return msgs, fmt.Errorf("creating lib dir: %w", err)
			}
			tpl := "void main(){}\n"
			if err := os.WriteFile(mainDart, []byte(tpl), 0o644); err != nil {
				return msgs, fmt.Errorf("creating main.dart: %w", err)
			}
			msgs = append(msgs, "Created lib/main.dart")
		}
	} else {
		msgs = append(msgs, "Found lib/main.dart")
	}
	// 2) Ensure project is a git repo
	gitDir := filepath.Join(proj.Path, ".git")
	if fi, err := os.Stat(gitDir); err != nil || !fi.IsDir() {
		msgs = append(msgs, "Not a git repository")
		if autoFix {
			if err := gitInit(proj.Path); err != nil {
				return msgs, fmt.Errorf("git init failed: %w", err)
			}
			msgs = append(msgs, "Initialized empty Git repository")
		}
	} else {
		msgs = append(msgs, "Git repository detected")
	}
	// 3) Basic pubspec readability
	if _, err := os.ReadFile(proj.PubspecPath); err != nil {
		msgs = append(msgs, "pubspec.yaml not readable")
		return msgs, err
	}
	msgs = append(msgs, "pubspec.yaml readable")
	return msgs, nil
}

func gitInit(dir string) error {
	git, err := exec.LookPath("git")
	if err != nil {
		return errors.New("git not found")
	}
	cmd := exec.Command(git, "init")
	cmd.Dir = dir
	return cmd.Run()
}
