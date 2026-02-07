# 🎯 Flutter Package Manager (Go Edition)

> **High-Performance Git Dependency Management for Flutter Projects**

A blazing-fast, cross-platform tool that transforms GitHub into your private package manager for Flutter projects. Built with Go and featuring a beautiful Terminal User Interface (TUI), it provides an intuitive way to add GitHub repositories as git dependencies.

## ⚡ Quick Start (One-Line Install)

### 🐧 Linux/macOS
```bash
curl -fsSL https://raw.githubusercontent.com/daslaller/GoFlutterGithubPackageManager/main/install.sh | bash
```

### 🪟 Windows (PowerShell)
```powershell
iwr -useb https://raw.githubusercontent.com/daslaller/GoFlutterGithubPackageManager/main/install.ps1 | iex
```

### 🚀 Run Immediately
After installation, simply run:
```bash
flutter-pm
```

## 🌟 Key Features

### 🚀 **High-Performance Architecture**
- **Concurrent project discovery** for fast scanning across directories
- **Direct CLI integration** with git, gh, and dart/flutter for reliable operations
- **Intelligent TTL-based caching** for GitHub API, Git operations, and package name lookups
- **Background cache warming** eliminates cold-start latency
- **Optimized UI rendering** with pre-allocated string builders

### 🎮 **Beautiful Terminal Interface**
- **Interactive TUI** with smooth animations and progress indicators
- **Menu-driven interface** matching beloved shell script behavior exactly
- **Real-time spinners** and progress bars for all operations
- **Keyboard shortcuts** for power users (1-6 for direct selection)

### 🤖 **Smart Features**
- **Automatic name mismatch resolution** — detects and fixes stale package name entries in pubspec.yaml
- **Intelligent conflict resolution** — handles version conflicts, git-vs-hosted conflicts, and more
- **Express Git updates** for existing dependencies
- **Nuclear cache clearing** (remove pubspec.lock + clear pub cache)
- **Auto-timeout menu** (60 seconds) with default selection
- **Self-update capability** via git integration

### 🔍 **Enhanced Project Discovery**
- **Local Scan**: Automatically finds Flutter projects in common directories
- **GitHub Clone**: Clone Flutter projects directly from GitHub  
- **Quick Detection**: Instant project detection in current directory
- **Multi-project support**: Handle multiple Flutter projects seamlessly

### 📦 **Advanced Package Management**
- **Multi-repository selection**: Select multiple repositories with interactive interface
- **Branch/tag selection**: Choose specific versions, branches, or commits
- **Batch operations**: Add multiple dependencies efficiently  
- **Automatic backups**: Safe operations with automatic pubspec.yaml backups

### 🔐 **Seamless Integration**
- **GitHub CLI integration**: Automatic authentication and repository access
- **Cross-platform**: Works on Linux, macOS, and Windows
- **Git support**: Full git operations with robust error recovery

## 🎮 Interactive Interface

The Terminal User Interface provides an intuitive menu-driven experience:

```
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║   🎯 Flutter Package Manager                                ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝

📱 Flutter Package Manager - Main Menu:
► 1. 📁 Scan directories
  2. 🐙 GitHub repo  
  3. ⚙️ Configure search
  4. 📦 Use detected: my_flutter_app [DEFAULT]
  5. 🚀 Express Git update for my_flutter_app
  6. 🔄 Check for Flutter-PM updates

💡 Detected Flutter project: my_flutter_app

↑/↓ navigate • enter/1-6 select • q quit
```

### Repository Selection Interface

```
Select repositories to add as Flutter packages:

Navigation: Up/Down arrows | Search: S | Select: SPACE | Confirm: ENTER | Quit: Q

✅ Selected: 2 packages

► [X] 01. 🔓 user/flutter_widgets
     Custom Flutter widgets and UI components
     
  [ ] 02. 🔒 user/api_client  
     RESTful API client with authentication
     
  [X] 03. 🔓 org/shared_models
     Shared data models and utilities

Selected: user/flutter_widgets, org/shared_models
```

## 📦 Generated Dependencies

The tool adds dependencies to your `pubspec.yaml` in the standard format:

```yaml
dependencies:
  flutter:
    sdk: flutter
    
  # Added by Flutter Package Manager
  flutter_widgets:
    git:
      url: https://github.com/user/flutter_widgets.git
      ref: main
      
  api_client:
    git:
      url: https://github.com/user/api_client.git
      ref: v1.2.0
      
  shared_models:
    git:
      url: https://github.com/org/shared_models.git
      ref: main
```

## 🔧 Advanced Usage

### Menu Options Explained

1. **📁 Scan directories** - Search configured directories for Flutter projects
2. **🐙 GitHub repo** - Use GitHub repositories as package source  
3. **⚙️ Configure search** - Configure search paths and settings
4. **📦 Use detected project** - Continue with locally detected project [DEFAULT]
5. **🚀 Express Git update** - Update existing git dependencies
6. **🔄 Check for Flutter-PM updates** - Self-update the tool

### Express Features

- **Express Git Update**: Quickly update all existing git dependencies in your project
- **Nuclear Cache Clear**: Remove `pubspec.lock` + clear pub cache + rebuild from scratch
- **Smart Recommendations**: AI analysis suggests packages based on your code patterns

### Configuration

Access the configuration menu (option 3) to customize:
- Search directories for project discovery
- Search depth for recursive scanning  
- GitHub integration settings

## 🛠️ Prerequisites

- **Flutter SDK** (https://flutter.dev/docs/get-started/install)
- **Git** (https://git-scm.com/)
- **GitHub CLI** (automatically installed if missing)

## 📊 Performance

The Go edition includes built-in performance benchmarking:

```bash
go test -bench=. ./internal/core/
```

## 🔄 Migration from Shell Version

The Go edition maintains **100% behavioral compatibility** with the original shell script while providing significant improvements. All menu options, keyboard shortcuts, and workflows remain identical.

### Key Advantages Over Shell Version
- ✅ **Smart error recovery** — automatically fixes name mismatches, version conflicts, and more
- ✅ **Better error handling** with detailed diagnostics
- ✅ **Cross-platform consistency**
- ✅ **Beautiful progress indicators**
- ✅ **Concurrent operations** for faster scanning
- ✅ **Self-update capability**

## 🐛 Troubleshooting

### Common Issues

**"name" field doesn't match expected name**

This is the most common error when adding git dependencies. It means an existing dependency in your `pubspec.yaml` has a package name that doesn't match the actual name in the repo's pubspec.yaml (e.g., the repo was renamed). Flutter Package Manager **automatically detects and fixes this**:

1. Scans all git dependencies in your local pubspec.yaml
2. Fetches the actual package name from each GitHub repo
3. Fixes any mismatches (creates a backup first)
4. Repairs the dart pub cache
5. Retries the operation

If you encounter this outside the tool, fix it manually:
```bash
# Clear the stale dart pub cache
dart pub cache repair

# Or nuclear option
dart pub cache clean --force
```

**"No Flutter projects found"**
```bash
# Navigate to your Flutter project first
cd /path/to/your/flutter/project
flutter-pm
```

**GitHub authentication failed**
```bash
# Authenticate with GitHub CLI
gh auth login
flutter-pm
```

**Permission denied (Linux/macOS)**
```bash
# Ensure the binary is executable
chmod +x ~/.local/bin/flutter-pm
```

### Manual Installation

If the one-line installer fails, you can install manually:

1. Download the appropriate binary from [Releases](https://github.com/daslaller/GoFlutterGithubPackageManager/releases)
2. Make it executable: `chmod +x flutter-pm`
3. Move to PATH: `mv flutter-pm ~/.local/bin/` (Linux/macOS) or `%LOCALAPPDATA%\flutter-pm\` (Windows)

## 🤝 Contributing

Contributions are welcome! Please check the [Issues](https://github.com/daslaller/GoFlutterGithubPackageManager/issues) page for current tasks and feature requests.

### Development Setup

```bash
git clone https://github.com/daslaller/GoFlutterGithubPackageManager.git
cd GoFlutterGithubPackageManager
go mod download
go run cmd/flutter-pm/main.go
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Links

- **Documentation**: [GitHub Repository](https://github.com/daslaller/GoFlutterGithubPackageManager)
- **Issues**: [Bug Reports & Feature Requests](https://github.com/daslaller/GoFlutterGithubPackageManager/issues)
- **Releases**: [Download Binaries](https://github.com/daslaller/GoFlutterGithubPackageManager/releases)
- **Original Shell Version**: [flutter_packagemanager_setup](https://github.com/daslaller/flutter_packagemanager_setup)

---

**⭐ Star this repo if Flutter Package Manager helps your development workflow!**