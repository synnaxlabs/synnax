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

echo "📦 Installing Playwright browsers on Unix..."

# Change to the integration test directory
cd integration

# Ensure Poetry is in PATH
export PATH="$HOME/.local/bin:$PATH"

# Install Playwright browsers with system dependencies
echo "Installing Playwright browsers..."
poetry run playwright install --with-deps

echo "✅ Playwright browsers installed successfully"
