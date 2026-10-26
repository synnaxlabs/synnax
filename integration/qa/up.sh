#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Starts an in-memory Core with the web Console embedded, writes live data, and opens a
# headless Chromium logged in to the Console with the sample project open. An agent
# drives that browser through the Chrome DevTools Protocol.
#
# Environment: SYNNAX_LICENSE_KEY (required), QA_DIR (default /tmp/console-qa),
# QA_PORT for the Core (default 9090), QA_CDP_PORT for Chromium (default 9222).
# Background processes stop on `kill $(cat "$QA_DIR/pids")`.

set -euo pipefail

: "${SYNNAX_LICENSE_KEY:?SYNNAX_LICENSE_KEY must hold a Core license}"
QA_DIR=${QA_DIR:-/tmp/console-qa}
QA_PORT=${QA_PORT:-9090}
QA_CDP_PORT=${QA_CDP_PORT:-9222}
ROOT=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
mkdir -p "$QA_DIR/images"
: > "$QA_DIR/pids"

wait_port() {
    for _ in $(seq 60); do
        nc -z localhost "$1" > /dev/null 2>&1 && return
        sleep 1
    done
    nc -z localhost "$1" > /dev/null 2>&1
}

cd "$ROOT"
pnpm install --frozen-lockfile
pnpm build:console-vite
cp -r console/dist/. core/pkg/console/dist/
(cd core && go build -tags=console -o "$QA_DIR/synnax" .)

"$QA_DIR/synnax" start -mi -l "localhost:$QA_PORT" > "$QA_DIR/core.log" 2>&1 &
echo $! >> "$QA_DIR/pids"
wait_port "$QA_PORT"
(cd console && node scripts/create_sample.ts "$QA_PORT")

cd integration
if [ "$(uname)" = Linux ]; then
    uv run playwright install --with-deps chromium
else
    uv run playwright install chromium
fi
nohup uv run python qa/live.py "$QA_PORT" > "$QA_DIR/live.log" 2>&1 &
echo $! >> "$QA_DIR/pids"

chromium=$(uv run python -c "from playwright.sync_api import sync_playwright as p
with p() as pw: print(pw.chromium.executable_path)")
nohup "$chromium" --headless=new --remote-debugging-port="$QA_CDP_PORT" \
    --user-data-dir="$QA_DIR/chromium" --window-size=1440,900 \
    --enable-unsafe-swiftshader --no-first-run --no-default-browser-check \
    > "$QA_DIR/chromium.log" 2>&1 &
echo $! >> "$QA_DIR/pids"
wait_port "$QA_CDP_PORT"

uv run python qa/login.py "$QA_PORT" "$QA_CDP_PORT"
echo "Console: http://localhost:$QA_PORT, CDP: http://localhost:$QA_CDP_PORT"
