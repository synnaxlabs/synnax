#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Fails unless every package manifest shares one minor that is the Core's latest stable
# minor or the next one. Usage: check_versions.sh [root], root defaulting to the repo.

set -euo pipefail

SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1
    pwd
)"
ROOT_DIR="$(cd "${1:-$SCRIPT_DIR/..}" > /dev/null 2>&1 && pwd)"

fail() {
    echo "Error: $*" >&2
    exit 1
}

require_dir() {
    [[ -d "$1" ]] || fail "directory not found: $1"
}

require_file() {
    [[ -f "$1" ]] || fail "file not found: $1"
}

# Extract major.minor (X.Y) from a semver-ish string.
major_minor() {
    echo "$1" | cut -d '.' -f1-2
}

# The train rule: every package shares one minor, and that minor is the Core's latest
# stable minor or the next one.
get_allowed_mm() {
    local core
    core="$(git -C "$ROOT_DIR" tag --list 'core/v*' \
        | sed -nE 's#^core/v([0-9]+\.[0-9]+\.[0-9]+)$#\1#p' \
        | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)"
    [[ -n "$core" ]] || fail "no stable core tag found; fetch tags first"
    local major minor
    IFS=. read -r major minor _ <<< "$core"
    CORE_MM="$major.$minor"
    NEXT_MM="$major.$((minor + 1))"
    echo "Allowed versions: ${CORE_MM}.x or ${NEXT_MM}.x (from core/v$core)"
}

read_node_mm() {
    local pkg_json="$1"
    require_file "$pkg_json"

    local full
    full="$(grep -m1 '"version"[[:space:]]*:' "$pkg_json" | cut -d '"' -f4)"
    [[ -n "$full" ]] || fail "could not read version from $pkg_json"
    major_minor "$full"
}

read_python_mm() {
    local pyproject="$1"
    require_file "$pyproject"

    local full
    full="$(grep -m1 '^version[[:space:]]*=' "$pyproject" | cut -d '"' -f2)"
    [[ -n "$full" ]] || fail "could not read version from $pyproject"
    major_minor "$full"
}

check_match() {
    local label="$1"
    local found_mm="$2"
    local expected_mm="$3"
    local source="$4"

    if [[ "$found_mm" != "$expected_mm" ]]; then
        echo "❌ $label version mismatch: found ${found_mm}.x, expected ${expected_mm}.x ($source)" >&2
        return 1
    fi
    echo "✅ $label version ok (${found_mm}.x)"
    return 0
}

main() {
    get_allowed_mm

    local ok=true
    local expected=""

    local PYTHON_DIRS=(
        "$ROOT_DIR/alamos/py"
        "$ROOT_DIR/freighter/py"
        "$ROOT_DIR/client/py"
        "$ROOT_DIR/x/py"
    )

    for d in "${PYTHON_DIRS[@]}"; do
        require_dir "$d"
        local f="$d/pyproject.toml"
        local found
        found="$(read_python_mm "$f")"
        expected="${expected:-$found}"
        if ! check_match "Python ($d)" "$found" "$expected" "$f"; then ok=false; fi
    done

    local NODE_DIRS=(
        "$ROOT_DIR/alamos/ts"
        "$ROOT_DIR/arc/ts"
        "$ROOT_DIR/client/ts"
        "$ROOT_DIR/drift"
        "$ROOT_DIR/freighter/ts"
        "$ROOT_DIR/pluto"
        "$ROOT_DIR/x/media"
        "$ROOT_DIR/x/ts"
    )

    for d in "${NODE_DIRS[@]}"; do
        require_dir "$d"
        local f="$d/package.json"
        local found
        found="$(read_node_mm "$f")"
        expected="${expected:-$found}"
        if ! check_match "Node ($d)" "$found" "$expected" "$f"; then ok=false; fi
    done

    local cpp_version="$ROOT_DIR/client/cpp/version/VERSION"
    require_file "$cpp_version"
    local found
    found="$(major_minor "$(tr -d '[:space:]' < "$cpp_version")")"
    expected="${expected:-$found}"
    if ! check_match "C++ (client/cpp)" "$found" "$expected" "$cpp_version"; then
        ok=false
    fi

    if [[ "$expected" != "$CORE_MM" && "$expected" != "$NEXT_MM" ]]; then
        echo "❌ packages are on ${expected}.x, not ${CORE_MM}.x or ${NEXT_MM}.x" >&2
        ok=false
    fi

    if [[ "$ok" == true ]]; then
        echo "All packages share ${expected}.x."
        exit 0
    else
        echo "Version check failed."
        exit 1
    fi
}

main
