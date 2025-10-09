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
    local current_branch=$(git branch --show-current)
    local workflow_file="${GITHUB_WORKFLOW:-test.integration.yaml}"
    local runs_json=$(gh run list --workflow="${workflow_file}" --branch="${current_branch}" --status="success" --limit=25 --json="databaseId,headSha")

    for row in $(echo "${runs_json}" | jq -r '.[] | @base64'); do
        local run_id=$(echo ${row} | base64 --decode | jq -r '.databaseId')
        local sha=$(echo ${row} | base64 --decode | jq -r '.headSha')

        if [ -n "${run_id}" ] && [ "${run_id}" != "null" ]; then
            if check_run_has_artifacts "${run_id}" && check_run_built_artifacts "${run_id}"; then
                echo "${run_id}:${sha}"
                return 0
            fi
        fi
    done
    return 1
}

check_if_rebuild_needed() {
    local sha=$1

    if ! git cat-file -e "${sha}" 2> /dev/null; then
        git fetch --quiet --unshallow 2> /dev/null || git fetch --quiet --depth=25 2> /dev/null || true
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
