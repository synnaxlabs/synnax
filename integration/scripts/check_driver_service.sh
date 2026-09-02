#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms the documented Linux install flow: a binary in /usr/local/bin logs in,
# installs itself as a systemd service, starts, reaches a Core, stops, and uninstalls.
# The install step runs from the installed path, which exercises the self-copy guard.

set -euo pipefail

CORE="$HOME/synnax-binaries/synnax"
DRIVER="$HOME/synnax-binaries/synnax-driver"
INSTALLED="/usr/local/bin/synnax-driver"
UNIT="/etc/systemd/system/synnax-driver.service"
WORK="$HOME/synnax-driver-service-check"
PORT=9095
DEADLINE=60

rm -rf "$WORK"
mkdir -p "$WORK"

CORE_PID=""
cleanup() {
    sudo systemctl stop synnax-driver 2> /dev/null || true
    sudo systemctl disable synnax-driver 2> /dev/null || true
    sudo rm -f "$UNIT"
    sudo systemctl daemon-reload || true
    sudo rm -f "$INSTALLED"
    sudo userdel synnax 2> /dev/null || true
    sudo rm -rf /var/lib/synnax-driver
    if [ -n "$CORE_PID" ]; then
        kill -TERM "$CORE_PID" 2> /dev/null || true
        wait "$CORE_PID" 2> /dev/null || true
    fi
}
trap cleanup EXIT

fail() {
    echo "ERROR: $1"
    echo "--- journal (last 40 lines) ---"
    sudo journalctl -u synnax-driver --no-pager -n 40 || true
    echo "--- core log (last 40 lines) ---"
    tail -40 "$WORK/core.log" 2> /dev/null || true
    exit 1
}

sudo install -m 755 "$DRIVER" "$INSTALLED"

echo "Starting an insecure Core on localhost:$PORT..."
"$CORE" start -mi --listen "localhost:$PORT" -d "$WORK/data" \
    > "$WORK/core.log" 2>&1 &
CORE_PID=$!
for _ in $(seq 1 "$DEADLINE"); do
    if grep -q "started successfully" "$WORK/core.log" 2> /dev/null; then
        break
    fi
    if ! kill -0 "$CORE_PID" 2> /dev/null; then
        fail "the Core exited during startup"
    fi
    sleep 1
done
if ! grep -q "started successfully" "$WORK/core.log" 2> /dev/null; then
    fail "the Core never started"
fi

echo "Logging in..."
if ! printf 'localhost\n%s\nn\nsynnax\nseldon\n' "$PORT" | sudo "$INSTALLED" login; then
    fail "login failed"
fi

echo "Installing the service from the installed binary..."
if ! sudo "$INSTALLED" install; then
    fail "install failed when run from $INSTALLED"
fi
if [ ! -f "$UNIT" ]; then
    fail "install did not write $UNIT"
fi

echo "Starting the service..."
START_TS="$(date '+%Y-%m-%d %H:%M:%S')"
if ! sudo "$INSTALLED" start; then
    fail "start failed"
fi
for _ in $(seq 1 "$DEADLINE"); do
    state=$(systemctl is-active synnax-driver || true)
    if [ "$state" = "active" ]; then
        break
    fi
    if [ "$state" = "failed" ]; then
        fail "the unit entered the failed state"
    fi
    sleep 1
done
if [ "$(systemctl is-active synnax-driver || true)" != "active" ]; then
    fail "the unit never became active"
fi

echo "Waiting for the Driver to reach the Core..."
reached=0
for _ in $(seq 1 "$DEADLINE"); do
    if sudo journalctl -u synnax-driver --since "$START_TS" --no-pager \
        | grep -q "successfully reached cluster"; then
        reached=1
        break
    fi
    sleep 1
done
if [ "$reached" -ne 1 ]; then
    fail "the Driver never reached the Core"
fi

restarts=$(systemctl show -p NRestarts --value synnax-driver)
if [ "$restarts" != "0" ]; then
    fail "the unit restarted $restarts times"
fi

if ! "$INSTALLED" status; then
    fail "status failed while the service was running"
fi

echo "Stopping the service..."
if ! sudo "$INSTALLED" stop; then
    fail "stop failed"
fi
if [ "$(systemctl is-active synnax-driver || true)" = "active" ]; then
    fail "the unit is still active after stop"
fi

echo "Uninstalling the service..."
if ! sudo "$INSTALLED" uninstall; then
    fail "uninstall failed"
fi
if [ -f "$UNIT" ]; then
    fail "uninstall left $UNIT behind"
fi

echo "Driver service lifecycle passed"
