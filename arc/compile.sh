#!/bin/bash
# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Exit on any error
set -e

echo "=== Starting Arc compilation pipeline ==="

rm -f arc
rm -f ./cmd/arc-language.vsix
touch ./cmd/arc-language.vsix
rm -f ./lsp/extensions/vscode/bin/arc

# Step 1: Compile arc
echo "Step 1: Compiling Arc..."
go build -o arc main.go

# Step 2: Embed arc into bin directory
echo "Step 2: Embedding Arc into bin directory..."
mkdir -p lsp/extensions/vscode/bin
cp arc lsp/extensions/vscode/bin/arc

# Step 3: Rebuild VSCode VSIX extension
echo "Step 3: Building VSCode extension..."
cd lsp/extensions/vscode
npm install
npx vsce package
cd ../../..

# Step 4: Copy VSIX to cmd directory for Go embedding
echo "Step 4: Copying VSIX for Go embedding..."
cp lsp/extensions/vscode/synnax-arc-*.vsix cmd/arc-language.vsix

# Step 5: Rebuild arc again (final build with embedded VSIX)
echo "Step 5: Final Arc build with embedded VSIX..."
go build -o arc main.go

echo "=== Compilation pipeline complete ==="
echo "VSIX extension available at: lsp/extensions/vscode/synnax-arc-*.vsix"
