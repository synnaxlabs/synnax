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

PLATFORM=${1:-linux}

if [ "${PLATFORM}" = "all" ]; then
    ARTIFACT_NAMES=("synnax-core-linux" "synnax-core-windows")
else
    ARTIFACT_NAMES=("synnax-core-${PLATFORM}")
fi

REBUILD_PATHS=(
    ".bazeliskrc"
    ".bazelrc"
    ".github/workflows/**"
    "alamos/go/**"
    "alamos/ts/**"
    "aspen/**"
    "cesium/**"
    "client/cpp/**"
    "client/ts/**"
    "configs/ts/**"
    "configs/vite/**"
    "console/**"
    "core/**"
    "drift/**"
    "driver/**"
    "freighter/cpp/**"
    "freighter/go/**"
    "freighter/ts/**"
    "go.work"
    "go.work.sum"
    "MODULE.bazel"
    "MODULE.bazel.lock"
    "package.json"
    "pluto/**"
    "pnpm-lock.yaml"
    "pnpm-workspace.yaml"
    "turbo.json"
    "vendor/mbedtls/**"
    "vendor/open62541/**"
    "x/cpp/**"
    "x/go/**"
    "x/ts/**"
)

check_run_has_artifacts() {
    local run_id=$1
    local artifacts_json=$(gh api "repos/:owner/:repo/actions/runs/${run_id}/artifacts")

    for artifact_name in "${ARTIFACT_NAMES[@]}"; do
        local artifact_exists=$(echo "${artifacts_json}" | jq -r --arg name "${artifact_name}" '.artifacts[]? | select(.name == $name) | .name' | head -1)
        if [ -z "${artifact_exists}" ]; then
            return 1
        fi
    done
    return 0
}

check_run_built_artifacts() {
    local run_id=$1
    local run_jobs=$(gh api "repos/:owner/:repo/actions/runs/${run_id}/jobs")
    local built_artifacts=$(echo "${run_jobs}" | jq -r '.jobs[].steps[]? | select(.name | contains("Upload") and contains("Artifact")) | select(.conclusion == "success") | .name' | head -1)
    [ -n "${built_artifacts}" ]
}

find_cached_run() {
    # Use GITHUB_HEAD_REF for PRs, fallback to git branch for direct pushes
    local current_branch="${GITHUB_HEAD_REF:-$(git branch --show-current)}"
    if [ -z "${current_branch}" ]; then
        # Fallback for detached HEAD without GITHUB_HEAD_REF
        current_branch="${GITHUB_BASE_REF:-${GITHUB_REF_NAME:-rc}}"
    fi
    # GITHUB_WORKFLOW contains the workflow name, not filename - hardcode the filename
    local workflow_file="test.integration.yaml"

    echo "DEBUG: GITHUB_HEAD_REF=${GITHUB_HEAD_REF}" >&2
    echo "DEBUG: GITHUB_BASE_REF=${GITHUB_BASE_REF}" >&2
    echo "DEBUG: GITHUB_REF_NAME=${GITHUB_REF_NAME}" >&2
    echo "DEBUG: current_branch=${current_branch}" >&2
    echo "DEBUG: workflow_file=${workflow_file}" >&2

    local runs_json=$(gh run list --workflow="${workflow_file}" --branch="${current_branch}" --limit=25 --json="databaseId,headSha")
    echo "DEBUG: Found $(echo "${runs_json}" | jq 'length') runs" >&2

    for row in $(echo "${runs_json}" | jq -r '.[] | @base64'); do
        local run_id=$(echo ${row} | base64 --decode | jq -r '.databaseId')
        local sha=$(echo ${row} | base64 --decode | jq -r '.headSha')

        if [ -n "${run_id}" ] && [ "${run_id}" != "null" ]; then
            echo "DEBUG: Checking run ${run_id} (sha: ${sha:0:8})" >&2
            if check_run_has_artifacts "${run_id}"; then
                echo "DEBUG:   - has_artifacts: YES" >&2
                if check_run_built_artifacts "${run_id}"; then
                    echo "DEBUG:   - built_artifacts: YES" >&2
                    echo "${run_id}:${sha}"
                    return 0
                else
                    echo "DEBUG:   - built_artifacts: NO" >&2
                fi
            else
                echo "DEBUG:   - has_artifacts: NO" >&2
            fi
        fi
    done
    return 1
}

check_if_rebuild_needed() {
    local sha=$1

    echo "DEBUG: check_if_rebuild_needed for sha=${sha:0:8}" >&2

    if ! git cat-file -e "${sha}" 2> /dev/null; then
        echo "DEBUG:   - SHA not found locally, fetching..." >&2
        git fetch --quiet --unshallow 2> /dev/null || git fetch --quiet --depth=25 2> /dev/null || true
    fi

    if ! git cat-file -e "${sha}" 2> /dev/null; then
        echo "DEBUG:   - SHA still not found after fetch, rebuild needed" >&2
        return 0
    fi

    local changed_files=$(git diff --name-only "${sha}" HEAD 2> /dev/null)
    if [ $? -ne 0 ] || [ -z "${changed_files}" ]; then
        echo "DEBUG:   - No changed files or diff failed, no rebuild needed" >&2
        return 1
    fi

    echo "DEBUG:   - Found $(echo "${changed_files}" | wc -l) changed files" >&2

    while IFS= read -r file; do
        if [ -z "${file}" ]; then
            continue
        fi

        local needs_rebuild=false
        for rebuild_path in "${REBUILD_PATHS[@]}"; do
            if [[ "${file}" == ${rebuild_path} ]] || [[ "${file}" == ${rebuild_path}* ]]; then
                needs_rebuild=true
                echo "DEBUG:   - File '${file}' matches rebuild path '${rebuild_path}'" >&2
                break
            fi
        done

        if [ "${needs_rebuild}" = "true" ]; then
            return 0
        fi
    done <<< "${changed_files}"

    echo "DEBUG:   - No rebuild paths matched, no rebuild needed" >&2
    return 1
}

main() {
    local cache_result=$(find_cached_run)

    if [ -z "${cache_result}" ]; then
        echo "CACHE_HIT=false" >> ${GITHUB_OUTPUT:-/dev/null}
        return 0
    fi

    local cached_run_id="${cache_result%:*}"
    local cached_sha="${cache_result#*:}"

    if check_if_rebuild_needed "${cached_sha}"; then
        echo "CACHE_HIT=false" >> ${GITHUB_OUTPUT:-/dev/null}
        return 0
    fi

    echo "CACHE_HIT=true" >> ${GITHUB_OUTPUT:-/dev/null}
    echo "CACHED_RUN_ID=${cached_run_id}" >> ${GITHUB_OUTPUT:-/dev/null}
    echo "✅ Cache hit! Found all required artifacts in run ${cached_run_id} (${cached_sha:0:8})"
}

main
