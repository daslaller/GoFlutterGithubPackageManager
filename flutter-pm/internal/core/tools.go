package core

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Tools struct {
	Git     string
	Dart    string
	Flutter string
	GH      string
}

func DetectTools() Tools {
	var t Tools
	t.Git, _ = exec.LookPath("git")
	t.Dart, _ = exec.LookPath("dart")
	t.Flutter, _ = exec.LookPath("flutter")
	t.GH, _ = exec.LookPath("gh")
	return t
}

func EnsureCoreTools(cfg Config) error {
	t := DetectTools()
	// Git is mandatory
	if t.Git == "" {
		return errors.New("git is required but not found on PATH")
	}
	// Dart/Flutter are optional at startup for parity with shell scripts.
	// We will check for them right before pub operations (add/get) and provide actionable hints then.
	if t.Dart == "" && t.Flutter == "" {
		LogInfo(cfg, "deps", "Neither 'dart' nor 'flutter' found on PATH. Pub-related features will prompt with install hints when used.")
	}
	// gh is optional; attempt install if missing
	if t.GH == "" {
		LogInfo(cfg, "deps", "GitHub CLI (gh) not found. Some GitHub features will be limited.")
		if err := ensureGHInstalledInteractive(cfg); err != nil {
			LogInfo(cfg, "deps", fmt.Sprintf("Skipping gh install: %v", err))
		}
	}
	return nil
}

func ensureGHInstalledInteractive(cfg Config) error {
	if !isInteractive() {
		return errors.New("non-interactive environment; not attempting install")
	}
	fmt.Print("Install GitHub CLI (gh) now? [Y/n]: ")
	ans := readLine()
	if ans != "" && strings.ToLower(string(ans[0])) == "n" {
		return errors.New("user declined gh install")
	}
	return installGH()
}

func isInteractive() bool {
	return isTTY(os.Stdin.Fd()) && isTTY(os.Stdout.Fd())
}

// isTTY is a conservative placeholder to avoid extra deps; returns false on unknown
func isTTY(fd uintptr) bool { return true }

func installGH() error {
	switch runtime.GOOS {
	case "windows":
		// Prefer winget, fallback to choco
		if has("winget") {
			return exec.Command("winget", "install", "--id=GitHub.cli", "-e", "--silent").Run()
		}
		if has("choco") {
			return exec.Command("choco", "install", "gh", "-y").Run()
		}
		return errors.New("winget/choco not available; install gh manually: https://cli.github.com/")
	case "darwin":
		if has("brew") {
			return exec.Command("brew", "install", "gh").Run()
		}
		return errors.New("homebrew not found; install gh: brew install gh or see https://cli.github.com/")
	default: // linux
		// Try apt, then dnf, then yum
		if has("apt") || has("apt-get") {
			// Attempt the documented install
			cmd := exec.Command("bash", "-lc", "type -p curl >/dev/null || sudo apt install curl -y; curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && sudo chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg && echo \"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main\" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null && sudo apt update && sudo apt install gh -y")
			return cmd.Run()
		}
		if has("dnf") {
			return exec.Command("sudo", "dnf", "install", "gh", "-y").Run()
		}
		if has("yum") {
			return exec.Command("sudo", "yum", "install", "gh", "-y").Run()
		}
		return errors.New("no known package manager found; install gh from https://cli.github.com/")
	}
}

func has(name string) bool { _, err := exec.LookPath(name); return err == nil }

func readLine() string {
	r := bufio.NewReader(os.Stdin)
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}
