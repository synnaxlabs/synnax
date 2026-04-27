#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Creates placeholder assets so that //go:embed directives succeed when the code
# is compiled with -tags driver. Release builds replace these placeholders with
# real binaries; this script never overwrites existing file content, so it is safe
# to run in any environment.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CORE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# --- Driver assets ---
DRIVER_ASSETS_DIR="$CORE_DIR/pkg/driver/assets"
mkdir -p "$DRIVER_ASSETS_DIR"
touch "$DRIVER_ASSETS_DIR/driver" "$DRIVER_ASSETS_DIR/driver.exe"
echo "Driver asset placeholders ready at $DRIVER_ASSETS_DIR"
