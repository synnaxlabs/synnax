#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Starts the QA Core that up.sh built on the data in $QA_DIR/data, then the live data
# writer. Run it again to restart the Core: the data stays, and only live data resumes.
# Writes the Core PID to $QA_DIR/core.pid, and both PIDs to $QA_DIR/pids.
#
# Environment: SYNNAX_LICENSE_KEY (required), QA_DIR (default /tmp/console-qa), and
# QA_PORT (default 9090), as for up.sh.

set -euo pipefail

: "${SYNNAX_LICENSE_KEY:?SYNNAX_LICENSE_KEY must hold a Core license}"
QA_DIR=$(cd "${QA_DIR:-/tmp/console-qa}" && pwd)
QA_PORT=${QA_PORT:-9090}
ROOT=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)

live=(qa/live.py "$QA_PORT")
if [ -d "$QA_DIR/data" ]; then live+=(--no-history); fi

# Stops the process in PID file $1 if its command holds $2, and waits for it to exit.
stop() {
    local pid
    [ -f "$1" ] || return 0
    pid=$(cat "$1")
    [[ $(ps -p "$pid" -o command= 2> /dev/null) == *"$2"* ]] || return 0
    kill "$pid" 2> /dev/null || return 0
    for _ in $(seq 60); do
        kill -0 "$pid" 2> /dev/null || return 0
        sleep 0.5
    done
    kill -9 "$pid" 2> /dev/null || true
}

# The old writer can outlive its Core, and two writers on one channel conflict. The old
# Core holds the lock on the data until it exits.
stop "$QA_DIR/live.pid" qa/live.py
stop "$QA_DIR/core.pid" "$QA_DIR/synnax"

nohup "$QA_DIR/synnax" start -i -d "$QA_DIR/data" -l "localhost:$QA_PORT" \
    >> "$QA_DIR/core.log" 2>&1 &
echo $! > "$QA_DIR/core.pid"
echo $! >> "$QA_DIR/pids"
for _ in $(seq 60); do
    nc -z localhost "$QA_PORT" > /dev/null 2>&1 && break
    sleep 1
done
nc -z localhost "$QA_PORT" > /dev/null 2>&1

cd "$ROOT/integration"
nohup uv run python "${live[@]}" >> "$QA_DIR/live.log" 2>&1 &
echo $! > "$QA_DIR/live.pid"
echo $! >> "$QA_DIR/pids"
