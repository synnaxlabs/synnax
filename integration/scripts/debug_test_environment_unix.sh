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

echo "Test failed - debugging info:"

echo "=== Python/Poetry environment ==="
export PATH="$HOME/.local/bin:$PATH"
python --version || echo "Python not found"
poetry --version || echo "Poetry not found"

echo "=== Synnax connectivity ==="
if command -v nc &> /dev/null; then
    nc -z localhost 9090 && echo "Port 9090 reachable" || echo "Port 9090 unreachable"
elif command -v telnet &> /dev/null; then
    timeout 5 telnet localhost 9090 && echo "Port 9090 reachable" || echo "Port 9090 unreachable"
else
    echo "No network testing tools available"
fi

echo "=== Service status ==="
ps aux | grep -v grep | grep synnax || echo "No synnax processes found"

echo "=== Test artifacts ==="
find integration -name "*.png" -o -name "*.log" -o -name "*.json" | head -10

echo "Debug information collection completed"
