#!/bin/bash

# Script to check for cached artifacts and download them if available
# Usage: check_artifact_cache_unix.sh <platform>
# Platform: linux, macos

set -e

PLATFORM=${1:-linux}

if [ "${PLATFORM}" = "all" ]; then
    ARTIFACT_NAMES=("synnax-core-linux" "synnax-core-windows" "synnax-core-macos")
    echo "Checking for all platform artifacts: ${ARTIFACT_NAMES[*]}"
else
    ARTIFACT_NAMES=("synnax-core-${PLATFORM}")
    echo "Checking for ${PLATFORM} artifact: ${ARTIFACT_NAMES[*]}"
fi

echo "Searching for cached artifacts..."

# First check for exact commit match
EXACT_MATCH_RUN=$(gh run list --workflow="test.integration.yaml" --status="success" --limit=100 --json="databaseId,headSha" | jq -r --arg sha "$(git rev-parse HEAD)" '.[] | select(.headSha == $sha) | .databaseId' | head -1)

if [ -n "${EXACT_MATCH_RUN}" ]; then
    echo "Found exact commit match from run ${EXACT_MATCH_RUN}"
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

            # Check if this run has our artifacts - with debug output
            echo "Running: gh api repos/:owner/:repo/actions/runs/${run_id}/artifacts"
            ARTIFACTS_JSON=$(gh api "repos/:owner/:repo/actions/runs/${run_id}/artifacts")
            echo "Artifacts in run ${run_id}:"
            echo "${ARTIFACTS_JSON}" | jq -r '.artifacts[]? | .name'

            # Check if all required artifacts exist
            ALL_ARTIFACTS_FOUND=true
            for artifact_name in "${ARTIFACT_NAMES[@]}"; do
                ARTIFACT_EXISTS=$(echo "${ARTIFACTS_JSON}" | jq -r --arg name "${artifact_name}" '.artifacts[]? | select(.name == $name) | .name' | head -1)
                if [ -z "${ARTIFACT_EXISTS}" ]; then
                    echo "Artifact ${artifact_name} not found in run ${run_id}"
                    ALL_ARTIFACTS_FOUND=false
                    break
                else
                    echo "Found artifact ${artifact_name} in run ${run_id}"
                fi
            done

            if [ "${ALL_ARTIFACTS_FOUND}" = "true" ]; then
                echo "All required artifacts found in run ${run_id} with SHA ${sha}"
                CACHED_RUN="${run_id}"
                RECENT_SHA="${sha}"
                break
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
            ".github/workflows/test.integration.yaml"
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

    # Download the cached artifacts (we already verified the artifact exists)
    echo "Downloading cached ${ARTIFACT_NAME} from run ${CACHED_RUN}..."

    if gh run download "${CACHED_RUN}" --name "${ARTIFACT_NAME}" --dir ./; then
        echo "Successfully downloaded cached artifacts"

        # Make the binary executable (not needed for Windows .exe files)
        if [ "${PLATFORM}" = "linux" ]; then
            chmod +x ./synnax-v*-linux
        elif [ "${PLATFORM}" = "macos" ]; then
            chmod +x ./synnax-v*-macos
        fi

        # Move to core directory
        if [ "${PLATFORM}" = "windows" ]; then
            mv ./synnax-v*-windows.exe core/ 2>/dev/null || true
        else
            mv ./synnax-v*-${PLATFORM}* core/ 2>/dev/null || true
        fi

    else
        echo "Failed to download artifact ${ARTIFACT_NAME} from run ${CACHED_RUN}, will build from scratch"
        echo "CACHE_HIT=false" >> $GITHUB_OUTPUT
    fi
else
    echo "No cached artifacts available, will build from scratch"
    echo "CACHE_HIT=false" >> $GITHUB_OUTPUT
fi