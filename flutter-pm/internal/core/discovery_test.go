package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNearestPubspec_WalksUp(t *testing.T) {
	dir := t.TempDir()
	// create nested directories
	a := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(a, 0o755); err != nil {
		t.Fatal(err)
	}
	// place pubspec at dir/a
	pub := filepath.Join(dir, "a", "pubspec.yaml")
	if err := os.WriteFile(pub, []byte("name: demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proj, err := NearestPubspec(a)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if proj.Path != filepath.Join(dir, "a") {
		t.Fatalf("got %s", proj.Path)
	}
	if proj.PubspecPath != pub {
		t.Fatalf("got %s", proj.PubspecPath)
	}
}
