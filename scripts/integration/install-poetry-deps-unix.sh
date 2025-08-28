#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# install-poetry-deps-unix.sh
# Installs Poetry and Python dependencies on Unix systems (Linux/macOS)
# Used by GitHub Actions workflow: test.integration.yaml

set -euo pipefail

echo "Installing Poetry and dependencies..."

# Change to the integration test directory
cd integration/test/py

# Install Poetry and dependencies via pyproject.toml
curl -sSL https://install.python-poetry.org | python3 -
export PATH="$HOME/.local/bin:$PATH"
poetry install

echo "Poetry and dependencies installed successfully"