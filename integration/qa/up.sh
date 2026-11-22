#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Starts a Core with the web Console embedded on new data, writes live data, and opens
# a headless Chromium logged in to the Console with the sample project open. An agent
# drives that browser through the Chrome DevTools Protocol. core.sh restarts the Core.
#
# Environment: SYNNAX_LICENSE_KEY (required), QA_DIR (default /tmp/console-qa),
# QA_PORT for the Core (default 9090), QA_CDP_PORT for Chromium (default 9222).
# Background processes stop on `xargs kill < "$QA_DIR/pids"`, and on a new run.

set -euo pipefail

: "${SYNNAX_LICENSE_KEY:?SYNNAX_LICENSE_KEY must hold a Core license}"
QA_DIR=${QA_DIR:-/tmp/console-qa}
QA_PORT=${QA_PORT:-9090}
QA_CDP_PORT=${QA_CDP_PORT:-9222}
ROOT=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
mkdir -p "$QA_DIR"
QA_DIR=$(cd "$QA_DIR" && pwd)
if [ ! -O "$QA_DIR" ]; then
    echo "$QA_DIR belongs to another user" >&2
    exit 1
fi
chmod 700 "$QA_DIR"
mkdir -p "$QA_DIR/images"
# Stops the processes of an earlier run, skipping PIDs the OS has since reused. They
# must exit before their files go, or Chromium writes into the profile as it is deleted.
if [ -f "$QA_DIR/pids" ]; then
    while read -r pid; do
        case $(ps -p "$pid" -o command= 2> /dev/null) in
            *"$QA_DIR"* | *qa/live.py*)
                kill "$pid" 2> /dev/null || continue
                for _ in $(seq 20); do
                    kill -0 "$pid" 2> /dev/null || continue 2
                    sleep 0.5
                done
                kill -9 "$pid" 2> /dev/null || true
                ;;
        esac
    done < "$QA_DIR/pids"
fi
rm -rf "$QA_DIR"/{pids,chromium,data,core.pid,live.pid,core.log,live.log}
touch "$QA_DIR/pids"
trap 'xargs kill < "$QA_DIR/pids" 2> /dev/null' ERR

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

for port in "$QA_PORT" "$QA_CDP_PORT"; do
    if nc -z localhost "$port" > /dev/null 2>&1; then
        echo "Port $port is in use" >&2
        exit 1
    fi
done

QA_DIR=$QA_DIR QA_PORT=$QA_PORT integration/qa/core.sh
(cd console && node scripts/create_sample.ts "$QA_PORT")

cd integration
if [ "$(uname)" = Linux ]; then
    uv run playwright install --with-deps chromium
else
    uv run playwright install chromium
fi
chromium=$(uv run python -c "from playwright.sync_api import sync_playwright as p
with p() as pw: print(pw.chromium.executable_path)")
flags=(--headless=new --remote-debugging-port="$QA_CDP_PORT"
    --user-data-dir="$QA_DIR/chromium" "--window-size=1440,900"
    --enable-unsafe-swiftshader --no-first-run --no-default-browser-check)
# Chromium refuses to start its sandbox as root.
if [ "$(id -u)" = 0 ]; then flags+=(--no-sandbox); fi
nohup "$chromium" "${flags[@]}" > "$QA_DIR/chromium.log" 2>&1 &
echo $! >> "$QA_DIR/pids"
wait_port "$QA_CDP_PORT"

uv run python qa/login.py "$QA_PORT" "$QA_CDP_PORT"
echo "Console: http://localhost:$QA_PORT, CDP: http://localhost:$QA_CDP_PORT"
