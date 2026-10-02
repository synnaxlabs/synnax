#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms the Driver CLI reads /etc/synnax/driver.env. The Core serves a self-signed
# certificate whose CA is not in the system trust store, so the login connection check
# can only pass when the trust anchor reaches the binary through that file.
#
# login runs under sudo, which resets the environment. The binary has to read the file
# itself; an exported variable would not survive.

set -euo pipefail

CORE="$HOME/synnax-binaries/synnax"
DRIVER="$HOME/synnax-binaries/synnax-driver"
ENV_FILE="/etc/synnax/driver.env"
WORK="$HOME/synnax-env-file-check"
PORT=9094
DEADLINE=60

rm -rf "$WORK"
mkdir -p "$WORK/certs"

CORE_PID=""
cleanup() {
    sudo rm -f "$ENV_FILE"
    if [ -n "$CORE_PID" ]; then
        kill -TERM "$CORE_PID" 2> /dev/null || true
        wait "$CORE_PID" 2> /dev/null || true
    fi
}
trap cleanup EXIT

fail() {
    echo "ERROR: $1"
    echo "--- login output ---"
    cat "$WORK/login.log" 2> /dev/null || true
    echo "--- core log (last 40 lines) ---"
    tail -40 "$WORK/core.log" 2> /dev/null || true
    exit 1
}

login() {
    printf 'localhost\n%s\ny\nsynnax\nseldon\n' "$PORT" \
        | sudo "$DRIVER" login --state-file "$WORK/state.json" \
            > "$WORK/login.log" 2>&1
}

"$CORE" cert ca --certs-dir "$WORK/certs"
"$CORE" cert node --certs-dir "$WORK/certs" "localhost:$PORT"

echo "Starting a Core with a self-signed certificate on localhost:$PORT..."
"$CORE" start \
    -m \
    -d "$WORK/data" \
    --listen "localhost:$PORT" \
    --certs-dir "$WORK/certs" \
    --log-file-path "$WORK/logs/synnax.log" \
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

sudo rm -f "$ENV_FILE"
echo "Logging in with no environment file..."
if login; then
    fail "login reached an untrusted Core with no trust anchor"
fi
echo "login refused the untrusted Core, as expected"

echo "Writing the trust anchor to $ENV_FILE..."
sudo install -d -m 755 "$(dirname "$ENV_FILE")"
printf '# added by check_driver_env_file.sh\nGRPC_DEFAULT_SSL_ROOTS_FILE_PATH=%s\n' \
    "$WORK/certs/ca.crt" | sudo tee "$ENV_FILE" > /dev/null

echo "Logging in with the environment file in place..."
if ! login; then
    fail "login failed with the trust anchor named by $ENV_FILE"
fi
if [ ! -f "$WORK/state.json" ]; then
    fail "login did not save the connection parameters"
fi

echo "Driver CLI read the environment file"
