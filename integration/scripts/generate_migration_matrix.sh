#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Discovers stable releases and outputs two migration chains:
#   MINIMUM_CHAIN: minimum_version -> latest (build artifact)
#   PREVIOUS_CHAIN: most_recent_release -> latest (build artifact)
#
# Usage:
#   generate_migration_matrix.sh

set -euo pipefail

MINIMUM_VERSION="0.53.0"

version_gte() {
    [[ "$(printf '%s\n%s' "$1" "$2" | sort -V | head -n1)" == "$2" ]]
}

echo "Discovering releases >= $MINIMUM_VERSION..."

RELEASES=$(gh release list --repo synnaxlabs/synnax --json tagName --limit 200 --jq '.[].tagName')

VERSIONS=()
for tag in $RELEASES; do
    if [[ "$tag" =~ ^synnax-v([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
        ver="${BASH_REMATCH[1]}"
        if version_gte "$ver" "$MINIMUM_VERSION"; then
            VERSIONS+=("$ver")
        fi
    fi
done

IFS=$'\n' VERSIONS=($(printf '%s\n' "${VERSIONS[@]}" | sort -V))
unset IFS

if [[ ${#VERSIONS[@]} -lt 1 ]]; then
    echo "ERROR: No versions found >= $MINIMUM_VERSION" >&2
    exit 1
fi

echo "Found ${#VERSIONS[@]} versions: ${VERSIONS[*]}"

MINIMUM="${VERSIONS[0]}"
PREVIOUS="${VERSIONS[-1]}"

MINIMUM_CHAIN="${MINIMUM},latest"
PREVIOUS_CHAIN="${PREVIOUS},latest"

echo "Minimum chain: $MINIMUM_CHAIN (from $MINIMUM)"
echo "Previous chain: $PREVIOUS_CHAIN (from $PREVIOUS)"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    echo "MINIMUM_CHAIN=$MINIMUM_CHAIN" >> "$GITHUB_OUTPUT"
    echo "PREVIOUS_CHAIN=$PREVIOUS_CHAIN" >> "$GITHUB_OUTPUT"
    echo "MINIMUM_VERSION=$MINIMUM" >> "$GITHUB_OUTPUT"
    echo "PREVIOUS_VERSION=$PREVIOUS" >> "$GITHUB_OUTPUT"
fi
