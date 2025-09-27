package sbpm

import (
	"context"
	"testing"
	"time"
)

func TestPlanner_Plan_EmptyConfig(t *testing.T) {
	p := NewPlanner(&FakeExecutor{}, Platform{OS: "windows", Arch: "amd64"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	plan, err := p.Plan(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Actions) != 0 {
		t.Fatalf("expected 0 actions, got %d", len(plan.Actions))
	}
}

func TestPlanner_Plan_WithPackages(t *testing.T) {
	p := NewPlanner(&FakeExecutor{}, Platform{OS: "linux", Arch: "amd64"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cfg := &Config{Packages: []Package{{Name: "flutter"}, {Name: "dart"}}}
	plan, err := p.Plan(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(plan.Actions))
	}
	if plan.Actions[0].Package.Name != "flutter" || plan.Actions[1].Package.Name != "dart" {
		t.Fatalf("unexpected plan order: %+v", plan.Actions)
	}
}
