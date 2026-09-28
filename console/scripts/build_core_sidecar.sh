#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Builds the Core from source and installs it as the Synnax Desktop sidecar for local
# development. The Core embeds the Driver when core/pkg/driver/assets holds a Driver
# binary; release builds take the Core from CI instead.
#
# Usage: build_core_sidecar.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
CORE_DIR="$REPO_ROOT/core"
BINARIES_DIR="$REPO_ROOT/console/src-tauri/binaries"

# Tauri resolves a sidecar by the Rust host triple.
TRIPLE="$(rustc -vV | sed -n 's/^host: //p')"
EXT=""
case "$TRIPLE" in *windows*) EXT=".exe" ;; esac

TAGS=""
if [ -f "$CORE_DIR/pkg/driver/assets/driver$EXT" ]; then
    TAGS="driver"
else
    echo "No Driver binary in core/pkg/driver/assets: building a Core without a Driver."
fi

mkdir -p "$BINARIES_DIR"
OUT="$BINARIES_DIR/synnax-core-$TRIPLE$EXT"
echo "Building the Core sidecar..."
(cd "$CORE_DIR" && go build -tags "$TAGS" -o "$OUT" .)
echo "✓ Core sidecar installed to $OUT"
