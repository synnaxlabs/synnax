#!/usr/bin/env bash
set -euo pipefail

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

SCRIPT_DIR="$(
  cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1
  pwd
)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." >/dev/null 2>&1 && pwd)"

if [[ -z "${1:-}" ]]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.51.0"
    exit 1
fi

VERSION="$1"

if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in semver format (e.g., 0.51.0)"
    exit 1
fi

echo "Bumping all versions to $VERSION"
echo "================================"

update_version_file() {
    local version_file="$ROOT_DIR/core/pkg/version/VERSION"
    echo "$VERSION" > "$version_file"
    echo "✅ Updated VERSION file: $version_file"
}

update_python() {
    local pyproject="$1"
    if [[ -f "$pyproject" ]]; then
        sed -i '' "s/^version = \".*\"/version = \"$VERSION\"/" "$pyproject"
        echo "✅ Updated Python: $pyproject"
    else
        echo "❌ File not found: $pyproject"
        return 1
    fi
}

update_node() {
    local pkg_json="$1"
    if [[ -f "$pkg_json" ]]; then
        sed -i '' "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$pkg_json"
        echo "✅ Updated Node: $pkg_json"
    else
        echo "❌ File not found: $pkg_json"
        return 1
    fi
}

update_tauri() {
    local tauri_conf="$1"
    if [[ -f "$tauri_conf" ]]; then
        sed -i '' "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$tauri_conf"
        echo "✅ Updated Tauri: $tauri_conf"
    else
        echo "❌ File not found: $tauri_conf"
        return 1
    fi
}

echo ""
echo "Updating VERSION file..."
update_version_file

echo ""
echo "Updating Python packages..."
PYTHON_DIRS=(
    "$ROOT_DIR/alamos/py"
    "$ROOT_DIR/freighter/py"
    "$ROOT_DIR/client/py"
)
for d in "${PYTHON_DIRS[@]}"; do
    update_python "$d/pyproject.toml"
done

echo ""
echo "Updating Node packages..."
NODE_DIRS=(
    "$ROOT_DIR/alamos/ts"
    "$ROOT_DIR/client/ts"
    "$ROOT_DIR/configs/eslint"
    "$ROOT_DIR/configs/stylelint"
    "$ROOT_DIR/configs/ts"
    "$ROOT_DIR/configs/vite"
    "$ROOT_DIR/drift"
    "$ROOT_DIR/freighter/ts"
    "$ROOT_DIR/pluto"
    "$ROOT_DIR/x/media"
    "$ROOT_DIR/x/ts"
)
for d in "${NODE_DIRS[@]}"; do
    update_node "$d/package.json"
done

echo ""
echo "Updating Tauri config..."
update_tauri "$ROOT_DIR/console/src-tauri/tauri.conf.json"

echo ""
echo "================================"
echo "Version bump complete. Running check_versions.sh to verify..."
echo ""

"$SCRIPT_DIR/check_versions.sh"

CHECK_STATUS=$?

if [[ $CHECK_STATUS -eq 0 ]]; then
    echo ""
    echo "✅ All version checks passed!"
else
    echo ""
    echo "❌ Some version checks failed. Please review the output above."
fi

exit $CHECK_STATUS
