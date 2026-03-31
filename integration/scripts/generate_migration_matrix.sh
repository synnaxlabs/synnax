#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Discovers stable releases and outputs a migration chain:
#   minimum,intermediate,latest
#
# The intermediate is the midpoint between the minimum and most recent release.
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

if [[ ${#VERSIONS[@]} -ge 3 ]]; then
    MID_IDX=$((${#VERSIONS[@]} / 2))
    INTERMEDIATE="${VERSIONS[$MID_IDX]}"
    CHAIN="${MINIMUM},${INTERMEDIATE},latest"
elif [[ ${#VERSIONS[@]} -eq 2 ]]; then
    CHAIN="${MINIMUM},${VERSIONS[1]},latest"
else
    CHAIN="${MINIMUM},latest"
fi

echo "Migration chain: $CHAIN"
echo "MIGRATION_CHAIN=$CHAIN" >> "$GITHUB_OUTPUT"
