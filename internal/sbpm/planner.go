package sbpm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"awesomeProject/internal/sbpm/pm"
)

// ActionType describes what to do.
//
type ActionType string

const (
	ActionInstall  ActionType = "install"
	ActionConfigure ActionType = "configure"
)

// Action represents a single step in a plan.
//
type Action struct {
	Type    ActionType
	Package Package
	Detail  string
}

// Plan is an ordered list of actions.
//
type Plan struct {
	Actions []Action
}

// pmExecAdapter bridges internal Executor to pm.Executor.
// Defined at package level to avoid function-local method definition.
//
 type pmExecAdapter struct{ inner Executor }

func (a pmExecAdapter) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	return a.inner.Run(ctx, name, args...)
}

func (p Plan) String() string {
	if len(p.Actions) == 0 {
		return "no actions"
	}
	var b strings.Builder
	for i, a := range p.Actions {
		fmt.Fprintf(&b, "%d. %s %s", i+1, a.Type, a.Package.Name)
		if a.Detail != "" {
			b.WriteString(" (" + a.Detail + ")")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Result summarizes execution.
//
type Result struct {
	Err error
}

// Planner creates a plan based on current system state and config.
// For now, it's a skeleton: it will plan to install every package in Config.
//
type Planner struct {
	exec     Executor
	plat     Platform
	logger   *slog.Logger
}

func NewPlanner(exec Executor, plat Platform, logger *slog.Logger) *Planner {
	if logger == nil {
		logger = slog.Default()
	}
	return &Planner{exec: exec, plat: plat, logger: logger}
}

func (p *Planner) Plan(ctx context.Context, cfg *Config) (Plan, error) {
	var plan Plan
	if cfg == nil {
		// No config means no actions for now; future: infer from platform defaults
		return plan, nil
	}
	for _, pkg := range cfg.Packages {
		plan.Actions = append(plan.Actions, Action{Type: ActionInstall, Package: pkg})
	}
	return plan, nil
}

func (p *Planner) Execute(ctx context.Context, plan Plan) Result {
	// Select a package manager based on platform.
	var m pm.Manager
	switch p.plat.OS {
	case "macos":
		m = pm.NewHomebrew(pmExecAdapter{inner: p.exec})
	default:
		m = pm.NoopManager{}
	}

	// Try to refresh package index; log error but proceed.
	if err := m.UpdateIndex(ctx); err != nil {
		p.logger.Warn("package index update failed", "manager", m.Name(), "err", err)
	}

	for _, a := range plan.Actions {
		p.logger.Info("execute", "type", a.Type, "pkg", a.Package.Name)
		switch a.Type {
		case ActionInstall:
			pkg := pm.Package{Name: a.Package.Name, Source: a.Package.Source}
			present, err := m.Present(ctx, pkg)
			if err != nil {
				return Result{Err: fmt.Errorf("present check failed for %s via %s: %w", pkg.Name, m.Name(), err)}
			}
			if present {
				continue
			}
			if err := m.Install(ctx, pkg); err != nil {
				return Result{Err: fmt.Errorf("install failed for %s via %s: %w", pkg.Name, m.Name(), err)}
			}
		case ActionConfigure:
			// Not implemented yet
		}
	}
	return Result{}
}
