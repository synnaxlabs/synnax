#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# debug-test-environment-windows.sh
# Provides debugging information when tests fail on Windows using bash
# Used by GitHub Actions workflow: test.integration.yaml

echo "Test failed - debugging info:"
echo "Matrix OS: windows"

echo "=== Python/Poetry environment ==="
# Add Poetry to PATH for current session
export PATH="$HOME/.local/bin:$PATH"
python --version || echo "Python not found"
poetry --version || echo "Poetry not found"

echo "=== Synnax connectivity ==="
# Test port connectivity using timeout and nc/bash
timeout 5 bash -c 'until nc -z localhost 9090 2>/dev/null; do sleep 1; done' && echo "Port 9090 reachable" || echo "Port 9090 unreachable"

echo "=== Service status ==="
# Use Windows tasklist command
tasklist | grep synnax || echo "No synnax processes found"

echo "=== Test artifacts ==="
# Find test artifacts
find integration/test/py -name "*.png" -o -name "*.log" -o -name "*.json" | head -10 || echo "No test artifacts found"

echo "Debug information collection completed"