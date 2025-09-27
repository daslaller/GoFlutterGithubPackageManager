package sbpm

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the desired state loaded from JSON files used by shell scripts.
// This is a starting point and will evolve to mirror scripts/windows/private-packages.json etc.
// Fields intentionally minimal for now.

type Config struct {
	Packages []Package `json:"packages"`
}

type Package struct {
	Name   string `json:"name"`
	Source string `json:"source,omitempty"` // e.g., system, brew, choco, winget, apt, url
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
