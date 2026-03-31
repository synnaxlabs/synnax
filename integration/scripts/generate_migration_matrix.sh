#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# TODO: Delete this script. Simplify migration logic to only test previous release
# version against current build.

set -euo pipefail

MINIMUM_VERSION="0.50.0"
RANDOM_CHAIN_COUNT=2
MAX_INTERMEDIATES=2

# --- Argument parsing ---

PLATFORM=""
SEED="${GITHUB_RUN_NUMBER:-$$}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --seed)
            SEED="$2"
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

# Map platform to binary asset suffix
case "$PLATFORM" in
    linux) ASSET_SUFFIX="-linux" ;;
    windows) ASSET_SUFFIX="-windows.exe" ;;
    *)
        echo "Unsupported platform: $PLATFORM" >&2
        exit 1
        ;;
esac

# --- Discover Available Versions ---

version_gte() {
    # Returns 0 (true) if $1 >= $2, using sort -V
    [[ "$(printf '%s\n%s' "$1" "$2" | sort -V | head -n1)" == "$2" ]]
}

echo "Discovering releases >= $MINIMUM_VERSION for platform $PLATFORM..."

RELEASES=$(gh release list --repo synnaxlabs/synnax --json tagName --limit 200 --jq '.[].tagName')

VERSIONS=()
for tag in $RELEASES; do
    # Only synnax-v* tags, no -rc suffix
    if [[ "$tag" =~ ^synnax-v([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
        ver="${BASH_REMATCH[1]}"
        if version_gte "$ver" "$MINIMUM_VERSION"; then
            # Verify binary asset exists for this platform
            asset_name="synnax-v${ver}${ASSET_SUFFIX}"
            if gh release view "$tag" --repo synnaxlabs/synnax --json assets --jq '.assets[].name' 2> /dev/null | grep -qx "$asset_name"; then
                VERSIONS+=("$ver")
            else
                echo "  Skipping $ver - no $asset_name asset found"
            fi
        fi
    fi
done

# Sort versions
IFS=$'\n' VERSIONS=($(printf '%s\n' "${VERSIONS[@]}" | sort -V))
unset IFS

if [[ ${#VERSIONS[@]} -lt 1 ]]; then
    echo "ERROR: No versions found >= $MINIMUM_VERSION with $PLATFORM binaries" >&2
    exit 1
fi

echo "Found ${#VERSIONS[@]} versions: ${VERSIONS[*]}"

MINIMUM="${VERSIONS[0]}"
PREVIOUS="${VERSIONS[-1]}"

# --- Build deterministic anchor chains ---

CHAINS=()
CHAIN_NAMES=()

# Anchor 1: minimum → latest
CHAINS+=("${MINIMUM},latest")
CHAIN_NAMES+=("minimum-to-latest")
echo "Anchor chain: $MINIMUM -> latest"

# Anchor 2: previous → latest (skip if same as minimum)
if [[ "$PREVIOUS" != "$MINIMUM" ]]; then
    CHAINS+=("${PREVIOUS},latest")
    CHAIN_NAMES+=("previous-to-latest")
    echo "Anchor chain: $PREVIOUS -> latest"
fi

# --- Build random chains ---

RANDOM=$SEED
echo "Random seed: $SEED"

for i in $(seq 1 "$RANDOM_CHAIN_COUNT"); do
    # Pick random start (not the last version - that would be same as previous-to-latest)
    max_start_idx=$((${#VERSIONS[@]} - 1))
    if [[ $max_start_idx -lt 1 ]]; then
        break
    fi
    start_idx=$((RANDOM % max_start_idx))
    start_ver="${VERSIONS[$start_idx]}"

    # Pick 0 to MAX_INTERMEDIATES random intermediates between start and end
    num_intermediates=$((RANDOM % (MAX_INTERMEDIATES + 1)))
    intermediates=()

    # Collect versions between start and previous (exclusive)
    candidates=()
    for v in "${VERSIONS[@]}"; do
        if version_gte "$v" "$start_ver" && [[ "$v" != "$start_ver" ]]; then
            candidates+=("$v")
        fi
    done

    if [[ ${#candidates[@]} -gt 0 && $num_intermediates -gt 0 ]]; then
        # Shuffle candidates using seed
        IFS=$'\n' shuffled=($(printf '%s\n' "${candidates[@]}" | shuf --random-source=<(echo "$SEED$i") 2> /dev/null || printf '%s\n' "${candidates[@]}" | sort -R))
        unset IFS
        for j in $(seq 0 $((num_intermediates - 1))); do
            if [[ $j -lt ${#shuffled[@]} ]]; then
                intermediates+=("${shuffled[$j]}")
            fi
        done
        # Sort intermediates
        if [[ ${#intermediates[@]} -gt 0 ]]; then
            IFS=$'\n' intermediates=($(printf '%s\n' "${intermediates[@]}" | sort -V))
            unset IFS
        fi
    fi

    # Build chain string
    chain="$start_ver"
    for mid in "${intermediates[@]}"; do
        chain="$chain,$mid"
    done
    chain="$chain,latest"

    CHAINS+=("$chain")
    CHAIN_NAMES+=("random-${i}")
    echo "Random chain $i: $chain"
done

# --- Generate matrix JSON ---

MATRIX_ENTRIES=""
for idx in "${!CHAINS[@]}"; do
    chain="${CHAINS[$idx]}"
    name="${CHAIN_NAMES[$idx]}"
    for test_type in inplace export_import; do
        entry_name="${name}-${test_type}"
        if [[ -n "$MATRIX_ENTRIES" ]]; then
            MATRIX_ENTRIES="$MATRIX_ENTRIES,"
        fi
        MATRIX_ENTRIES="$MATRIX_ENTRIES{\"name\":\"${entry_name}\",\"chain\":\"${chain}\",\"test_type\":\"${test_type}\"}"
    done
done

MATRIX="{\"include\":[${MATRIX_ENTRIES}]}"

echo ""
echo "Generated matrix:"
echo "$MATRIX" | python3 -m json.tool 2> /dev/null || echo "$MATRIX"
echo ""

echo "MIGRATION_MATRIX=$MATRIX" >> "$GITHUB_OUTPUT"
