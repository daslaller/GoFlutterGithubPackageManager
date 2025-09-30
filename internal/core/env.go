package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration
type Config struct {
	// Core settings
	RootDir    string
	DryRun     bool
	Quiet      bool
	Debug      bool
	JSONOutput bool
	Jobs       int
	Explain    bool

	// Command line args
	Command     string
	CLICommand  string
	ShowVersion bool
}

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// Logger provides structured logging
type Logger struct {
	cfg     *Config
	level   LogLevel
	entries []LogEntry
}

// LogEntry represents a single log entry
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ParseEnv creates a Config from environment variables and command line args
func ParseEnv() Config {
	cfg := Config{
		RootDir:    os.Getenv("FLUTTER_PM_ROOT"),
		DryRun:     parseBool(os.Getenv("FLUTTER_PM_DRY_RUN")),
		Quiet:      parseBool(os.Getenv("FLUTTER_PM_QUIET")),
		Debug:      parseBool(os.Getenv("FLUTTER_PM_DEBUG")),
		JSONOutput: parseBool(os.Getenv("FLUTTER_PM_JSON")),
		Jobs:       parseInt(os.Getenv("FLUTTER_PM_JOBS"), 4),
		Explain:    parseBool(os.Getenv("FLUTTER_PM_EXPLAIN")),
	}

	// Parse command line arguments
	args := os.Args[1:]
	for i, arg := range args {
		switch arg {
		case "--version", "-v":
			cfg.ShowVersion = true
		case "--dry-run":
			cfg.DryRun = true
		case "--quiet", "-q":
			cfg.Quiet = true
		case "--debug":
			cfg.Debug = true
		case "--json":
			cfg.JSONOutput = true
		case "--explain":
			cfg.Explain = true
		case "--root":
			if i+1 < len(args) {
				cfg.RootDir = args[i+1]
			}
		case "--jobs":
			if i+1 < len(args) {
				cfg.Jobs = parseInt(args[i+1], 4)
			}
		case "add", "sync", "status", "reco":
			cfg.CLICommand = arg
		}
	}

	return cfg
}

// NewLogger creates a new logger instance
func NewLogger(cfg *Config) *Logger {
	level := LogLevelInfo
	if cfg.Debug {
		level = LogLevelDebug
	} else if cfg.Quiet {
		level = LogLevelError
	}

	return &Logger{
		cfg:     cfg,
		level:   level,
		entries: make([]LogEntry, 0),
	}
}

// Log adds a log entry
func (l *Logger) Log(level LogLevel, component, message string, data map[string]interface{}) {
	if level > l.level {
		return
	}

	entry := LogEntry{
		Level:     logLevelString(level),
		Message:   message,
		Component: component,
		Data:      data,
	}

	l.entries = append(l.entries, entry)

	if l.cfg.JSONOutput {
		json.NewEncoder(os.Stderr).Encode(entry)
	} else if !l.cfg.Quiet {
		l.printEntry(entry)
	}
}

// Error logs an error
func (l *Logger) Error(component string, err error) {
	l.Log(LogLevelError, component, err.Error(), nil)
}

// Info logs an info message
func (l *Logger) Info(component, message string) {
	l.Log(LogLevelInfo, component, message, nil)
}

// Debug logs a debug message
func (l *Logger) Debug(component, message string) {
	l.Log(LogLevelDebug, component, message, nil)
}

// LogCommand logs a command execution
func (l *Logger) LogCommand(component, command string, args []string) {
	if l.cfg.Explain {
		fullCmd := command + " " + strings.Join(args, " ")
		l.Info(component, "executing: "+fullCmd)
	}
}

// GetEntries returns all log entries
func (l *Logger) GetEntries() []LogEntry {
	return l.entries
}

func (l *Logger) printEntry(entry LogEntry) {
	prefix := ""
	switch entry.Level {
	case "error":
		prefix = "❌ "
	case "warn":
		prefix = "⚠️ "
	case "info":
		prefix = "ℹ️ "
	case "debug":
		prefix = "🔍 "
	}

	message := entry.Message
	if entry.Component != "" {
		message = fmt.Sprintf("[%s] %s", entry.Component, message)
	}

	fmt.Fprintf(os.Stderr, "%s%s\\n", prefix, message)
}

func parseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return i
}

func logLevelString(level LogLevel) string {
	switch level {
	case LogLevelError:
		return "error"
	case LogLevelWarn:
		return "warn"
	case LogLevelInfo:
		return "info"
	case LogLevelDebug:
		return "debug"
	default:
		return "unknown"
	}
}
