#!/bin/bash

# Install embedded console into core
# This script builds the console web assets and copies them to the core server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
CONSOLE_DIR="$REPO_ROOT/console"
CORE_DIR="$REPO_ROOT/core"
CONSOLE_DIST_DIR="$CONSOLE_DIR/dist"
CORE_CONSOLE_DIR="$CORE_DIR/pkg/service/console/dist"

echo "Building console web assets..."
cd "$REPO_ROOT"
pnpm build:console-vite

echo "Creating core console assets directory..."
mkdir -p "$CORE_CONSOLE_DIR"

echo "Copying console assets to core..."
cp -r "$CONSOLE_DIST_DIR"/* "$CORE_CONSOLE_DIR/"

echo "✓ Console assets installed to $CORE_CONSOLE_DIR"
