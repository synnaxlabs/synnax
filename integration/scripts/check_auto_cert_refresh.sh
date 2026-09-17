#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms --auto-cert reissues a node certificate that no longer covers the listen
# address, and the embedded Driver connects after the replacement. The first boot
# doubles as the plain --auto-cert happy path over TLS.

set -euo pipefail

BINARY="$HOME/synnax-binaries/synnax"
WORK="$HOME/synnax-auto-cert-refresh-check"
PORT=9096
DEADLINE=60

rm -rf "$WORK"
mkdir -p "$WORK"

cleanup() {
    kill -TERM "$PID" 2> /dev/null || true
    wait "$PID" 2> /dev/null || true
}
trap cleanup EXIT

echo "Starting Synnax with --auto-cert on localhost:$PORT..."
"$BINARY" start \
    -m \
    -d "$WORK/data" \
    --auto-cert \
    --listen "localhost:$PORT" \
    --certs-dir "$WORK/certs" \
    --log-file-path "$WORK/logs/synnax.log" \
    > "$WORK/core1.log" 2>&1 &
PID=$!

started=0
for _ in $(seq 1 "$DEADLINE"); do
    if grep -q "started successfully" "$WORK/core1.log" 2> /dev/null; then
        started=1
        break
    fi
    if ! kill -0 "$PID" 2> /dev/null; then
        break
    fi
    sleep 1
done
if [ "$started" -ne 1 ]; then
    echo "ERROR: the first boot never started"
    tail -40 "$WORK/core1.log"
    exit 1
fi
kill -TERM "$PID"
wait "$PID" 2> /dev/null || true

echo "Restarting on 127.0.0.1:$PORT to force a certificate refresh..."
"$BINARY" start \
    -m \
    -d "$WORK/data" \
    --auto-cert \
    --listen "127.0.0.1:$PORT" \
    --certs-dir "$WORK/certs" \
    --log-file-path "$WORK/logs/synnax.log" \
    > "$WORK/core2.log" 2>&1 &
PID=$!

for _ in $(seq 1 "$DEADLINE"); do
    if grep -q "started successfully" "$WORK/core2.log" 2> /dev/null; then
        if grep -q "replacing node certificate" "$WORK/core2.log" 2> /dev/null; then
            echo "Node certificate refreshed and the embedded Driver reconnected"
            exit 0
        fi
        echo "ERROR: the Core restarted without reissuing the node certificate"
        tail -40 "$WORK/core2.log"
        exit 1
    fi
    if ! kill -0 "$PID" 2> /dev/null; then
        break
    fi
    sleep 1
done

echo "ERROR: the second boot never started"
echo "--- certificate failures ---"
grep -iE "certificate|advertised|cover" "$WORK/core2.log" | head -10 | cut -c1-300 || true
echo "--- last 40 lines ---"
tail -40 "$WORK/core2.log"
exit 1
