#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Ensures core/pkg/driver/assets/{driver,driver.exe} exist so that the
# //go:embed directives in core/pkg/driver/{unix,windows}_enabled.go succeed
# when the code is compiled with -tags driver. Release builds replace these
# placeholders with the real driver binary; touch never overwrites file
# content, so this script is safe to run in any environment.

set -euo pipefail

SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1
    pwd
)"
REPO_ROOT="$(
    cd "$SCRIPT_DIR/.." > /dev/null 2>&1
    pwd
)"
ASSETS_DIR="$REPO_ROOT/core/pkg/driver/assets"

mkdir -p "$ASSETS_DIR"
touch "$ASSETS_DIR/driver" "$ASSETS_DIR/driver.exe"

echo "Driver asset placeholders ready at $ASSETS_DIR"
