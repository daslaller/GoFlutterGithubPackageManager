package core

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type GitDep struct {
	Name string
	URL  string
	Ref  string
	SHA  string // resolved in lock
}

// LockIsStale returns true if pubspec.lock is older than 24h.
func LockIsStale(root string) (bool, string, error) {
	proj, err := NearestPubspec(root)
	if err != nil {
		return false, "", err
	}
	lock := filepath.Join(proj.Path, "pubspec.lock")
	fi, err := os.Stat(lock)
	if err != nil {
		return false, lock, nil
	}
	age := time.Since(fi.ModTime())
	return age > 24*time.Hour, lock, nil
}

// ParseGitDepsFromLock parses a minimal subset of pubspec.lock to find git deps with revision.
func ParseGitDepsFromLock(root string) ([]GitDep, error) {
	proj, err := NearestPubspec(root)
	if err != nil {
		return nil, err
	}
	lock := filepath.Join(proj.Path, "pubspec.lock")
	f, err := os.Open(lock)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var deps []GitDep
	var cur *GitDep
	inGit := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, " ") == false { // reset on top-level
			cur = nil
			inGit = false
		}
		// Match dep name section: "  somepkg:"
		if strings.HasPrefix(line, "  ") && strings.HasSuffix(line, ":") && !strings.Contains(line, " ") { // simplistic
			name := strings.TrimSuffix(strings.TrimSpace(line), ":")
			cur = &GitDep{Name: name}
			continue
		}
		if cur == nil {
			continue
		}
		if strings.Contains(line, "source:") && strings.Contains(line, "git") {
			inGit = true
		}
		if inGit {
			if strings.Contains(line, "url:") {
				cur.URL = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "url:"))
			}
			if strings.Contains(line, "ref:") {
				cur.Ref = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "ref:"))
			}
			if strings.Contains(line, "revision:") {
				cur.SHA = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "revision:"))
				deps = append(deps, *cur)
				cur = nil
				inGit = false
			}
		}
	}
	return deps, nil
}

// ExpressGitUpdate performs a quick update of git-based dependencies.
// It backs up pubspec.yaml, checks upstream shas, and runs `pub upgrade` if any mismatch.
func ExpressGitUpdate(root string) error {
	proj, err := NearestPubspec(root)
	if err != nil {
		return err
	}
	if _, err := BackupPubspec(proj.Path); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	deps, _ := ParseGitDepsFromLock(proj.Path) // tolerate parse failure; we'll still try a general upgrade
	needsUpgrade := false
	for _, d := range deps {
		if d.URL == "" || d.Ref == "" || d.SHA == "" {
			continue
		}
		up, err := GitLsRemote(d.URL, d.Ref)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(up, d.SHA) {
			needsUpgrade = true
			break
		}
	}
	tool := findFirstOnPath("dart", "flutter")
	if tool == "" {
		return errors.New("neither 'dart' nor 'flutter' found on PATH; install Dart or Flutter")
	}
	if needsUpgrade {
		// run pub upgrade
		if err := runCmd(tool, proj.Path, []string{"pub", "upgrade"}); err != nil {
			return err
		}
	}
	// Always run pub get at the end to sync
	return runCmd(tool, proj.Path, []string{"pub", "get"})
}

func runCmd(tool, dir string, args []string) error {
	cmd := exec.Command(tool, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Running %s %v in %s\n", tool, args, dir)
	return cmd.Run()
}
