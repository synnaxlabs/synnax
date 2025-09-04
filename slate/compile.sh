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

echo "=== Starting Slate compilation pipeline ==="

rm -f slate
rm -f ./cmd/slate-language.vsix
touch ./cmd/slate-language.vsix
rm -f ./lsp/extensions/vscode/bin/slate

# Step 1: Compile slate
echo "Step 1: Compiling Slate..."
go build -o slate main.go

# Step 2: Embed slate into bin directory
echo "Step 2: Embedding Slate into bin directory..."
mkdir -p lsp/extensions/vscode/bin
cp slate lsp/extensions/vscode/bin/slate

# Step 3: Rebuild VSCode VSIX extension
echo "Step 3: Building VSCode extension..."
cd lsp/extensions/vscode
npm install
npx vsce package
cd ../../..

# Step 4: Copy VSIX to cmd directory for Go embedding
echo "Step 4: Copying VSIX for Go embedding..."
cp lsp/extensions/vscode/synnax-slate-*.vsix cmd/slate-language.vsix

# Step 5: Rebuild slate again (final build with embedded VSIX)
echo "Step 5: Final Slate build with embedded VSIX..."
go build -o slate main.go

echo "=== Compilation pipeline complete ==="
echo "VSIX extension available at: lsp/extensions/vscode/synnax-slate-*.vsix"
