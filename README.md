Flutter Package Manager

A cross-platform tool that transforms GitHub into your private package manager for Flutter projects. Easily add GitHub repositories as git dependencies with an interactive interface.
⚡ Quick Start (One-Line Install)
🐧 Linux/macOS
🚀 Install & Run Immediately (remove the first 'bash' word if having problems)

🏃 Run Directly (No Installation)

curl -sSL https://raw.githubusercontent.com/daslaller/flutter_packagemanager_setup/main/install/run.sh | bash

📦 Install Only (Run Later)

curl -sSL https://raw.githubusercontent.com/daslaller/flutter_packagemanager_setup/main/install/install.sh | bash -s -- --no-run
flutter-pm  # Run anytime!

🪟 Windows
🚀 Install & Run Immediately

iwr -useb https://raw.githubusercontent.com/daslaller/flutter_packagemanager_setup/main/install/install.ps1 | iex

🏃 Run Directly (No Installation)

iwr -useb https://raw.githubusercontent.com/daslaller/flutter_packagemanager_setup/main/install/run.ps1 | iex

📦 Install Only (Run Later)

iwr -useb https://raw.githubusercontent.com/daslaller/flutter_packagemanager_setup/main/install/install.ps1 | iex -NoRun
flutter-pm  # Run anytime!

🚀 Features

    🤖 Smart Dependency Recommendations: AI-powered code analysis that detects Flutter patterns and suggests high-quality packages with intelligent quality scoring
    🔍 Enhanced Project Discovery:
        Local Scan: Automatically finds Flutter projects in common directories
        GitHub Fetch: Clone Flutter projects directly from GitHub with custom save locations
    📦 Multi-Repository Selection: Select multiple repositories at once using an interactive interface
    🎯 Cross-Platform: Works on Linux, macOS, and Windows
    🔐 GitHub Integration: Seamless authentication and repository access via GitHub CLI
    ⚡ Interactive UI: Spacebar to select, arrow keys to navigate, Enter to confirm
    🛡️ Safe Operations: Automatic backups before modifying pubspec.yaml files
🎮 Interactive Interface

The multiselect interface provides an intuitive way to choose multiple repositories:

Select repositories (SPACE to select, ENTER to confirm):

Use ↑/↓ or j/k to navigate, SPACE to select/deselect, ENTER to confirm, q to quit

  [ ] user/flutter_widgets (public) - Custom Flutter widgets
► [✓] user/api_client (private) - API client library  
  [✓] org/shared_models (public) - Shared data models
  [ ] user/another_package (public) - Another useful package

Selected: 2 items

📦 Generated Dependencies

The script adds dependencies to your pubspec.yaml in this format:

dependencies:
  flutter:
    sdk: flutter
  custom_widgets:
    git:
      url: https://github.com/user/flutter_widgets.git
      ref: main
  api_client:
    git:
      url: https://github.com/user/api_client.git
      ref: v1.2.0

🔧 Advanced Usage
Custom Repository URLs

You can also add repositories by providing URLs directly instead of selecting from your repositories.
Branch/Tag Selection

For each repository, you can specify:

    Specific branches (e.g., develop, feature/new-api)
    Tagged releases (e.g., v1.0.0, v2.1.3)
    Commit hashes for precise version control

Backup and Recovery

    Original pubspec.yaml files are automatically backed up as .backup
    If a package already exists, you'll be prompted to replace it
    Failed operations don't affect your original files
