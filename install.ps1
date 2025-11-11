# Flutter Package Manager (Go) - Windows Installer - Encoded UTF-8 with BOM
# One-line install: iwr -useb https://raw.githubusercontent.com/daslaller/GoFlutterGithubPackageManager/refs/heads/master/install.ps1 | iex

param(
    [switch]$Force = $false,
    [string]$InstallDir = "$env:LOCALAPPDATA\flutter-pm"
)

$ErrorActionPreference = "Stop"

# Helper function to safely wait for key press
function Wait-ForKeyPress {
    try {
        $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    } catch {
        # ReadKey not supported in this host (e.g., non-interactive environments)
        # Just continue without waiting
    }
}

# ASCII Art Header
Write-Host @"
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║   🎯 Flutter Package Manager (Go Edition)                   ║
║   🚀 High-Performance Git Dependency Management             ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
"@ -ForegroundColor Cyan

Write-Host ""
Write-Host "🔧 Installing Flutter Package Manager..." -ForegroundColor Yellow

# Check if Go version is already installed
$existingInstall = Get-Command "flutter-pm" -ErrorAction SilentlyContinue
if ($existingInstall -and -not $Force) {
    Write-Host "✅ Flutter Package Manager is already installed!" -ForegroundColor Green
    Write-Host "📍 Location: $($existingInstall.Source)" -ForegroundColor Gray
    Write-Host ""

    # Get current version
    try {
        $currentVersion = & flutter-pm --version 2>$null | Select-Object -First 1
        Write-Host "📦 Current Version: $currentVersion" -ForegroundColor Cyan
    } catch {
        Write-Host "📦 Current Version: Unknown" -ForegroundColor Gray
    }

    Write-Host ""
    Write-Host "Would you like to update/reinstall? (Y/N)" -ForegroundColor Yellow -NoNewline
    Write-Host " " -NoNewline
    $response = Read-Host

    if ($response -match '^[Yy]') {
        Write-Host ""
        Write-Host "🔄 Updating Flutter Package Manager..." -ForegroundColor Yellow
        $Force = $true
        # Continue with installation/update
    } else {
        Write-Host ""
        Write-Host "🚀 Run 'flutter-pm' to start!" -ForegroundColor Green
        Write-Host ""
        Write-Host "💡 To force update later, run:" -ForegroundColor Yellow
        Write-Host "   iwr -useb https://raw.githubusercontent.com/daslaller/GoFlutterGithubPackageManager/refs/heads/master/install.ps1 | iex" -ForegroundColor Gray
        Write-Host ""
        Write-Host "Press any key to exit..." -ForegroundColor Gray
        Wait-ForKeyPress
        exit 0
    }
}

# Create install directory
if (-not (Test-Path $InstallDir)) {
    Write-Host "📁 Creating install directory: $InstallDir" -ForegroundColor Gray
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Determine architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$downloadUrl = "https://github.com/daslaller/GoFlutterGithubPackageManager/releases/download/v1.0.0-alpha/flutter-pm-windows-$arch.exe"

Write-Host "🌐 Downloading Flutter Package Manager..." -ForegroundColor Yellow
Write-Host "   Source: $downloadUrl" -ForegroundColor Gray

try {
    $exePath = Join-Path $InstallDir "flutter-pm.exe"
    
    # Download with progress
    $webClient = New-Object System.Net.WebClient
    $webClient.DownloadFile($downloadUrl, $exePath)
    
    Write-Host "✅ Download completed" -ForegroundColor Green
    
    # Verify download
    if (-not (Test-Path $exePath)) {
        throw "Downloaded file not found"
    }
    
    $fileSize = (Get-Item $exePath).Length
    if ($fileSize -lt 1MB) {
        throw "Downloaded file appears to be incomplete (size: $fileSize bytes)"
    }
    
    Write-Host "✅ File verification passed ($([math]::Round($fileSize/1MB, 1)) MB)" -ForegroundColor Green
    
} catch {
    Write-Host "❌ Download failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "🔗 Please check: $downloadUrl" -ForegroundColor Yellow
    Write-Host "💡 You can also download manually and place in: $InstallDir" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Press any key to exit..." -ForegroundColor Gray
    Wait-ForKeyPress
    exit 1
}

# Add to PATH if not already there
$currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($currentPath -notlike "*$InstallDir*") {
    Write-Host "🔧 Adding to PATH..." -ForegroundColor Yellow
    $newPath = "$InstallDir;$currentPath"
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    
    # Update current session PATH
    $env:PATH = "$InstallDir;$env:PATH"
    
    Write-Host "✅ Added to PATH" -ForegroundColor Green
} else {
    Write-Host "✅ Already in PATH" -ForegroundColor Green
}

# Test installation
Write-Host "🧪 Testing installation..." -ForegroundColor Yellow
try {
    & $exePath --version | Out-Null
    Write-Host "✅ Installation verified" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Installation completed but verification failed" -ForegroundColor Yellow
    Write-Host "   You may need to restart your terminal" -ForegroundColor Gray
}

Write-Host ""
Write-Host "🎉 Installation completed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📍 Installed to: $exePath" -ForegroundColor Gray
Write-Host "🚀 Run 'flutter-pm' to start the package manager" -ForegroundColor Cyan
Write-Host ""
Write-Host "💡 Pro Tips:" -ForegroundColor Yellow
Write-Host "   • Navigate to your Flutter project directory first" -ForegroundColor Gray
Write-Host "   • Use 'flutter-pm --help' for command-line options" -ForegroundColor Gray
Write-Host "   • The TUI provides an intuitive menu interface" -ForegroundColor Gray
Write-Host ""

# Check for Flutter and provide guidance
$flutterInstalled = Get-Command "flutter" -ErrorAction SilentlyContinue
if (-not $flutterInstalled) {
    Write-Host "📝 Note: Flutter not detected in PATH" -ForegroundColor Yellow
    Write-Host "   Install Flutter from: https://flutter.dev/docs/get-started/install" -ForegroundColor Gray
    Write-Host ""
}

# Check for Git
$gitInstalled = Get-Command "git" -ErrorAction SilentlyContinue
if (-not $gitInstalled) {
    Write-Host "📝 Note: Git not detected in PATH" -ForegroundColor Yellow
    Write-Host "   Install Git from: https://git-scm.com/download/win" -ForegroundColor Gray
    Write-Host ""
}

Write-Host "🔗 Documentation: https://github.com/daslaller/GoFlutterGithubPackageManager" -ForegroundColor Cyan
Write-Host "🐛 Issues: https://github.com/daslaller/GoFlutterGithubPackageManager/issues" -ForegroundColor Cyan
Write-Host ""
Write-Host "Press any key to exit..." -ForegroundColor Gray
Wait-ForKeyPress
