#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Writes a version into every npm and PyPI package manifest. A build runs it before it
# compiles the packages, so the packages and the clients' compatibility checks carry the
# release version. Usage: stamp_versions.sh <version>

set -euo pipefail

VERSION=${1:?usage: stamp_versions.sh <version>}
if ! [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
    echo "malformed version: $VERSION" >&2
    exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." > /dev/null 2>&1 && pwd)"

for package in alamos x freighter client; do
    node -e '
        const fs = require("fs");
        const [path, version] = process.argv.slice(1);
        const pkg = JSON.parse(fs.readFileSync(path, "utf8"));
        pkg.version = version;
        fs.writeFileSync(path, JSON.stringify(pkg, null, 2) + "\n");
    ' "$ROOT_DIR/$package/ts/package.json" "$VERSION"
    pyproject="$ROOT_DIR/$package/py/pyproject.toml"
    sed -i.bak -E "s/^version = \".*\"$/version = \"$VERSION\"/" "$pyproject"
    rm "$pyproject.bak"
    grep -q "^version = \"$VERSION\"$" "$pyproject" || {
        echo "no version line in $pyproject" >&2
        exit 1
    }
done
echo "stamped every package manifest with $VERSION"
