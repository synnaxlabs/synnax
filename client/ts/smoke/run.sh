#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Packs the client and its Synnax dependencies, installs the tarballs in a new project
# outside the workspace, and runs smoke.mjs in Node.js (current and 22.12.0), Deno, Bun,
# Chromium, and WebKit. The test only reads from the Core at SYNNAX_TEST_HOST and
# SYNNAX_TEST_PORT (default localhost:9090).

set -euo pipefail

smoke=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$smoke/../../.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

pnpm --dir "$repo" build:client
for dir in x alamos freighter client; do
    pnpm --dir "$repo/$dir/ts" pack --out "$work/$dir.tgz" > /dev/null
done

# The overrides keep pnpm from fetching the Synnax dependencies from the registry. A
# hoisted node_modules is the layout that npm makes.
cd "$work"
cp "$smoke/smoke.mjs" "$smoke/index.html" .
cat > package.json << EOF
{ "type": "module", "dependencies": { "@synnaxlabs/client": "file:client.tgz" } }
EOF
cat > pnpm-workspace.yaml << EOF
nodeLinker: hoisted
trustPolicy: no-downgrade
fetchRetries: 5
overrides:
  "@synnaxlabs/x": file:$work/x.tgz
  "@synnaxlabs/alamos": file:$work/alamos.tgz
  "@synnaxlabs/freighter": file:$work/freighter.tgz
EOF
pnpm install --reporter=silent

echo "Node.js $(node --version)"
node smoke.mjs
# The workspace tooling needs a newer Node.js, so only this test covers the floor.
echo "Node.js 22.12.0"
pnpm --package=node@runtime:22.12.0 dlx node smoke.mjs
deno --version | head -1
# Immer reads NODE_ENV when it loads.
deno run --allow-net --allow-env=NODE_ENV,SYNNAX_TEST_HOST,SYNNAX_TEST_PORT smoke.mjs
echo "Bun $(bun --version)"
bun smoke.mjs
node "$smoke/browser.mjs" "$work"
