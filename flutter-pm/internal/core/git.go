package core

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

func GitLsRemote(url, ref string) (string, error) {
	args := []string{"ls-remote", url, ref}
	out, err := runCmdCapture("git", args...)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 && strings.HasSuffix(parts[1], ref) {
			return parts[0], nil
		}
	}
	return "", errors.New("ref not found in ls-remote output")
}

func runCmdCapture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", errors.New(strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
