package sbpm

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestPlanner_Execute_MacOS_UsesBrew(t *testing.T) {
	fe := &FakeExecutor{Responses: map[string]ExecResponse{
		"brew\x00list\x00--versions\x00git":  {Err: fmt.Errorf("not installed")},
		"brew\x00list\x00--versions\x00wget": {Err: fmt.Errorf("not installed")},
	}}
	p := NewPlanner(fe, Platform{OS: "macos", Arch: "amd64"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := &Config{Packages: []Package{{Name: "git"}, {Name: "wget"}}}
	plan, err := p.Plan(ctx, cfg)
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	res := p.Execute(ctx, plan)
	if res.Err != nil {
		t.Fatalf("execute error: %v", res.Err)
	}

	// Expected sequence: brew update, list+install for each pkg
	want := []ExecCall{
		{Name: "brew", Args: []string{"update"}},
		{Name: "brew", Args: []string{"list", "--versions", "git"}},
		{Name: "brew", Args: []string{"install", "git"}},
		{Name: "brew", Args: []string{"list", "--versions", "wget"}},
		{Name: "brew", Args: []string{"install", "wget"}},
	}
	if len(fe.Calls) != len(want) {
		t.Fatalf("unexpected calls len: got %d want %d: %+v", len(fe.Calls), len(want), fe.Calls)
	}
	for i := range want {
		if fe.Calls[i].Name != want[i].Name {
			t.Fatalf("call %d name: got %s want %s", i, fe.Calls[i].Name, want[i].Name)
		}
		if len(fe.Calls[i].Args) != len(want[i].Args) {
			t.Fatalf("call %d args len: got %v want %v", i, fe.Calls[i].Args, want[i].Args)
		}
		for j := range want[i].Args {
			if fe.Calls[i].Args[j] != want[i].Args[j] {
				t.Fatalf("call %d arg %d: got %q want %q", i, j, fe.Calls[i].Args[j], want[i].Args[j])
			}
		}
	}
}
