package core

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds global flags/env
// Default behavior is verbose unless --quiet
// JSON logs are NDJSON lines when enabled.
type Config struct {
	JSON        bool
	DryRun      bool
	Explain     bool
	Quiet       bool
	RootDir     string
	ShowVersion bool
	CLICommand  string
	Args        []string
}

func ParseEnv() Config {
	var cfg Config
	fs := flag.NewFlagSet("flutter-pm", flag.ContinueOnError)
	fs.SetOutput(new(flagErrSink))
	fs.BoolVar(&cfg.JSON, "json", false, "emit NDJSON logs to stdout")
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "print intended operations, do not change anything")
	fs.BoolVar(&cfg.Explain, "explain", false, "print exact external commands that will be executed")
	fs.BoolVar(&cfg.Quiet, "quiet", false, "less chatty output")
	fs.StringVar(&cfg.RootDir, "root", "", "project root (defaults to CWD or nearest pubspec dir)")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print version and exit")

	// crude subcommand capture: if first arg looks like one of our commands, record it
	if len(os.Args) > 1 {
		cmd := os.Args[1]
		switch cmd {
		case "add", "sync", "status", "reco":
			cfg.CLICommand = cmd
			_ = fs.Parse(os.Args[2:])
			cfg.Args = fs.Args()
			return cfg
		}
	}
	_ = fs.Parse(os.Args[1:])
	cfg.Args = fs.Args()
	return cfg
}

// Logging helpers (simple for now)

type flagErrSink struct{}

func (f *flagErrSink) Write(p []byte) (int, error) {
	// ensure flag errors are readable
	fmt.Fprint(os.Stderr, string(p))
	return len(p), nil
}

func now() string { return time.Now().Format(time.RFC3339Nano) }

func LogJSON(cfg Config, level, action, msg string, fields map[string]any) {
	if !cfg.JSON {
		return
	}
	b := &strings.Builder{}
	b.WriteString("{")
	fmt.Fprintf(b, "\"time\":\"%s\",\"level\":\"%s\",\"action\":\"%s\",\"msg\":\"%s\"", now(), esc(level), esc(action), esc(msg))
	for k, v := range fields {
		fmt.Fprintf(b, ",\"%s\":%s", esc(k), toJSON(v))
	}
	b.WriteString("}\n")
	os.Stdout.WriteString(b.String())
}

func LogError(cfg Config, action string, err error) {
	if cfg.JSON {
		LogJSON(cfg, "error", action, err.Error(), nil)
		return
	}
	fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", action, err)
}

func LogInfo(cfg Config, action, msg string) {
	if cfg.JSON {
		LogJSON(cfg, "info", action, msg, nil)
		return
	}
	if !cfg.Quiet {
		fmt.Printf("[INFO] %s: %s\n", action, msg)
	}
}

func esc(s string) string { return strings.ReplaceAll(s, "\"", "\\\"") }

func toJSON(v any) string {
	switch x := v.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", esc(x))
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("\"%v\"", x)
	}
}
