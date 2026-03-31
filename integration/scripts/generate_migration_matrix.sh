#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Discovers the most recent stable release and outputs a migration chain:
#   previous_version,latest
#
# Usage:
#   generate_migration_matrix.sh --platform <linux|windows>

set -euo pipefail

MINIMUM_VERSION="0.50.0"

# --- Argument parsing ---

PLATFORM=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        *)
            echo "Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

if [[ -z "$PLATFORM" ]]; then
    echo "Usage: generate_migration_matrix.sh --platform <linux|windows>" >&2
    exit 1
fi

case "$PLATFORM" in
    linux) ASSET_SUFFIX="-linux" ;;
    windows) ASSET_SUFFIX="-windows.exe" ;;
    *)
        echo "Unsupported platform: $PLATFORM" >&2
        exit 1
        ;;
esac

# --- Discover the most recent stable release ---

version_gte() {
    [[ "$(printf '%s\n%s' "$1" "$2" | sort -V | head -n1)" == "$2" ]]
}

echo "Discovering releases >= $MINIMUM_VERSION for platform $PLATFORM..."

RELEASES=$(gh release list --repo synnaxlabs/synnax --json tagName --limit 200 --jq '.[].tagName')

VERSIONS=()
for tag in $RELEASES; do
    if [[ "$tag" =~ ^synnax-v([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
        ver="${BASH_REMATCH[1]}"
        if version_gte "$ver" "$MINIMUM_VERSION"; then
            asset_name="synnax-v${ver}${ASSET_SUFFIX}"
            if gh release view "$tag" --repo synnaxlabs/synnax --json assets --jq '.assets[].name' 2> /dev/null | grep -qx "$asset_name"; then
                VERSIONS+=("$ver")
            else
                echo "  Skipping $ver - no $asset_name asset found"
            fi
        fi
    fi
done

IFS=$'\n' VERSIONS=($(printf '%s\n' "${VERSIONS[@]}" | sort -V))
unset IFS

if [[ ${#VERSIONS[@]} -lt 1 ]]; then
    echo "ERROR: No versions found >= $MINIMUM_VERSION with $PLATFORM binaries" >&2
    exit 1
fi

PREVIOUS="${VERSIONS[-1]}"
CHAIN="${PREVIOUS},latest"

echo "Migration chain: $CHAIN"
echo "MIGRATION_CHAIN=$CHAIN" >> "$GITHUB_OUTPUT"
