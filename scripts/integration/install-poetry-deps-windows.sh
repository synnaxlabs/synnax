#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# install-poetry-deps-windows.sh
# Installs Poetry and Python dependencies on Windows using bash
# Used by GitHub Actions workflow: test.integration.yaml

set -euo pipefail

echo "Installing Poetry and dependencies on Windows..."

# Change to the integration test directory
cd "integration/test/py"

# Install Poetry on Windows
curl -sSL https://install.python-poetry.org | python -

# Add Poetry to PATH for current session
export PATH="$HOME/.local/bin:$PATH"

# Verify Poetry is available
echo "Verifying Poetry installation..."
poetry --version

# Remove existing lock file and recreate it fresh
if [ -f "poetry.lock" ]; then 
    rm "poetry.lock"
fi

# Install dependencies
poetry env remove --all || true
poetry install --no-cache

echo "Poetry and dependencies installed successfully"