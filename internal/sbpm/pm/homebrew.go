package pm

import (
	"context"
	"errors"
	"fmt"
)

// Executor matches the command runner used by higher layers. It is re-declared
// here to avoid an import cycle between sbpm and pm packages.
// It is satisfied by internal/sbpm.DefaultExecutor and FakeExecutor.
type Executor interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// Homebrew implements Manager using the `brew` CLI on macOS.
// It assumes Homebrew is installed and available on PATH.
// Commands are invoked directly without shell wrapping.

type Homebrew struct {
	exec Executor
}

func NewHomebrew(exec Executor) *Homebrew { return &Homebrew{exec: exec} }

func (h *Homebrew) Name() string { return "homebrew" }

func (h *Homebrew) Present(ctx context.Context, pkg Package) (bool, error) {
	// `brew list --versions <name>` returns 0 if installed and prints versions.
	_, _, err := h.exec.Run(ctx, "brew", "list", "--versions", pkg.Name)
	if err == nil {
		return true, nil
	}
	// Distinguish between not found vs other errors is tricky without stderr text.
	// For skeleton, any error is treated as not present.
	return false, nil
}

func (h *Homebrew) Install(ctx context.Context, pkg Package) error {
	// Prefer formula install. Casks can be supported later via Source or extra field.
	_, stderr, err := h.exec.Run(ctx, "brew", "install", pkg.Name)
	if err != nil {
		return fmt.Errorf("brew install %s failed: %v: %s", pkg.Name, err, string(stderr))
	}
	return nil
}

func (h *Homebrew) UpdateIndex(ctx context.Context) error {
	// Keep Homebrew up to date before installs.
	_, stderr, err := h.exec.Run(ctx, "brew", "update")
	if err != nil {
		// Allow proceeding if update fails due to network, but surface error.
		return errors.New("brew update failed: " + string(stderr))
	}
	return nil
}
