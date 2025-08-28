#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# kill-synnax-processes-linux.sh
# Forcibly terminates existing Synnax processes on Linux
# Used by GitHub Actions workflow: test.integration.yaml

#!/bin/bash

echo "Checking for existing synnax processes..."
if pgrep -f "synnax" > /dev/null; then
  echo "Found synnax processes. Terminating..."
  pkill -f "synnax"
  KILL_EXIT_CODE=$?
  echo "Initial kill exit code: $KILL_EXIT_CODE"
  sleep 2
  # Force kill if still running
  if pgrep -f "synnax" > /dev/null; then
    echo "Force killing remaining synnax processes..."
    pkill -9 -f "synnax"
    FORCE_KILL_EXIT_CODE=$?
    echo "Force kill exit code: $FORCE_KILL_EXIT_CODE"
  fi
  echo "All synnax processes terminated."
else
  echo "No synnax processes found."
fi

echo "Script completed successfully"
exit 0