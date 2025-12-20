#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

set -euo pipefail

echo "Installing Poetry and dependencies..."
cd integration

# Install Poetry if not already installed
if ! command -v poetry &> /dev/null; then
    echo "Installing Poetry..."
    curl -sSL https://install.python-poetry.org | python3 -
fi

export PATH="$HOME/.local/bin:$PATH"

# Remove any existing virtualenvs that may be using an incompatible Python version
# We delete directly because poetry env remove fails if the Python version is no longer installed
echo "Removing existing virtualenvs..."
rm -rf "$HOME/.cache/pypoetry/virtualenvs/synnax-test-framework-"* 2> /dev/null || true

poetry env use python3
poetry install

echo "Poetry and dependencies installed successfully"
