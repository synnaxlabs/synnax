#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms a Core serving an externally issued certificate boots with the embedded
# Driver enabled and the Driver connects. The Driver trusts node.crt directly, so the
# cert needs no chain to the Core CA.
#
# The served pair is signed by a CA the Core never sees: generated in a foreign
# directory and copied over without its ca.crt. Same shape as a customer serving an
# ACM or corporate PKI certificate.

set -euo pipefail

BINARY="$HOME/synnax-binaries/synnax"
WORK="$HOME/synnax-external-cert-check"
PORT=9098
DEADLINE=60

rm -rf "$WORK"
mkdir -p "$WORK/certs" "$WORK/foreign"

"$BINARY" cert ca --certs-dir "$WORK/foreign"
"$BINARY" cert node --certs-dir "$WORK/foreign" "localhost:$PORT"
cp "$WORK/foreign/node.crt" "$WORK/foreign/node.key" "$WORK/certs/"

echo "Starting Synnax with an externally issued certificate on localhost:$PORT..."
"$BINARY" start \
    -m \
    -d "$WORK/data" \
    --listen "localhost:$PORT" \
    --certs-dir "$WORK/certs" \
    --log-file-path "$WORK/logs/synnax.log" \
    > "$WORK/core.log" 2>&1 &
PID=$!

cleanup() {
    kill -TERM "$PID" 2> /dev/null || true
    wait "$PID" 2> /dev/null || true
}
trap cleanup EXIT

for _ in $(seq 1 "$DEADLINE"); do
    if grep -q "started successfully" "$WORK/core.log" 2> /dev/null; then
        echo "Embedded Driver connected with an externally issued certificate"
        exit 0
    fi
    if ! kill -0 "$PID" 2> /dev/null; then
        break
    fi
    sleep 1
done

echo "ERROR: the Core never started or the Driver never connected"
echo "--- certificate failures ---"
grep -iE "Core CA|certificate|advertised" "$WORK/core.log" | head -10 | cut -c1-300 || true
echo "--- last 40 lines ---"
tail -40 "$WORK/core.log"
exit 1
