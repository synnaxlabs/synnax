#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Pins the version constraints of internal workspace dependencies (alamos,
# synnax-freighter, synnax-x) in each Python package's pyproject.toml to
# ">=<dep-version>,<<dep-next-minor>>".
#
# Each constraint is derived from the depended-on package's own version, not the
# depending package's. This keeps the published metadata satisfiable even when versions
# diverge: if synnax is 0.55.2 but synnax-x is still 0.55.1, synnax pins
# "synnax-x>=0.55.1,<0.56.0" (the release that actually exists) rather than demanding a
# nonexistent 0.55.2.
#
# uv does not rewrite workspace dependency constraints at publish time, and the
# pyproject.toml entries are intentionally left unconstrained for local development.
# This script is run by the deploy pipeline immediately before `uv build` so the
# published wheel metadata pins matching releases (see SY-4246). It is not meant to be
# committed: it mutates the working tree of an ephemeral release checkout.

set -euo pipefail

SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1
    pwd
)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." > /dev/null 2>&1 && pwd)"

INTERNAL_PACKAGES=("alamos" "synnax-freighter" "synnax-x")

PACKAGE_DIRS=(
    "$ROOT_DIR/alamos/py"
    "$ROOT_DIR/freighter/py"
    "$ROOT_DIR/client/py"
    "$ROOT_DIR/x/py"
)

# pyproject.toml that defines each internal package.
pyproject_for() {
    case "$1" in
        alamos) echo "$ROOT_DIR/alamos/py/pyproject.toml" ;;
        synnax-freighter) echo "$ROOT_DIR/freighter/py/pyproject.toml" ;;
        synnax-x) echo "$ROOT_DIR/x/py/pyproject.toml" ;;
        *)
            echo "❌ Unknown internal package: $1" >&2
            return 1
            ;;
    esac
}

version_of() {
    local pyproject="$1" version
    if [[ ! -f "$pyproject" ]]; then
        echo "❌ File not found: $pyproject" >&2
        return 1
    fi
    version="$(grep -m1 '^version[[:space:]]*=' "$pyproject" | sed "s/.*=[[:space:]]*['\"]//;s/['\"].*//")"
    if [[ -z "$version" ]]; then
        echo "❌ Could not read version from $pyproject" >&2
        return 1
    fi
    echo "$version"
}

# Maps a version to a ">=<version>,<<next-minor>>" constraint, e.g. 0.55.1 ->
# >=0.55.1,<0.56.0.
constraint_for() {
    local version="$1" major minor
    major="${version%%.*}"
    minor="${version#*.}"
    minor="${minor%%.*}"
    echo ">=${version},<${major}.$((minor + 1)).0"
}

# Fails if any internal package is still listed as an unconstrained dependency. This
# guards against a silent sed no-op (e.g. an unexpected pyproject.toml format) leaving a
# release with the very unconstrained metadata this script exists to prevent.
verify_pinned() {
    local pyproject="$1" name
    for name in "${INTERNAL_PACKAGES[@]}"; do
        if grep -Eq "^[[:space:]]*\"${name}\"" "$pyproject"; then
            echo "❌ ${pyproject}: '${name}' is still unconstrained after pinning" >&2
            return 1
        fi
    done
    return 0
}

SED_ARGS=()
for name in "${INTERNAL_PACKAGES[@]}"; do
    dep_pyproject="$(pyproject_for "$name")"
    dep_version="$(version_of "$dep_pyproject")"
    dep_constraint="$(constraint_for "$dep_version")"
    SED_ARGS+=(-e "s|^([[:space:]]*)\"${name}([<>=!~][^\"]*)?\"(,?)[[:space:]]*\$|\1\"${name}${dep_constraint}\"\3|")
    echo "→ ${name}${dep_constraint}"
done

for d in "${PACKAGE_DIRS[@]}"; do
    pyproject="$d/pyproject.toml"
    if [[ ! -f "$pyproject" ]]; then
        echo "❌ File not found: $pyproject" >&2
        exit 1
    fi
    tmp="$(mktemp)"
    sed -E "${SED_ARGS[@]}" "$pyproject" > "$tmp"
    mv "$tmp" "$pyproject"
    if ! verify_pinned "$pyproject"; then
        exit 1
    fi
    echo "✅ Pinned internal deps in $pyproject"
done
