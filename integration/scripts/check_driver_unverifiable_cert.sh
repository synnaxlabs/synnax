#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms a Core refuses to boot when the advertised listener serves a certificate
# the embedded Driver cannot verify: not the node certificate and not signed by the
# Core CA. Guards the startup check itself; the positive checks alone cannot tell a
# relaxed check from a deleted one.

set -euo pipefail

BINARY="$HOME/synnax-binaries/synnax"
WORK="$HOME/synnax-unverifiable-cert-check"
PORT=9097
DEADLINE=60

rm -rf "$WORK"
mkdir -p "$WORK/certs" "$WORK/foreign"

"$BINARY" cert ca --certs-dir "$WORK/certs"
"$BINARY" cert node --certs-dir "$WORK/certs" "localhost:$PORT"
"$BINARY" cert ca --certs-dir "$WORK/foreign"
"$BINARY" cert node --certs-dir "$WORK/foreign" "localhost:$PORT"

cat > "$WORK/config.yaml" << EOF
listen:
  - address: localhost:$PORT
    advertise: true
    cert:
      source: file
      cert: $WORK/foreign/node.crt
      key: $WORK/foreign/node.key
EOF

echo "Starting Synnax with a certificate outside the trust anchors..."
"$BINARY" start \
    -m \
    -d "$WORK/data" \
    -c "$WORK/config.yaml" \
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
    if grep -q "the embedded Driver cannot verify it" "$WORK/core.log" 2> /dev/null; then
        echo "Core refused the unverifiable advertised certificate"
        exit 0
    fi
    if grep -q "started successfully" "$WORK/core.log" 2> /dev/null; then
        echo "ERROR: the Core booted with a certificate the Driver cannot verify"
        tail -40 "$WORK/core.log"
        exit 1
    fi
    if ! kill -0 "$PID" 2> /dev/null; then
        break
    fi
    sleep 1
done

if grep -q "the embedded Driver cannot verify it" "$WORK/core.log" 2> /dev/null; then
    echo "Core refused the unverifiable advertised certificate"
    exit 0
fi
echo "ERROR: the Core never rejected the certificate"
tail -40 "$WORK/core.log"
exit 1
