#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Builds Synnax Desktop on this machine for local testing: the Core sidecar from
# source, then an unsigned app with no updater artifacts. Release builds come from CI.
#
# Usage: build_desktop.sh [--dev] [tauri arguments...]
#   --dev    Runs the app with hot reload instead of building a bundle.
#
# Any other argument goes to the Tauri CLI, for example `--bundles app` or `--debug`.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONSOLE_DIR="$(dirname "$SCRIPT_DIR")"
REPO_ROOT="$(dirname "$CONSOLE_DIR")"
DESKTOP_CONFIG="src-tauri/tauri.desktop.conf.json"

DEV=false
if [ "${1:-}" = "--dev" ]; then
    DEV=true
    shift
fi

"$SCRIPT_DIR/build_core_sidecar.sh"

export VITE_DESKTOP=true

if [ "$DEV" = true ]; then
    # Turbo builds the workspace packages the app depends on.
    (cd "$REPO_ROOT" && pnpm dev:desktop)
    exit 0
fi

echo "Building the workspace packages..."
(cd "$REPO_ROOT" && pnpm exec turbo build --filter "@synnaxlabs/console^...")

echo "Building Synnax Desktop..."
cd "$CONSOLE_DIR"
# The updater artifacts need the release signing key, so a local build leaves them out.
pnpm exec tauri build --no-sign \
    --config "$DESKTOP_CONFIG" \
    --config '{"bundle":{"createUpdaterArtifacts":false}}' \
    "$@"

echo "✓ Bundles are in $CONSOLE_DIR/src-tauri/target/release/bundle"
