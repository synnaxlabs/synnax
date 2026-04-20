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
# is compiled with -tags driver,console. Release builds replace these placeholders
# with real binaries/bundles; this script never overwrites existing file content,
# so it is safe to run in any environment.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CORE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# --- Driver assets ---
DRIVER_ASSETS_DIR="$CORE_DIR/pkg/driver/assets"
mkdir -p "$DRIVER_ASSETS_DIR"
touch "$DRIVER_ASSETS_DIR/driver" "$DRIVER_ASSETS_DIR/driver.exe"
echo "Driver asset placeholders ready at $DRIVER_ASSETS_DIR"

# --- Console assets ---
CONSOLE_DIST_DIR="$CORE_DIR/pkg/console/dist"
mkdir -p "$CONSOLE_DIST_DIR"

if [ ! -f "$CONSOLE_DIST_DIR/index.html" ]; then
    cat > "$CONSOLE_DIST_DIR/index.html" << 'HTMLEOF'
<!doctype html>
<html lang="en">
<head><meta charset="UTF-8" /><title>Synnax Console</title></head>
<body><div id="root"></div></body>
</html>
HTMLEOF
fi

if [ ! -f "$CONSOLE_DIST_DIR/favicon.svg" ]; then
    cat > "$CONSOLE_DIST_DIR/favicon.svg" << 'SVGEOF'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><rect width="1" height="1"/></svg>
SVGEOF
fi

echo "Console asset placeholders ready at $CONSOLE_DIST_DIR"
