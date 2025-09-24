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

echo "Starting Synnax server..."

# Create data directory
mkdir -p $HOME/synnax-data

# Start Synnax in background
cd $HOME/synnax-data
$HOME/synnax-binaries/synnax start -mi &

# Wait for startup
echo "Waiting for server startup..."
sleep 5

# Verify Synnax is running
if pgrep -f "synnax" > /dev/null; then
    echo "Synnax is running"

    # Verify port 9090 is listening
    portReady=false
    for i in {1..5}; do
        if nc -z localhost 9090; then
            echo "Port 9090 is ready"
            portReady=true
            break
        fi
        echo "Waiting for port 9090... (attempt $i/5)"
        sleep 3
    done

    if [ "$portReady" = false ]; then
        echo "ERROR: Port 9090 never became available"
        exit 1
    fi
else
    echo "ERROR: Synnax process not found after startup"
    exit 1
fi

echo "Synnax server started successfully and is ready!"

# Output Synnax version
$HOME/synnax-binaries/synnax version
