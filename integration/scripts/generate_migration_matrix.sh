#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Outputs a GitHub Actions matrix for migration tests.
#
# If CUSTOM_CHAIN is set (e.g. "0.52.0, 0.53.1, 0.54.2"), outputs a single
# matrix entry with that chain. Otherwise, discovers stable releases and
# builds major and patch chains.
#
# Outputs (via GITHUB_OUTPUT):
#   MATRIX   — JSON for strategy.matrix via fromJSON()
#   VERSIONS — comma-separated unique versions that need client wheels built
#
# Usage:
#   CUSTOM_CHAIN="0.52.0,0.53.0" generate_migration_matrix.sh
#   generate_migration_matrix.sh

set -euo pipefail

output_matrix() {
    # Build matrix JSON with os × chain combinations
    local json='{"include":['
    local first=true
    for entry in "$@"; do
        local name="${entry%%=*}"
        local chain="${entry#*=}"
        local version="${chain%%,*}"
        for os in ubuntu-latest windows-latest; do
            if [[ "$first" == true ]]; then first=false; else json+=','; fi
            json+="{\"chain_name\":\"$name\",\"os\":\"$os\",\"chain\":\"$chain\",\"version\":\"$version\"}"
        done
    done
    json+=']}'
    echo "$json"
}

collect_versions() {
    local seen=""
    for entry in "$@"; do
        local chain="${entry#*=}"
        IFS=',' read -ra parts <<< "$chain"
        for v in "${parts[@]}"; do
            if ! echo "$seen" | grep -qw "$v"; then
                seen="$seen $v"
            fi
        done
    done
    echo "${seen## }" | tr ' ' ','
}

# Custom chain: normalize and output directly
if [[ -n "${CUSTOM_CHAIN:-}" ]]; then
    CHAIN=$(echo "$CUSTOM_CHAIN" | tr -d ' ')
    echo "Custom chain: $CHAIN"

    MATRIX=$(output_matrix "custom=$CHAIN")
    VERSIONS=$(collect_versions "custom=$CHAIN")

    echo "Matrix: $MATRIX"
    echo "Versions: $VERSIONS"

    if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
        echo "MATRIX=$MATRIX" >> "$GITHUB_OUTPUT"
        echo "VERSIONS=$VERSIONS" >> "$GITHUB_OUTPUT"
    fi
    exit 0
fi

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

PATCH="${VERSIONS[${#VERSIONS[@]} - 1]}"

# Build chain entries (chain_name=version_chain)
# Minimum chain disabled while latest major == minimum
ENTRIES=("major=${MAJOR},latest" "patch=${PATCH},latest")

echo "Major chain: ${MAJOR},latest"
echo "Patch chain: ${PATCH},latest"

MATRIX=$(output_matrix "${ENTRIES[@]}")
WHEEL_VERSIONS=$(collect_versions "${ENTRIES[@]}")

echo "Matrix: $MATRIX"
echo "Versions: $WHEEL_VERSIONS"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    echo "MATRIX=$MATRIX" >> "$GITHUB_OUTPUT"
    echo "VERSIONS=$WHEEL_VERSIONS" >> "$GITHUB_OUTPUT"
fi
