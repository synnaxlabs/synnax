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

CURRENT_BRANCH=$(git branch --show-current)

RUNS_WITH_ARTIFACTS=$(gh run list --workflow="test.integration.yaml" --branch="${CURRENT_BRANCH}" --status="success" --limit=50 --json="databaseId,headSha")

CACHED_RUN=""
RECENT_SHA=""

for row in $(echo "${RUNS_WITH_ARTIFACTS}" | jq -r '.[] | @base64'); do
    _jq() {
        echo ${row} | base64 --decode | jq -r ${1}
    }

    run_id=$(_jq '.databaseId')
    sha=$(_jq '.headSha')

    if [ -n "${run_id}" ] && [ "${run_id}" != "null" ]; then

        if check_run_has_artifacts "${run_id}" && check_run_built_artifacts "${run_id}"; then
            CACHED_RUN="${run_id}"
            RECENT_SHA="${sha}"
            break
        fi
    fi
done

if [ -n "${CACHED_RUN}" ] && [ -n "${RECENT_SHA}" ]; then
    SAFE_PATHS=(
        "docs/"
        "integration/"
        "*.md"
        "LICENSE"
        ".editorconfig"
    )

    if ! git cat-file -e "${RECENT_SHA}" 2>/dev/null; then
        git fetch --unshallow || git fetch --depth=100 || true
    fi

    # Check again if the SHA exists
    if ! git cat-file -e "${RECENT_SHA}" 2>/dev/null; then
        NEEDS_REBUILD=true
    else
        CHANGED_FILES=$(git diff --name-only "${RECENT_SHA}" HEAD 2>/dev/null)
        if [ $? -ne 0 ]; then
            NEEDS_REBUILD=true
        fi
    fi

    if [ "${NEEDS_REBUILD}" != "true" ]; then
        if [ -z "${CHANGED_FILES}" ]; then
            echo "No changes detected since last successful build"
            NEEDS_REBUILD=false
        else
            NEEDS_REBUILD=false

            while IFS= read -r file; do
                if [ -z "${file}" ]; then
                    continue
                fi

                IS_SAFE=false
                for safe_path in "${SAFE_PATHS[@]}"; do
                    if [[ "${file}" == ${safe_path} ]] || [[ "${file}" == ${safe_path}* ]]; then
                        IS_SAFE=true
                        break
                    fi
                done

                if [ "${IS_SAFE}" = "false" ]; then
                    NEEDS_REBUILD=true
                    break
                fi
            done <<< "${CHANGED_FILES}"
        fi
    fi

    if [ "${NEEDS_REBUILD}" = "false" ]; then
        echo "CACHE_HIT=true" >> ${GITHUB_OUTPUT:-/dev/null}
        echo "CACHED_RUN_ID=${CACHED_RUN}" >> ${GITHUB_OUTPUT:-/dev/null}
        CACHED_RUN="${CACHED_RUN}"
    else
        echo "CACHE_HIT=false" >> ${GITHUB_OUTPUT:-/dev/null}
        CACHED_RUN=""
    fi
else
    echo "CACHE_HIT=false" >> ${GITHUB_OUTPUT:-/dev/null}
    CACHED_RUN=""
fi

if [ -n "${CACHED_RUN}" ]; then
    echo "✅ Cache hit! Found all required artifacts in run ${CACHED_RUN}"
else
    echo "Building artifacts from scratch"
    echo "CACHE_HIT=false" >> ${GITHUB_OUTPUT:-/dev/null}
fi
