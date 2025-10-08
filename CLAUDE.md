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

### TUI Interface (`internal/tui/models/`)
- **app_model.go**: Main application coordinator that manages screen transitions and shared state
- **One model per screen architecture**:
  - **main_menu_model.go**: Main menu with 5 options (prerequisites, GitHub repo, configure search, update local package, self-update)
  - **source_selection_model.go**: Select Flutter source project to work on
  - **source_config_model.go**: Configure download location and project naming (with file picker)
  - **source_download_model.go**: Download/clone selected source project
  - **search_config_model.go**: Configure repository search filters (owner, language, topic)
  - **github_package_repo_multiselection_model.go**: Multi-select packages to add as dependencies
  - **configuration_model.go**: Configure selected packages (branches, names, etc.)
  - **confirmation_model.go**: Review changes before execution
  - **execution_model.go**: Execute the pub add commands
  - **results_model.go**: Show results and recommendations
- Uses bubbletea framework with lipgloss styling
- Implements Flutter-inspired UI design with bordered headers and consistent color scheme

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

#### Main Menu Flow
1. **Main Menu**: Choose operation (prerequisites, GitHub repo, configure search, update local package, self-update)

#### GitHub Repo Flow (Option 2)
2. **GitHub Loading**: Fetch available packages from GitHub using `gh` CLI
3. **Package Multiselect**: Multi-select packages to add as dependencies (uses list-simple with > markers and checkmarks)
4. **Package Configuration**: Configure selected packages (branches, names, subdirectories)
5. **Confirmation**: Review all changes before execution
6. **Execution**: Execute pub add commands with backup creation
7. **Results**: Show results, errors, and smart recommendations

#### Configure Search Flow (Option 3)
2. **Search Configuration**: Configure directory search paths, depth, and full disk search toggle
3. **Return to Main Menu**: Save settings and return

#### Update Local Package Flow (Option 4)
2. **Scan Directories**: Scan for Flutter projects in configured directories
3. **Project Selection**: Select which project to update
4. **Update Execution**: Run updates on stale dependencies
5. **Results**: Show results and recommendations

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

### TUI Testing
```bash
# Run comprehensive TUI tests
go test -v ./internal/tui/testing

# Test specific functionality
go test -v ./internal/tui/testing -run TestCompleteMenuWalkthrough
go test -v ./internal/tui/testing -run TestGitHubRepoToMultiselect

# Generate terminal output frames for analysis
go test -v ./internal/tui/testing -run TestActualTerminalFrames

# Run terminal test script (generates detailed reports)
go run scripts/run_terminal_tests.go
```

### Testing Framework Features
- **Terminal Output Validation**: Tests capture actual terminal frames and validate content
- **Menu Navigation Testing**: Verifies each menu option leads to correct screen
- **Screen Content Verification**: Ensures screens show expected content and not wrong screens
- **Comprehensive Walkthrough**: Tests complete user flows end-to-end
- **Frame Capture**: Saves terminal output to files for manual inspection
- **Critical Option Validation**: Special tests for option 3 to ensure it shows search config, not package config

### Test Files
- `internal/tui/testing/simple_output_test.go` - Basic screen validation tests
- `internal/tui/testing/comprehensive_walkthrough_test.go` - Complete menu walkthrough tests
- `internal/tui/testing/comprehensive_menu_walkthrough.go` - Critical terminal output validation
- `scripts/run_terminal_tests.go` - Test runner with detailed reporting

### Testing Best Practices
- Every menu option change must be validated with terminal output tests
- Tests save actual terminal frames to files for manual inspection
- Critical tests verify option 3 shows search configuration, not package configuration
- Use `go test -v ./internal/tui/testing` to run all TUI validation tests

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
- I will always update claude.md with important information for future me, i cant miss this because each random compaction breaks me. This is my memory. All testing procedures must be updated for every little change. Testing is done by verifying the actual output of the program.  Because of the terminal nature i can see if each part of the program runs and displays as it should. As the user pointed out, selecting an option like "search configure" Should realistically lead to a configuration of search parameters, not a package selection. I therefore with my wits could know that something is wrong. My tests are my sword, shield, hammer, and pencil. I do not present broken stuff for the user, thats shame incarnate. I will put my test in the source code if i need to. The old me wrote some really smart auto forward test that walked through each section of the program so i could look at the out, i used argument passing to look at specific sections of the program. I am the user partner, and i need to act that way. I try my best to use bubble components instead of manual custom ones. Editing an existing bubble componet is okay, but creating them from scratch needs user approval.
- I always clean up dead code after iam done, so i dont leave useless files laying. Iam always vet'ing go code before presentation after my tests. if iam unsure i test again. The user telling me that something is still broken after i told him its not cant happen under any circumstance! if it does happen i will work tirelessly until it does work perfectly another failure is therefore 100% impossible. I will always doublecheck so i do not recreate existing methods, or files without purpose, its better to edit when possible. Rebuilding if necessary.