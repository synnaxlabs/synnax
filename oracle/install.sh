#!/bin/bash
# Copyright 2025 Synnax Labs, Inc.
#
# Oracle Installation Script for macOS
# Builds oracle, installs the Cursor/VSCode LSP extension, and adds oracle to PATH.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORACLE_ROOT="$SCRIPT_DIR"
EXTENSION_DIR="$ORACLE_ROOT/lsp/extensions/vscode"
INSTALL_DIR="$HOME/.local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_step() {
    echo -e "${BLUE}[$1/$TOTAL_STEPS]${NC} $2"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}!${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║              Oracle Installation Script                       ║"
echo "║              for macOS                                        ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Determine what to install
INSTALL_CLI=true
INSTALL_EXTENSION=true
ADD_TO_PATH=true

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --cli-only)
            INSTALL_EXTENSION=false
            shift
            ;;
        --extension-only)
            INSTALL_CLI=false
            ADD_TO_PATH=false
            shift
            ;;
        --no-path)
            ADD_TO_PATH=false
            shift
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --cli-only        Only build and install the CLI (skip extension)"
            echo "  --extension-only  Only install the VSCode/Cursor extension"
            echo "  --no-path         Don't add oracle to PATH"
            echo "  --install-dir DIR Install oracle binary to DIR (default: ~/.local/bin)"
            echo "  -h, --help        Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Calculate total steps
TOTAL_STEPS=0
if $INSTALL_CLI; then ((TOTAL_STEPS+=1)); fi
if $ADD_TO_PATH; then ((TOTAL_STEPS+=1)); fi
if $INSTALL_EXTENSION; then ((TOTAL_STEPS+=4)); fi

CURRENT_STEP=0

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go first: https://go.dev/dl/"
    exit 1
fi
print_success "Go $(go version | awk '{print $3}')"

if $INSTALL_EXTENSION; then
    if ! command -v npm &> /dev/null; then
        print_error "npm is not installed. Please install Node.js first: https://nodejs.org/"
        exit 1
    fi
    print_success "npm $(npm --version)"
fi

echo ""

# Step: Build Oracle CLI
if $INSTALL_CLI; then
    ((CURRENT_STEP+=1))
    print_step $CURRENT_STEP "Building Oracle CLI..."

    cd "$ORACLE_ROOT"
    mkdir -p "$INSTALL_DIR"
    go build -o "$INSTALL_DIR/oracle" ./cmd/oracle
    chmod +x "$INSTALL_DIR/oracle"

    print_success "Built: $INSTALL_DIR/oracle"

    # Also copy to extension bin for LSP
    if $INSTALL_EXTENSION; then
        mkdir -p "$EXTENSION_DIR/bin"
        cp "$INSTALL_DIR/oracle" "$EXTENSION_DIR/bin/oracle"
    fi
fi

# Step: Add to PATH
if $ADD_TO_PATH; then
    ((CURRENT_STEP+=1))
    print_step $CURRENT_STEP "Configuring PATH..."

    # Determine shell config file
    SHELL_CONFIG=""
    if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ] || [ "$SHELL" = "/bin/bash" ]; then
        if [ -f "$HOME/.bash_profile" ]; then
            SHELL_CONFIG="$HOME/.bash_profile"
        else
            SHELL_CONFIG="$HOME/.bashrc"
        fi
    fi

    PATH_EXPORT="export PATH=\"$INSTALL_DIR:\$PATH\""

    if [ -n "$SHELL_CONFIG" ]; then
        # Check if already in config
        if grep -q "$INSTALL_DIR" "$SHELL_CONFIG" 2>/dev/null; then
            print_success "PATH already configured in $SHELL_CONFIG"
        else
            echo "" >> "$SHELL_CONFIG"
            echo "# Oracle CLI" >> "$SHELL_CONFIG"
            echo "$PATH_EXPORT" >> "$SHELL_CONFIG"
            print_success "Added to $SHELL_CONFIG"
        fi
    else
        print_warning "Could not determine shell config file"
        echo "    Add this to your shell config manually:"
        echo "    $PATH_EXPORT"
    fi

    # Also export for current session
    export PATH="$INSTALL_DIR:$PATH"
fi

# Steps: Install VSCode/Cursor extension
if $INSTALL_EXTENSION; then
    # Install npm dependencies
    ((CURRENT_STEP+=1))
    print_step $CURRENT_STEP "Installing extension dependencies..."
    cd "$EXTENSION_DIR"
    npm install --silent
    print_success "Dependencies installed"

    # Compile TypeScript
    ((CURRENT_STEP+=1))
    print_step $CURRENT_STEP "Compiling extension..."
    npm run compile --silent
    print_success "Extension compiled"

    # Package extension
    ((CURRENT_STEP+=1))
    print_step $CURRENT_STEP "Packaging extension..."
    npx @vscode/vsce package --allow-missing-repository -o oracle-language.vsix 2>/dev/null
    print_success "Packaged: oracle-language.vsix"

    # Install into Cursor
    ((CURRENT_STEP+=1))
    print_step $CURRENT_STEP "Installing extension into Cursor..."

    CURSOR_CLI=""
    if command -v cursor &> /dev/null; then
        CURSOR_CLI="cursor"
    elif [ -f "/Applications/Cursor.app/Contents/Resources/app/bin/cursor" ]; then
        CURSOR_CLI="/Applications/Cursor.app/Contents/Resources/app/bin/cursor"
    elif [ -f "$HOME/Applications/Cursor.app/Contents/Resources/app/bin/cursor" ]; then
        CURSOR_CLI="$HOME/Applications/Cursor.app/Contents/Resources/app/bin/cursor"
    fi

    # Also check for VSCode
    VSCODE_CLI=""
    if command -v code &> /dev/null; then
        VSCODE_CLI="code"
    elif [ -f "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code" ]; then
        VSCODE_CLI="/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
    fi

    INSTALLED_TO=""

    if [ -n "$CURSOR_CLI" ]; then
        "$CURSOR_CLI" --install-extension "$EXTENSION_DIR/oracle-language.vsix" 2>/dev/null
        INSTALLED_TO="Cursor"
    fi

    if [ -n "$VSCODE_CLI" ]; then
        "$VSCODE_CLI" --install-extension "$EXTENSION_DIR/oracle-language.vsix" 2>/dev/null
        if [ -n "$INSTALLED_TO" ]; then
            INSTALLED_TO="$INSTALLED_TO and VSCode"
        else
            INSTALLED_TO="VSCode"
        fi
    fi

    if [ -n "$INSTALLED_TO" ]; then
        print_success "Installed to $INSTALLED_TO"
    else
        print_warning "Neither Cursor nor VSCode CLI found"
        echo "    Install the extension manually:"
        echo "    1. Open Cursor/VSCode"
        echo "    2. Go to Extensions (Cmd+Shift+X)"
        echo "    3. Click '...' menu -> 'Install from VSIX...'"
        echo "    4. Select: $EXTENSION_DIR/oracle-language.vsix"
    fi
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║              Installation Complete!                           ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

if $INSTALL_CLI; then
    echo "Oracle CLI installed at: $INSTALL_DIR/oracle"
    echo ""
    echo "Try it out:"
    echo "  oracle --help"
    echo "  oracle generate"
    echo ""
fi

if $ADD_TO_PATH && [ -n "$SHELL_CONFIG" ]; then
    echo "To use oracle in this terminal, run:"
    echo "  source $SHELL_CONFIG"
    echo ""
fi

if $INSTALL_EXTENSION; then
    echo "Restart Cursor/VSCode to activate the Oracle language extension."
    echo ""
fi
