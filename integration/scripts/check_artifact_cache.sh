#!/bin/bash

# Script to check for cached artifacts and download them if available
# Usage: check_artifact_cache_unix.sh <platform>
# Platform: linux, macos

set -e

PLATFORM=${1:-linux}

if [ "${PLATFORM}" = "all" ]; then
    # Only check for enabled platforms (macOS is currently disabled)
    ARTIFACT_NAMES=("synnax-core-linux" "synnax-core-windows")
    echo "Checking for enabled platform artifacts: ${ARTIFACT_NAMES[*]}"
else
    ARTIFACT_NAMES=("synnax-core-${PLATFORM}")
    echo "Checking for ${PLATFORM} artifact: ${ARTIFACT_NAMES[*]}"
fi

echo "Searching for cached artifacts..."

# Helper function to check if a run has all required artifacts
check_run_has_artifacts() {
    local run_id=$1
    local artifacts_json=$(gh api "repos/:owner/:repo/actions/runs/${run_id}/artifacts")

    for artifact_name in "${ARTIFACT_NAMES[@]}"; do
        local artifact_exists=$(echo "${artifacts_json}" | jq -r --arg name "${artifact_name}" '.artifacts[]? | select(.name == $name) | .name' | head -1)
        if [ -z "${artifact_exists}" ]; then
            return 1  # Artifact not found
        fi
    done
    return 0  # All artifacts found
}

# Helper function to check if a run actually built artifacts (has upload steps)
check_run_built_artifacts() {
    local run_id=$1
    local run_jobs=$(gh api "repos/:owner/:repo/actions/runs/${run_id}/jobs")
    local built_artifacts=$(echo "${run_jobs}" | jq -r '.jobs[].steps[]? | select(.name | contains("Upload") and contains("Artifact")) | select(.conclusion == "success") | .name' | head -1)
    [ -n "${built_artifacts}" ]
}

# Get current branch name
CURRENT_BRANCH=$(git branch --show-current)
echo "Current branch: ${CURRENT_BRANCH}"

# Search for the most recent successful run that actually built artifacts
echo "Searching for most recent run that built artifacts..."

RUNS_WITH_ARTIFACTS=$(gh run list --workflow="test.integration.yaml" --branch="${CURRENT_BRANCH}" --status="success" --limit=50 --json="databaseId,headSha")

CACHED_RUN=""
RECENT_SHA=""

# Process runs one by one to find the last one that built artifacts
for row in $(echo "${RUNS_WITH_ARTIFACTS}" | jq -r '.[] | @base64'); do
    _jq() {
        echo ${row} | base64 --decode | jq -r ${1}
    }

    run_id=$(_jq '.databaseId')
    sha=$(_jq '.headSha')

    if [ -n "${run_id}" ] && [ "${run_id}" != "null" ]; then
        echo "Checking run ${run_id}..."

        if check_run_has_artifacts "${run_id}" && check_run_built_artifacts "${run_id}"; then
            echo "✅ Found last successful build: run ${run_id} with SHA ${sha}"
            CACHED_RUN="${run_id}"
            RECENT_SHA="${sha}"
            break
        else
            echo "Run ${run_id} used cached artifacts, continuing search..."
        fi
    fi
done

if [ -n "${CACHED_RUN}" ] && [ -n "${RECENT_SHA}" ]; then
    echo "Comparing changes since last successful build (SHA ${RECENT_SHA})..."

    # Check if changes are only in safe directories that don't require rebuild
    SAFE_PATHS=(
        "docs/"
        "integration/"
        "*.md"
        "LICENSE"
        ".editorconfig"
    )

    # Get all changed files since last successful build
    CHANGED_FILES=$(git diff --name-only "${RECENT_SHA}" HEAD 2>/dev/null || echo "")

    if [ -z "${CHANGED_FILES}" ]; then
        echo "No changes detected since last build"
        NEEDS_REBUILD=false
    else
        echo "Changed files since last build:"
        echo "${CHANGED_FILES}"
        NEEDS_REBUILD=false

        # Check each changed file against safe paths
        while IFS= read -r file; do
            if [ -z "${file}" ]; then
                continue
            fi

            IS_SAFE=false
            for safe_path in "${SAFE_PATHS[@]}"; do
                if [[ "${file}" == ${safe_path} ]] || [[ "${file}" == ${safe_path}* ]]; then
                    echo "  ${file} - SAFE (matches ${safe_path})"
                    IS_SAFE=true
                    break
                fi
            done

            if [ "${IS_SAFE}" = "false" ]; then
                echo "  ${file} - REBUILD REQUIRED (not in safe paths)"
                NEEDS_REBUILD=true
                break
            fi
        done <<< "${CHANGED_FILES}"
    fi

    if [ "${NEEDS_REBUILD}" = "false" ]; then
        echo "All changes are in safe paths, using cached artifacts from run ${CACHED_RUN}"
        echo "CACHE_HIT=true" >> ${GITHUB_OUTPUT:-/dev/null}
        echo "CACHED_RUN_ID=${CACHED_RUN}" >> ${GITHUB_OUTPUT:-/dev/null}
        CACHED_RUN="${CACHED_RUN}"
    else
        echo "Rebuild required due to changes outside safe paths"
        echo "CACHE_HIT=false" >> ${GITHUB_OUTPUT:-/dev/null}
        CACHED_RUN=""
    fi
else
    echo "No recent successful builds with artifacts found"
    echo "CACHE_HIT=false" >> ${GITHUB_OUTPUT:-/dev/null}
    CACHED_RUN=""
fi

# Final output
if [ -n "${CACHED_RUN}" ]; then
    echo "✅ Cache hit! Found all required artifacts in run ${CACHED_RUN}"
    echo "Will skip build and use cached artifacts"
else
    echo "❌ No cached artifacts available, will build from scratch"
    echo "CACHE_HIT=false" >> ${GITHUB_OUTPUT:-/dev/null}
fi