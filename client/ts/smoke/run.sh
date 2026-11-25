#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Packs and installs the client with pack_ts.sh, then runs smoke.mjs in Node.js (current
# and 22.12.0), Deno, Bun, Chromium, Firefox, and WebKit, and loads the client with
# require() in both Node.js versions. The test writes channels and projects to the Core
# at SYNNAX_TEST_HOST and SYNNAX_TEST_PORT (default localhost:9090). When
# SYNNAX_TEST_CA_CERT_FILE names a CA certificate, the test also runs in Node.js, Deno,
# and Bun against the TLS Core at SYNNAX_TEST_TLS_PORT (default 9091) that the CA signs.

set -euo pipefail

smoke=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$smoke/../../.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

"$repo/scripts/pack_ts.sh" client "$work"
cd "$work"
cp "$smoke/smoke.mjs" "$smoke/index.html" "$smoke/importmap.html" .
require='require("@synnaxlabs/client")'
echo "Node.js $(node --version)"
node smoke.mjs
node --eval "$require"
# The workspace tooling needs a newer Node.js, so only this test covers the floor.
echo "Node.js 22.12.0"
pnpm --package=node@runtime:22.12.0 dlx node smoke.mjs
pnpm --package=node@runtime:22.12.0 dlx node --eval "$require"
deno --version | head -1
allow_env=SYNNAX_TEST_HOST,SYNNAX_TEST_PORT,SYNNAX_TEST_SECURE,SYNNAX_TEST_CA_CERT
deno run --allow-net --allow-env="$allow_env" smoke.mjs
echo "Bun $(bun --version)"
bun smoke.mjs
if [ -n "${SYNNAX_TEST_CA_CERT_FILE:-}" ]; then
    echo "TLS"
    tls=(SYNNAX_TEST_PORT="${SYNNAX_TEST_TLS_PORT:-9091}" SYNNAX_TEST_SECURE=true)
    ca=$(cat "$SYNNAX_TEST_CA_CERT_FILE")
    env "${tls[@]}" SYNNAX_TEST_CA_CERT="$ca" node smoke.mjs
    # A Deno WebSocket cannot trust a caCert, so Deno takes the CA with --cert.
    env "${tls[@]}" deno run --allow-net --allow-env="$allow_env" \
        --cert "$SYNNAX_TEST_CA_CERT_FILE" smoke.mjs
    env "${tls[@]}" SYNNAX_TEST_CA_CERT="$ca" bun smoke.mjs
else
    echo "Skipping the TLS runs: SYNNAX_TEST_CA_CERT_FILE is not set"
fi
node "$smoke/browser.mjs" "$work"
