#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms the embedded Driver connects to a Core serving TLS. The rest of the
# integration suite runs insecure, so this is the only coverage of that path.
#
# The listen list with an auto advertised listener is load-bearing. A single
# file-source listener serves node.crt, which is what the Driver used to pin, so it
# passes even when nothing else can verify the Core.

set -euo pipefail

BINARY="$HOME/synnax-binaries/synnax"
WORK="$HOME/synnax-tls-check"
PORT=9099
DEADLINE=60

rm -rf "$WORK"
mkdir -p "$WORK/certs"

cat > "$WORK/config.yaml" <<EOF
listen:
  - address: localhost:$PORT
    advertise: true
    cert:
      source: auto
EOF

# A listen list rejects --auto-cert, and the security provider loads the node pair
# whatever source the listeners use, so generate both the way an operator would.
"$BINARY" cert ca --certs-dir "$WORK/certs"
"$BINARY" cert node --certs-dir "$WORK/certs" "localhost:$PORT"

echo "Starting Synnax with TLS on localhost:$PORT..."
"$BINARY" start \
    -m \
    -d "$WORK/data" \
    -c "$WORK/config.yaml" \
    --certs-dir "$WORK/certs" \
    --log-file-path "$WORK/logs/synnax.log" \
    > "$WORK/core.log" 2>&1 &
PID=$!

cleanup() {
    kill -TERM "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
}
trap cleanup EXIT

for _ in $(seq 1 "$DEADLINE"); do
    if grep -q "started successfully" "$WORK/core.log" 2>/dev/null; then
        echo "Embedded Driver connected over TLS"
        exit 0
    fi
    if ! kill -0 "$PID" 2>/dev/null; then
        break
    fi
    sleep 1
done

echo "ERROR: the embedded Driver never connected to a Core serving TLS"
echo "--- handshake failures ---"
grep -iE "handshake|certificate|verify" "$WORK/core.log" | head -10 | cut -c1-300 || true
echo "--- last 40 lines ---"
tail -40 "$WORK/core.log"
exit 1
