#!/bin/bash
# Flutter Package Manager (Go) - Linux/macOS Installer
# One-line install: curl -fsSL https://raw.githubusercontent.com/daslaller/GoFlutterGithubPackageManager/main/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

# Default installation directory
INSTALL_DIR="${HOME}/.local/bin"
FORCE_INSTALL=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE_INSTALL=true
            shift
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -h|--help)
            echo "Flutter Package Manager (Go) Installer"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --force          Force reinstall even if already installed"
            echo "  --install-dir    Custom installation directory (default: ~/.local/bin)"
            echo "  -h, --help       Show this help message"
            echo ""
            echo "One-line install:"
            echo "  curl -fsSL https://raw.githubusercontent.com/daslaller/GoFlutterGithubPackageManager/main/install.sh | bash"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# ASCII Art Header
cat << 'EOF'
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║   🎯 Flutter Package Manager (Go Edition)                   ║
║   🚀 High-Performance Git Dependency Management             ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
EOF

echo ""
echo -e "${YELLOW}🔧 Installing Flutter Package Manager...${NC}"

# Check if already installed
if command -v flutter-pm >/dev/null 2>&1 && [ "$FORCE_INSTALL" = false ]; then
    existing_path=$(command -v flutter-pm)
    echo -e "${GREEN}✅ Flutter Package Manager is already installed at: $existing_path${NC}"
    echo -e "${GRAY}📍 Location: $existing_path${NC}"
    echo ""
    echo -e "${YELLOW}💡 To reinstall, run with --force:${NC}"
    echo -e "${GRAY}   curl -fsSL https://raw.githubusercontent.com/daslaller/GoFlutterGithubPackageManager/main/install.sh | bash -s -- --force${NC}"
    echo ""
    echo -e "${GREEN}🚀 Run 'flutter-pm' to start!${NC}"
    exit 0
fi

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $OS in
    linux*)
        OS="linux"
        ;;
    darwin*)
        OS="darwin"
        ;;
    *)
        echo -e "${RED}❌ Unsupported operating system: $OS${NC}"
        echo -e "${YELLOW}💡 Supported: Linux, macOS${NC}"
        exit 1
        ;;
esac

case $ARCH in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    armv7l)
        ARCH="arm"
        ;;
    i386|i686)
        ARCH="386"
        ;;
    *)
        echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
        echo -e "${YELLOW}💡 Supported: amd64, arm64, arm, 386${NC}"
        exit 1
        ;;
esac

# Create install directory
mkdir -p "$INSTALL_DIR"

# Construct download URL
BINARY_NAME="flutter-pm-$OS-$ARCH"
if [ "$OS" = "windows" ]; then
    BINARY_NAME="$BINARY_NAME.exe"
fi

DOWNLOAD_URL="https://github.com/daslaller/GoFlutterGithubPackageManager/releases/download/main/$BINARY_NAME"
INSTALL_PATH="$INSTALL_DIR/flutter-pm"

echo -e "${YELLOW}🌐 Downloading Flutter Package Manager...${NC}"
echo -e "${GRAY}   Source: $DOWNLOAD_URL${NC}"
echo -e "${GRAY}   Target: $INSTALL_PATH${NC}"

# Download with progress
if command -v curl >/dev/null 2>&1; then
    if ! curl -L --fail --progress-bar "$DOWNLOAD_URL" -o "$INSTALL_PATH"; then
        echo -e "${RED}❌ Download failed${NC}"
        echo -e "${YELLOW}🔗 Please check: $DOWNLOAD_URL${NC}"
        echo -e "${GRAY}💡 You can also download manually and place in: $INSTALL_DIR${NC}"
        exit 1
    fi
elif command -v wget >/dev/null 2>&1; then
    if ! wget --progress=bar:force "$DOWNLOAD_URL" -O "$INSTALL_PATH"; then
        echo -e "${RED}❌ Download failed${NC}"
        echo -e "${YELLOW}🔗 Please check: $DOWNLOAD_URL${NC}"
        echo -e "${GRAY}💡 You can also download manually and place in: $INSTALL_DIR${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ Neither curl nor wget found${NC}"
    echo -e "${YELLOW}💡 Please install curl or wget and try again${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Download completed${NC}"

# Verify download
if [ ! -f "$INSTALL_PATH" ]; then
    echo -e "${RED}❌ Downloaded file not found${NC}"
    exit 1
fi

file_size=$(wc -c < "$INSTALL_PATH")
if [ "$file_size" -lt 1048576 ]; then # Less than 1MB
    echo -e "${RED}❌ Downloaded file appears to be incomplete (size: $file_size bytes)${NC}"
    exit 1
fi

echo -e "${GREEN}✅ File verification passed ($(echo "scale=1; $file_size/1048576" | bc -l 2>/dev/null || echo "unknown") MB)${NC}"

# Make executable
chmod +x "$INSTALL_PATH"

# Add to PATH if necessary
case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        echo -e "${GREEN}✅ Directory already in PATH${NC}"
        ;;
    *)
        echo -e "${YELLOW}🔧 Adding to PATH...${NC}"
        
        # Determine shell configuration file
        shell_config=""
        if [ -n "$ZSH_VERSION" ]; then
            shell_config="$HOME/.zshrc"
        elif [ -n "$BASH_VERSION" ]; then
            if [ -f "$HOME/.bashrc" ]; then
                shell_config="$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                shell_config="$HOME/.bash_profile"
            fi
        fi
        
        if [ -n "$shell_config" ]; then
            if ! grep -q "$INSTALL_DIR" "$shell_config" 2>/dev/null; then
                echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$shell_config"
                echo -e "${GREEN}✅ Added to $shell_config${NC}"
                echo -e "${YELLOW}💡 Restart your terminal or run: source $shell_config${NC}"
            else
                echo -e "${GREEN}✅ Already configured in $shell_config${NC}"
            fi
        else
            echo -e "${YELLOW}⚠️  Could not detect shell configuration file${NC}"
            echo -e "${GRAY}💡 Manually add to your PATH: export PATH=\"$INSTALL_DIR:\$PATH\"${NC}"
        fi
        
        # Update current session
        export PATH="$INSTALL_DIR:$PATH"
        ;;
esac

# Test installation
echo -e "${YELLOW}🧪 Testing installation...${NC}"
if "$INSTALL_PATH" --version >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Installation verified${NC}"
else
    echo -e "${YELLOW}⚠️  Installation completed but verification failed${NC}"
    echo -e "${GRAY}   You may need to restart your terminal${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Installation completed successfully!${NC}"
echo ""
echo -e "${GRAY}📍 Installed to: $INSTALL_PATH${NC}"
echo -e "${CYAN}🚀 Run 'flutter-pm' to start the package manager${NC}"
echo ""
echo -e "${YELLOW}💡 Pro Tips:${NC}"
echo -e "${GRAY}   • Navigate to your Flutter project directory first${NC}"
echo -e "${GRAY}   • Use 'flutter-pm --help' for command-line options${NC}"
echo -e "${GRAY}   • The TUI provides an intuitive menu interface${NC}"
echo ""

# Check prerequisites and provide guidance
if ! command -v flutter >/dev/null 2>&1; then
    echo -e "${YELLOW}📝 Note: Flutter not detected in PATH${NC}"
    echo -e "${GRAY}   Install Flutter from: https://flutter.dev/docs/get-started/install${NC}"
    echo ""
fi

if ! command -v git >/dev/null 2>&1; then
    echo -e "${YELLOW}📝 Note: Git not detected in PATH${NC}"
    echo -e "${GRAY}   Install Git from your package manager or https://git-scm.com/${NC}"
    echo ""
fi

echo -e "${CYAN}🔗 Documentation: https://github.com/daslaller/GoFlutterGithubPackageManager${NC}"
echo -e "${CYAN}🐛 Issues: https://github.com/daslaller/GoFlutterGithubPackageManager/issues${NC}"