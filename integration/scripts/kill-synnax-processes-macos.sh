#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# kill-synnax-processes-macos.sh
# Forcibly terminates existing Synnax processes on macOS
# Used by GitHub Actions workflow: test.integration.yaml

set -euo pipefail

echo "Checking for existing Synnax processes on macOS..."

echo "Checking for existing synnax processes..."
pkill -f "synnax" 2>/dev/null && echo "Killed synnax processes" || echo "No synnax processes found"
sleep 2
pkill -9 -f "synnax" 2>/dev/null || true

echo "Synnax process cleanup completed"
exit 0