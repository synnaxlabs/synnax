#!/bin/bash
# Copyright 2025 Synnax Labs, Inc.
#
# Builds and installs the Oracle LSP extension into Cursor.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORACLE_ROOT="$(cd "$SCRIPT_DIR/../../../" && pwd)"
EXTENSION_DIR="$SCRIPT_DIR"

echo "=== Oracle LSP Installer for Cursor ==="
echo ""

# Step 1: Build the Oracle CLI
echo "[1/5] Building Oracle CLI..."
cd "$ORACLE_ROOT"
go build -o "$EXTENSION_DIR/bin/oracle" ./cmd/oracle
echo "      Built: $EXTENSION_DIR/bin/oracle"

# Step 2: Install npm dependencies
echo "[2/5] Installing npm dependencies..."
cd "$EXTENSION_DIR"
npm install

# Step 3: Compile TypeScript
echo "[3/5] Compiling TypeScript..."
npm run compile

# Step 4: Package the extension
echo "[4/5] Packaging extension..."
npx @vscode/vsce package --allow-missing-repository -o oracle-language.vsix

# Step 5: Install into Cursor
echo "[5/5] Installing into Cursor..."

# Find Cursor CLI
CURSOR_CLI=""
if command -v cursor &> /dev/null; then
    CURSOR_CLI="cursor"
elif [ -f "/Applications/Cursor.app/Contents/Resources/app/bin/cursor" ]; then
    CURSOR_CLI="/Applications/Cursor.app/Contents/Resources/app/bin/cursor"
elif [ -f "$HOME/Applications/Cursor.app/Contents/Resources/app/bin/cursor" ]; then
    CURSOR_CLI="$HOME/Applications/Cursor.app/Contents/Resources/app/bin/cursor"
fi

if [ -z "$CURSOR_CLI" ]; then
    echo ""
    echo "ERROR: Cursor CLI not found."
    echo "Please install the extension manually:"
    echo "  1. Open Cursor"
    echo "  2. Go to Extensions (Cmd+Shift+X)"
    echo "  3. Click '...' menu -> 'Install from VSIX...'"
    echo "  4. Select: $EXTENSION_DIR/oracle-language.vsix"
    exit 1
fi

"$CURSOR_CLI" --install-extension "$EXTENSION_DIR/oracle-language.vsix"

echo ""
echo "=== Installation Complete ==="
echo ""
echo "The Oracle language extension has been installed."
echo "Please restart Cursor to activate the extension."
echo ""
echo "The Oracle binary is at: $EXTENSION_DIR/bin/oracle"
echo "You may also want to add it to your PATH or set oracle.lsp.path in settings."
