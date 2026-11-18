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
# and 22.12.0), Deno, Bun, Chromium, Firefox, and WebKit. The test only reads from the
# Core at SYNNAX_TEST_HOST and SYNNAX_TEST_PORT (default localhost:9090).

set -euo pipefail

smoke=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$smoke/../../.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

"$repo/scripts/pack_ts.sh" client "$work"
cd "$work"
cp "$smoke/smoke.mjs" "$smoke/index.html" "$smoke/importmap.html" .
echo "Node.js $(node --version)"
node smoke.mjs
# The workspace tooling needs a newer Node.js, so only this test covers the floor.
echo "Node.js 22.12.0"
pnpm --package=node@runtime:22.12.0 dlx node smoke.mjs
deno --version | head -1
deno run --allow-net --allow-env=SYNNAX_TEST_HOST,SYNNAX_TEST_PORT smoke.mjs
echo "Bun $(bun --version)"
bun smoke.mjs
node "$smoke/browser.mjs" "$work"
