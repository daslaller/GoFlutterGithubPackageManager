# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains a Go-based replacement for the shell-based Flutter Package Manager, designed to have full functional parity with the original shell scripts while providing a modern Terminal User Interface (TUI) using bubbletea.

## Architecture

The project follows a clean architecture pattern with clear separation of concerns:

### Core Modules (`internal/core/`)
- **env.go**: Configuration, logging, and environment variable parsing
- **types.go**: Data structures and type definitions
- **discovery.go**: Project and repository discovery logic (matches shell script behavior)
- **pub.go**: Dart/Flutter pub command integration and pubspec.yaml management
- **git.go**: Git operations and GitHub CLI integration  
- **stale.go**: Stale dependency detection and express update functionality
- **reco.go**: Smart recommendations system

### TUI Interface (`internal/tui/`)
- **model.go**: Main TUI application state and update logic using bubbletea
- **view.go**: Step-based UI rendering with Flutter-inspired styling
- Implements 6-step workflow: Detect Project → Choose Source → List Repos → Edit Specs → Confirm → Execute → Summary

### Commands (`cmd/`)
- **root.go**: CLI command routing and execution
- Supports both interactive TUI (default) and non-interactive CLI commands

## Development Commands

### Build and Test
```bash
# Build the application
go build -o flutter-pm.exe .

# Run from source  
go run . --help

# Run tests
go test ./...

# Run with debug logging
FLUTTER_PM_DEBUG=1 go run .

# Show version
go run . --version
```

### CLI Commands (Non-Interactive)
```bash
# Sync dependencies in current project
./flutter-pm sync

# Show project status  
./flutter-pm status

# Generate recommendations
./flutter-pm reco

# Show version
./flutter-pm --version
```

### TUI Mode (Interactive - Default)
```bash
# Launch interactive TUI
./flutter-pm

# Launch with specific project directory
./flutter-pm --root /path/to/project

# Dry run mode (shows what would be executed)
./flutter-pm --dry-run
```

## Shell Script Parity

This Go implementation maintains full functional parity with the shell scripts:

### Matched Features
- **Project Discovery**: Nearest pubspec.yaml detection, common directory scanning
- **GitHub Integration**: Uses `gh` CLI for authentication and repository listing  
- **Multi-select Interface**: Space-bar selection for multiple repositories
- **Express Git Update**: Quick update of stale git dependencies
- **Backup Strategy**: Automatic pubspec.yaml.backup creation
- **Stale Detection**: Both heuristic (24h) and precise (SHA comparison) methods
- **Recommendations**: Smart suggestions for pinning refs, SSH URLs, etc.

### Key Workflow
1. **Detect Project**: Find Flutter projects (current dir → scan common roots)
2. **Choose Source**: GitHub repos, manual URL, or local scan
3. **List Repositories**: Browse and multi-select with space bar
4. **Configure Packages**: Set package names, branches/tags, subdirectories
5. **Confirm & Execute**: Review changes, create backup, add dependencies
6. **Summary**: Show results and recommendations

## Dependencies and Prerequisites

### Runtime Dependencies
- **Go 1.25+**: For the application itself
- **git**: Required for all Git operations
- **dart** or **flutter**: Required for pub operations
- **gh** (GitHub CLI): Optional but recommended for GitHub integration

### Go Dependencies
- `github.com/charmbracelet/bubbletea`: TUI framework
- `github.com/charmbracelet/lipgloss`: TUI styling
- All dependencies are managed via go.mod

## Configuration

### Environment Variables
- `FLUTTER_PM_DEBUG=1`: Enable debug logging
- `FLUTTER_PM_DRY_RUN=1`: Enable dry-run mode
- `FLUTTER_PM_QUIET=1`: Quiet mode (errors only)
- `FLUTTER_PM_JSON=1`: JSON output format
- `FLUTTER_PM_ROOT=path`: Default project root directory

### Command Line Flags
- `--version`: Show version information
- `--dry-run`: Show what would be executed without doing it
- `--quiet`: Minimize output
- `--debug`: Enable debug logging  
- `--json`: Output structured JSON logs
- `--root PATH`: Specify project directory

## Testing Strategy

### Unit Tests
```bash
# Run all tests
go test ./...

# Test specific modules
go test ./internal/core
go test ./internal/tui

# Run with coverage
go test -cover ./...
```

### Integration Testing
The application includes integration with the shell scripts:
- Uses same Git operations
- Same GitHub CLI integration
- Same pubspec.yaml manipulation patterns
- Same backup and safety mechanisms

### Parity Validation
To verify parity with shell scripts:
1. Run shell script on test project and capture pubspec.yaml changes
2. Run Go binary on same test project  
3. Compare final pubspec.yaml files - should be identical
4. Compare command sequences in logs (with --explain flag)

## Source of Truth

The shell script implementation (`ShellBasedPackageManager/scripts/linux-macos/linux_macos_full.sh`) remains the source of truth for functional behavior. Any discrepancies should be resolved by aligning the Go implementation with the shell script behavior.

## Key Design Principles

1. **Shell Script Parity**: Maintain exact functional equivalence
2. **Git CLI Fidelity**: Always use git CLI for operations (not Go libraries)
3. **Pub Command Integration**: Use `dart pub add` / `flutter pub add` (not YAML surgery)
4. **Safety First**: Always create backups before modifying files
5. **Cross-Platform**: Works on Windows, macOS, and Linux
6. **No CGO**: Static builds for easy distribution