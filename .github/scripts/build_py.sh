#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Usage: build_py.sh <directory>
#
# Builds the wheel and the sdist of each published Python package into <directory>, with
# the internal dependency pins of pin_internal_deps.sh. It restores the pyproject.toml
# files after the build.

set -euo pipefail

out=$(mkdir -p "$1" && cd "$1" && pwd)
repo=$(cd "$(dirname "$0")/../.." && pwd)
packages=(alamos/py x/py freighter/py client/py)

backup=$(mktemp -d)
for package in "${packages[@]}"; do
    mkdir -p "$backup/$package"
    cp "$repo/$package/pyproject.toml" "$backup/$package/pyproject.toml"
done
restore() {
    for package in "${packages[@]}"; do
        cp "$backup/$package/pyproject.toml" "$repo/$package/pyproject.toml"
    done
    rm -rf "$backup"
}
trap restore EXIT

"$repo/.github/scripts/pin_internal_deps.sh"
for package in "${packages[@]}"; do
    uv build --quiet "$repo/$package" --out-dir "$out"
done
