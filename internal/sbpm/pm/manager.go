package pm

import (
	"context"
)

type Package struct {
	Name   string
	Source string
}

type Manager interface {
	Name() string
	Present(ctx context.Context, pkg Package) (bool, error)
	Install(ctx context.Context, pkg Package) error
	UpdateIndex(ctx context.Context) error
}

// NoopManager is a stub that reports nothing present and performs no real installs.
// Useful while wiring the planner.

type NoopManager struct{}

func (n NoopManager) Name() string { return "noop" }

func (n NoopManager) Present(ctx context.Context, pkg Package) (bool, error) { return false, nil }

func (n NoopManager) Install(ctx context.Context, pkg Package) error { return nil }

func (n NoopManager) UpdateIndex(ctx context.Context) error { return nil }
