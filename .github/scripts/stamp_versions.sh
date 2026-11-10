#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Writes a version into the npm (ts) or PyPI (py) package manifests. A build runs it
# before it compiles the packages, so the packages and the clients' compatibility checks
# carry the release version. Usage: stamp_versions.sh <ts|py> <version>

set -euo pipefail

LANGUAGE=${1:?usage: stamp_versions.sh <ts|py> <version>}
VERSION=${2:?usage: stamp_versions.sh <ts|py> <version>}
if ! [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
    echo "malformed version: $VERSION" >&2
    exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." > /dev/null 2>&1 && pwd)"

stamp_ts() {
    node -e '
        const fs = require("fs");
        const [path, version] = process.argv.slice(1);
        const pkg = JSON.parse(fs.readFileSync(path, "utf8"));
        pkg.version = version;
        fs.writeFileSync(path, JSON.stringify(pkg, null, 2) + "\n");
    ' "$1" "$VERSION"
}

stamp_py() {
    sed -i.bak -E "s/^version = \".*\"$/version = \"$VERSION\"/" "$1"
    rm "$1.bak"
    grep -q "^version = \"$VERSION\"$" "$1" || {
        echo "no version line in $1" >&2
        exit 1
    }
}

case "$LANGUAGE" in
    ts) manifest=package.json ;;
    py) manifest=pyproject.toml ;;
    *)
        echo "unknown language: $LANGUAGE (expected ts or py)" >&2
        exit 1
        ;;
esac

for package in alamos x freighter client; do
    "stamp_$LANGUAGE" "$ROOT_DIR/$package/$LANGUAGE/$manifest"
done
echo "stamped every $LANGUAGE package manifest with $VERSION"
