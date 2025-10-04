#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

set -e

IMAGE_NAME="ghcr.io/synnaxlabs/synnax"

REBUILD_PATHS=(
    ".github/workflows/test.driver.yaml"
    ".github/workflows/test.client.yaml"
    ".github/workflows/test.migration.yaml"
    "alamos/go/**"
    "aspen/**"
    "cesium/**"
    "core/**"
    "Earthfile"
    "freighter/go/**"
    "go.work"
    "go.work.sum"
    "x/go/**"
)

check_image_exists() {
    local sha=$1
    gh api \
        -H "Accept: application/vnd.github+json" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        "/orgs/synnaxlabs/packages/container/synnax/versions" \
        --jq ".[] | select(.metadata.container.tags[] | contains(\"${sha}\")) | .metadata.container.tags[]" \
        2>/dev/null | grep -q "^${sha}$"
}

check_if_rebuild_needed() {
    local sha=$1

    if ! git cat-file -e "${sha}" 2> /dev/null; then
        git fetch --quiet --unshallow 2> /dev/null || git fetch --quiet --depth=50 2> /dev/null || true
    fi

    if ! git cat-file -e "${sha}" 2> /dev/null; then
        return 0
    fi

    local changed_files=$(git diff --name-only "${sha}" HEAD 2> /dev/null)
    if [ $? -ne 0 ] || [ -z "${changed_files}" ]; then
        return 1
    fi

    while IFS= read -r file; do
        if [ -z "${file}" ]; then
            continue
        fi

        local needs_rebuild=false
        for rebuild_path in "${REBUILD_PATHS[@]}"; do
            if [[ "${file}" == ${rebuild_path} ]] || [[ "${file}" == ${rebuild_path}* ]]; then
                needs_rebuild=true
                break
            fi
        done

        if [ "${needs_rebuild}" = "true" ]; then
            return 0
        fi
    done <<< "${changed_files}"

    return 1
}

find_cached_image() {
    local current_sha=$(git rev-parse HEAD)

    # Check if image exists for current SHA first
    if check_image_exists "${current_sha}"; then
        echo "${current_sha}"
        return 0
    fi

    # Walk back through git history to find the most recent commit with an image
    local commit_count=0
    local max_commits=50

    for sha in $(git log --format=%H -n ${max_commits}); do
        commit_count=$((commit_count + 1))

        # Check if we need to rebuild since this commit
        if ! check_if_rebuild_needed "${sha}"; then
            # No rebuild needed, check if image exists
            if check_image_exists "${sha}"; then
                echo "✅ Found cached image at ${sha:0:8} ($commit_count commits back)" >&2
                echo "${sha}"
                return 0
            fi
        fi
    done

    return 1
}

main() {
    local cached_sha=$(find_cached_image)

    if [ -z "${cached_sha}" ]; then
        echo "CACHE_HIT=false" >> ${GITHUB_OUTPUT:-/dev/null}
        echo "IMAGE_SHA=${GITHUB_SHA}" >> ${GITHUB_OUTPUT:-/dev/null}
        echo "⚠️  No cached image found, will build new image" >&2
        return 0
    fi

    echo "CACHE_HIT=true" >> ${GITHUB_OUTPUT:-/dev/null}
    echo "IMAGE_SHA=${cached_sha}" >> ${GITHUB_OUTPUT:-/dev/null}
}

main
