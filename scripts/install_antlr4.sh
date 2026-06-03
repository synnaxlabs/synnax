#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Installs a pinned version of antlr4 and exposes it on PATH as `antlr4`.
# Bypasses antlr4-tools so we don't depend on its runtime lookup of the
# "latest version" against the Sonatype search API, which has been
# unreliable.

set -euo pipefail

VERSION="4.13.2"
EXPECTED_SHA256="eae2dfa119a64327444672aff63e9ec35a20180dc5b8090b7a6ab85125df4d76"
INSTALL_DIR="${HOME}/.local/share/antlr4"
BIN_DIR="${HOME}/.local/bin"
JAR_PATH="${INSTALL_DIR}/antlr-${VERSION}-complete.jar"
JAR_URL="https://www.antlr.org/download/antlr-${VERSION}-complete.jar"

mkdir -p "$INSTALL_DIR" "$BIN_DIR"

if [ ! -f "$JAR_PATH" ]; then
    echo "Downloading antlr-${VERSION}-complete.jar"
    curl -fsSL -o "$JAR_PATH" "$JAR_URL"
fi

# sha256sum on Linux, shasum on macOS.
if command -v sha256sum > /dev/null 2>&1; then
    ACTUAL_SHA256=$(sha256sum "$JAR_PATH" | awk '{print $1}')
else
    ACTUAL_SHA256=$(shasum -a 256 "$JAR_PATH" | awk '{print $1}')
fi
if [ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]; then
    echo "Error: SHA-256 mismatch for $JAR_PATH" >&2
    echo "  expected: $EXPECTED_SHA256" >&2
    echo "  actual:   $ACTUAL_SHA256" >&2
    rm -f "$JAR_PATH"
    exit 1
fi

cat > "${BIN_DIR}/antlr4" << EOF
#!/bin/bash
exec java -jar "${JAR_PATH}" "\$@"
EOF
chmod +x "${BIN_DIR}/antlr4"

echo "Installed antlr4 ${VERSION} -> ${BIN_DIR}/antlr4"
