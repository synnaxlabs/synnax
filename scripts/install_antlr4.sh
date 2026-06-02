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
INSTALL_DIR="${HOME}/.local/share/antlr4"
BIN_DIR="${HOME}/.local/bin"
JAR_PATH="${INSTALL_DIR}/antlr-${VERSION}-complete.jar"
JAR_URL="https://www.antlr.org/download/antlr-${VERSION}-complete.jar"

mkdir -p "$INSTALL_DIR" "$BIN_DIR"

if [ ! -f "$JAR_PATH" ]; then
    echo "Downloading antlr-${VERSION}-complete.jar"
    curl -fsSL -o "$JAR_PATH" "$JAR_URL"
fi

cat > "${BIN_DIR}/antlr4" <<EOF
#!/bin/bash
exec java -jar "${JAR_PATH}" "\$@"
EOF
chmod +x "${BIN_DIR}/antlr4"

echo "Installed antlr4 ${VERSION} -> ${BIN_DIR}/antlr4"
