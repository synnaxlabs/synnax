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

echo "Cleaning up synnax directories..."
[ -d "$HOME/synnax-binaries" ] && rm -rf "$HOME/synnax-binaries" && echo "Removed synnax-binaries directory from $HOME" || echo "No synnax-binaries directory found in $HOME"
[ -d "$HOME/synnax-data" ] && rm -rf "$HOME/synnax-data" && echo "Removed synnax-data directory from $HOME" || echo "No synnax-data directory found in $HOME"

# Check if directories still exist after cleanup
echo "Verifying directory cleanup..."
if [ -d "$HOME/synnax-binaries" ] || [ -d "$HOME/synnax-data" ]; then
    echo "❌ ERROR: synnax directories still exist after cleanup:"
    [ -d "$HOME/synnax-binaries" ] && echo "  - $HOME/synnax-binaries still exists"
    [ -d "$HOME/synnax-data" ] && echo "  - $HOME/synnax-data still exists"
    echo "Directory cleanup failed"
    exit 1
else
    echo "✅ Directory cleanup verified - no synnax directories found"
fi

echo "Checking for existing synnax processes..."
if pgrep -f "synnax" > /dev/null 2>&1; then
    echo "Found synnax processes, killing them..."
    pkill -f "synnax" 2> /dev/null || true
    sleep 2
    pkill -9 -f "synnax" 2> /dev/null || true
    echo "Synnax processes killed"
else
    echo "No synnax processes found"
fi

echo "Synnax process cleanup completed"
exit 0
