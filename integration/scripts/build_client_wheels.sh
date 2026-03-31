#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Builds Python client wheels for a specific version.
#
# For "latest", builds from the current checkout. For any other version, checks
# out the corresponding git tag (synnax-v{VERSION}), builds, then restores the
# previous checkout.
#
# Usage:
#   build_client_wheels.sh <version> <output_dir>
#
# Examples:
#   build_client_wheels.sh latest wheels/latest
#   build_client_wheels.sh 0.53.0 wheels/0.53.0

set -euo pipefail

VERSION="${1:?Usage: build_client_wheels.sh <version> <output_dir>}"
OUT_DIR="${2:?Usage: build_client_wheels.sh <version> <output_dir>}"
PACKAGES=("alamos/py" "freighter/py" "x/py" "client/py")

mkdir -p "$OUT_DIR"

if [ "$VERSION" != "latest" ]; then
    echo "Checking out synnax-v${VERSION}..."
    git checkout "synnax-v${VERSION}"
fi

for pkg in "${PACKAGES[@]}"; do
    echo "Building $pkg..."
    uv build "$pkg" --wheel -o "$OUT_DIR"
done

if [ "$VERSION" != "latest" ]; then
    git checkout -
fi

echo "Wheels for ${VERSION}:"
ls -la "$OUT_DIR"/*.whl
