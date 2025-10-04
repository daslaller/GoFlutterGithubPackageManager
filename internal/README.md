# Internal Package Documentation

This document provides an overview of all files in the `internal/` folder and their purposes.

## Core Package (`internal/core/`)

The core package contains all business logic and shell script parity implementations.

### Configuration and Environment
- **`env.go`** - Configuration Management and Environment Setup
  - Parse environment variables (FLUTTER_PM_*)
  - Manage application settings (debug, dry-run, quiet modes)
  - Provide structured logging with different output formats (text/JSON)
  - Handle configuration validation and defaults
  - Support both CLI flags and environment variable configuration

### Data Structures
- **`types.go`** - Core Data Structures and Type Definitions
  - Project: Represents a Flutter/Dart project with pubspec.yaml
  - RepoCandidate: GitHub repository that can be added as dependency  
  - PkgSpec: Package specification for git dependencies
  - ActionResult: Standardized result format for all operations
  - Reco: Smart recommendations for improvements
  - Step: TUI workflow state enumeration

### Project Discovery
- **`discovery.go`** - Project and Repository Discovery Logic
  - NearestPubspec: Walk up directory tree to find closest pubspec.yaml
  - ScanCommonRoots: Scan typical development directories for projects
  - Concurrent scanning with proper error handling and timeouts
  - Cross-platform support (Windows, macOS, Linux)
  - Shell script parity for project detection logic

### Dart/Flutter Integration
- **`pub.go`** - Dart/Flutter Pub Command Integration and pubspec.yaml Management
  - FindPubTool: Auto-detect available dart/flutter commands (shell script parity)
  - AddGitDependency: Add git dependencies using pub commands (not direct YAML editing)
  - Sync: Execute pub get/flutter packages get operations
  - CreateBackup: Safe backup creation before modifying pubspec.yaml
  - Cross-platform pub command execution with proper error handling

### Git Operations
- **`git.go`** - Git Operations and GitHub CLI Integration
  - GitHub CLI integration for repository listing and authentication
  - Git clone operations with proper error handling and conflict resolution
  - Git version checking and command availability validation
  - Concurrent Git operations with timeout management
  - SHA-based comparison for precise dependency staleness detection

### Dependency Management
- **`stale.go`** - Stale Dependency Detection and Express Update Functionality
  - CheckStaleHeuristic: Fast 24-hour time-based staleness detection
  - CheckStalePrecise: SHA-based comparison for exact staleness detection
  - ExpressGitUpdate: Bulk update of all stale git dependencies
  - pubspec.lock parsing and analysis for dependency tracking
  - Shell script compatible update workflow and behavior

### Recommendations
- **`reco.go`** - Smart Recommendations System
  - SuggestPopularPkgs: Recommend commonly used Flutter packages
  - GenerateFullRecommendations: Comprehensive project analysis and suggestions
  - pubspec.yaml analysis for missing common dependencies
  - Git dependency optimization suggestions (SSH URLs, branch pinning)
  - Security and best practice recommendations

### Performance
- **`benchmark.go`** - Performance Benchmarking and Optimization Analysis
  - BenchmarkOperation: Measure execution time and memory usage of operations
  - Memory usage tracking and garbage collection analysis
  - Performance comparison reporting between shell script and Go implementation
  - Operation profiling for TUI responsiveness optimization

## TUI Package (`internal/tui/`)

The TUI package contains Terminal User Interface implementations using BubbleTea.

### Active Implementation
- **`parity_model.go`** - Shell Script Parity TUI Implementation (ACTIVE)
  - **Shell Script Parity**: Exact menu structure (1-6 options), 60-second timeout, multi-select interface
  - **Bubbletea Components**: list.Model, spinner.Model, progress.Model, textinput.Model, viewport.Model
  - **Harmonica Vector Smoothing**: Spring physics for smooth scrolling, progress animations, page transitions
  - **Complete Workflow**: All 8 workflow steps matching shell script behavior exactly
  - **Used by**: Run() function - this is the active TUI implementation

### Legacy Implementation  
- **`bubbletea_model.go`** - Legacy BubbleTea TUI Implementation (DEPRECATED)
  - Original TUI implementation that was replaced by parity model
  - Demonstrates proper bubbletea component usage but lacks shell script parity
  - Different menu structure, no timeout behavior, generic workflow steps
  - Maintained for reference only - NOT used in production

## Architecture Overview

```
internal/
├── core/           # Business Logic (Shell Script Parity)
│   ├── env.go      # Configuration & Logging
│   ├── types.go    # Data Structures
│   ├── discovery.go # Project Detection
│   ├── pub.go      # Dart/Flutter Integration
│   ├── git.go      # Git & GitHub Operations
│   ├── stale.go    # Dependency Management
│   ├── reco.go     # Smart Recommendations
│   └── benchmark.go # Performance Measurement
└── tui/            # Terminal User Interface
    ├── parity_model.go     # ACTIVE: Shell Script Parity TUI
    └── bubbletea_model.go  # LEGACY: Original TUI (deprecated)
```

## Key Design Principles

1. **Shell Script Parity**: All core modules maintain exact functional equivalence with the shell script
2. **Git CLI Fidelity**: Always use git CLI for operations (not Go libraries) 
3. **Pub Command Integration**: Use `dart pub add` / `flutter pub add` (not YAML surgery)
4. **Safety First**: Always create backups before modifying files
5. **Cross-Platform**: Works on Windows, macOS, and Linux
6. **No CGO**: Static builds for easy distribution
7. **Proper Components**: Use only bubbletea framework components
8. **Smooth Animations**: Harmonica vector smoothing for all UI transitions

## Usage

The active implementation uses `parity_model.go` for the TUI:

```go
import "github.com/daslaller/GoFlutterGithubPackageManager/internal/tui"

// Start the shell script parity TUI
err := tui.RunParity(cfg, logger)
```

All core functionality is accessed through the `core` package:

```go
import "github.com/daslaller/GoFlutterGithubPackageManager/internal/core"

// Example: Detect projects
projects, err := core.ScanCommonRoots()

// Example: Add git dependency  
result := core.AddGitDependency(logger, &cfg, projectDir, pkgSpec)
```