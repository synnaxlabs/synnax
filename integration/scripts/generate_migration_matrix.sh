#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Discovers stable releases and outputs three migration chains:
#   MINIMUM_CHAIN: hardcoded minimum version -> latest
#   MAJOR_CHAIN:   most recent .0 release -> latest
#   PATCH_CHAIN:   most recent release overall -> latest
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

# Latest major: most recent X.Y.0 release
MAJOR=""
for ((i = ${#VERSIONS[@]} - 1; i >= 0; i--)); do
    if [[ "${VERSIONS[$i]}" =~ ^[0-9]+\.[0-9]+\.0$ ]]; then
        MAJOR="${VERSIONS[$i]}"
        break
    fi
done
if [[ -z "$MAJOR" ]]; then
    echo "ERROR: No .0 release found >= $MINIMUM_VERSION" >&2
    exit 1
fi

# Latest patch: most recent release overall
PATCH="${VERSIONS[-1]}"

MINIMUM_CHAIN="${MINIMUM},latest"
MAJOR_CHAIN="${MAJOR},latest"
PATCH_CHAIN="${PATCH},latest"

echo "Minimum chain: $MINIMUM_CHAIN"
echo "Major chain:   $MAJOR_CHAIN"
echo "Patch chain:   $PATCH_CHAIN"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    echo "MINIMUM_CHAIN=$MINIMUM_CHAIN" >> "$GITHUB_OUTPUT"
    echo "MAJOR_CHAIN=$MAJOR_CHAIN" >> "$GITHUB_OUTPUT"
    echo "PATCH_CHAIN=$PATCH_CHAIN" >> "$GITHUB_OUTPUT"
    echo "MINIMUM_VERSION=$MINIMUM" >> "$GITHUB_OUTPUT"
    echo "MAJOR_VERSION=$MAJOR" >> "$GITHUB_OUTPUT"
    echo "PATCH_VERSION=$PATCH" >> "$GITHUB_OUTPUT"
fi
