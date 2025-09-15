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

# First check for exact commit match
EXACT_MATCH_RUNS=$(gh run list --workflow="test.integration.yaml" --status="success" --limit=100 --json="databaseId,headSha" | jq -r --arg sha "$(git rev-parse HEAD)" '.[] | select(.headSha == $sha) | .databaseId')

EXACT_MATCH_RUN=""
if [ -n "${EXACT_MATCH_RUNS}" ]; then
    echo "Found exact commit matches, checking which one actually built artifacts..."
    for run_id in ${EXACT_MATCH_RUNS}; do
        echo "Checking if run ${run_id} built artifacts..."

        if check_run_has_artifacts "${run_id}" && check_run_built_artifacts "${run_id}"; then
            echo "✅ Exact match run ${run_id} actually built artifacts"
            EXACT_MATCH_RUN="${run_id}"
            break
        else
            echo "Exact match run ${run_id} used cached artifacts, continuing search..."
        fi
    done
fi

if [ -n "${EXACT_MATCH_RUN}" ]; then
    echo "Found exact commit match that built artifacts: run ${EXACT_MATCH_RUN}"
    echo "CACHE_HIT=true" >> $GITHUB_OUTPUT
    echo "CACHED_RUN_ID=${EXACT_MATCH_RUN}" >> $GITHUB_OUTPUT
    CACHED_RUN="${EXACT_MATCH_RUN}"
else
    # Check for recent successful run that has artifacts and compare file changes
    echo "No exact match, searching for most recent run with artifacts..."

    # Get current branch name
    CURRENT_BRANCH=$(git branch --show-current)
    echo "Current branch: ${CURRENT_BRANCH}"

    # First, show all runs from test.integration.yaml for debugging
    echo "All recent runs from test.integration.yaml workflow on branch ${CURRENT_BRANCH}:"
    gh run list --workflow="test.integration.yaml" --branch="${CURRENT_BRANCH}" --limit=20 --json="databaseId,headSha,status,conclusion,displayTitle" | jq -r '.[] | "\(.databaseId) \(.status) \(.conclusion) \(.headSha[0:8]) \(.displayTitle)"'

    echo ""
    echo "Successful runs only on branch ${CURRENT_BRANCH}:"
    gh run list --workflow="test.integration.yaml" --branch="${CURRENT_BRANCH}" --status="success" --limit=50 --json="databaseId,headSha,conclusion,displayTitle" | jq -r '.[] | "\(.databaseId) \(.conclusion) \(.headSha[0:8]) \(.displayTitle)"'

    # Search through recent successful runs to find one that has our artifact
    RUNS_WITH_ARTIFACTS=$(gh run list --workflow="test.integration.yaml" --branch="${CURRENT_BRANCH}" --status="success" --limit=50 --json="databaseId,headSha")

    echo ""
    echo "Raw JSON response from gh run list:"
    echo "${RUNS_WITH_ARTIFACTS}"

    CACHED_RUN=""
    RECENT_SHA=""

    echo ""
    echo "Now checking each successful run for required artifacts..."

    # Process runs one by one to avoid subshell issues
    for row in $(echo "${RUNS_WITH_ARTIFACTS}" | jq -r '.[] | @base64'); do
        _jq() {
            echo ${row} | base64 --decode | jq -r ${1}
        }

        run_id=$(_jq '.databaseId')
        sha=$(_jq '.headSha')

        if [ -n "${run_id}" ] && [ "${run_id}" != "null" ]; then
            echo "Checking run ${run_id} for required artifacts..."

            if check_run_has_artifacts "${run_id}"; then
                echo "All required artifacts found in run ${run_id}, checking if run actually built them..."

                if check_run_built_artifacts "${run_id}"; then
                    echo "✅ Run ${run_id} actually built artifacts (found upload steps)"
                    CACHED_RUN="${run_id}"
                    RECENT_SHA="${sha}"
                    break
                else
                    echo "Run ${run_id} used cached artifacts, skipping to find original build"
                fi
            else
                echo "Not all required artifacts found in run ${run_id}"
            fi
        fi
    done

    if [ -n "${CACHED_RUN}" ] && [ -n "${RECENT_SHA}" ]; then
        echo "Found run ${CACHED_RUN} with artifacts, comparing changes since SHA ${RECENT_SHA}..."

        # Check if changes are only in safe directories that don't require rebuild
        SAFE_PATHS=(
            "docs/"
            "integration/"
            "*.md"
            "LICENSE"
            ".git*"
            ".editorconfig"
        )

        # Get all changed files
        CHANGED_FILES=$(git diff --name-only "${RECENT_SHA}" HEAD 2>/dev/null || echo "")

        if [ -z "${CHANGED_FILES}" ]; then
            echo "No changes detected"
            NEEDS_REBUILD=false
        else
            echo "Changed files: ${CHANGED_FILES}"
            NEEDS_REBUILD=false

            # Check each changed file
            while IFS= read -r file; do
                IS_SAFE=false
                for safe_path in "${SAFE_PATHS[@]}"; do
                    if [[ "${file}" == ${safe_path} ]] || [[ "${file}" == ${safe_path}* ]]; then
                        IS_SAFE=true
                        break
                    fi
                done

                if [ "${IS_SAFE}" = "false" ]; then
                    echo "Change detected in ${file} - rebuild required"
                    NEEDS_REBUILD=true
                    break
                fi
            done <<< "${CHANGED_FILES}"
        fi

        if [ "${NEEDS_REBUILD}" = "false" ]; then
            echo "No rebuild required, using cached artifacts from run ${CACHED_RUN}"
            echo "CACHE_HIT=true" >> $GITHUB_OUTPUT
            echo "CACHED_RUN_ID=${CACHED_RUN}" >> $GITHUB_OUTPUT
        else
            echo "Rebuild required due to source changes"
            echo "CACHE_HIT=false" >> $GITHUB_OUTPUT
            CACHED_RUN=""
        fi
    else
        echo "No recent successful runs with artifacts found"
        echo "CACHE_HIT=false" >> $GITHUB_OUTPUT
        CACHED_RUN=""
    fi
fi

if [ -n "${CACHED_RUN}" ]; then
    echo "✅ Cache hit! Found all required artifacts in run ${CACHED_RUN}"
    echo "Will skip build and use cached artifacts"
else
    echo "❌ No cached artifacts available, will build from scratch"
    echo "CACHE_HIT=false" >> $GITHUB_OUTPUT
fi