#!/bin/bash

# Script to check for cached artifacts and download them if available
# Usage: check_artifact_cache_unix.sh <platform>
# Platform: linux, macos

set -e

PLATFORM=${1:-linux}
ARTIFACT_NAME="synnax-core-${PLATFORM}"

echo "Searching for cached artifacts..."

# First check for exact commit match
EXACT_MATCH_RUN=$(gh run list --workflow="test.integration.yaml" --status="success" --limit=100 --json="databaseId,headSha" | jq -r --arg sha "$(git rev-parse HEAD)" '.[] | select(.headSha == $sha) | .databaseId' | head -1)

if [ -n "${EXACT_MATCH_RUN}" ]; then
    echo "Found exact commit match from run ${EXACT_MATCH_RUN}"
    echo "CACHE_HIT=true" >> $GITHUB_OUTPUT
    echo "CACHED_RUN_ID=${EXACT_MATCH_RUN}" >> $GITHUB_OUTPUT
    CACHED_RUN="${EXACT_MATCH_RUN}"
else
    # Check for recent successful run and compare file changes
    echo "No exact match, checking if recent artifacts can be reused..."
    RECENT_RUN=$(gh run list --workflow="test.integration.yaml" --status="success" --limit=10 --json="databaseId,headSha" | jq -r '.[0] | .databaseId + " " + .headSha')

    if [ -n "${RECENT_RUN}" ]; then
        RECENT_RUN_ID=$(echo "${RECENT_RUN}" | cut -d' ' -f1)
        RECENT_SHA=$(echo "${RECENT_RUN}" | cut -d' ' -f2)

        echo "Found recent successful run ${RECENT_RUN_ID} with SHA ${RECENT_SHA}"
        echo "Comparing changes since last successful build (${RECENT_SHA})..."

        # Check if changes are only in safe directories that don't require rebuild
        SAFE_PATHS=(
            "docs/"
            "integration/test/"
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
            echo "No rebuild required, using cached artifacts from run ${RECENT_RUN_ID}"
            echo "CACHE_HIT=true" >> $GITHUB_OUTPUT
            echo "CACHED_RUN_ID=${RECENT_RUN_ID}" >> $GITHUB_OUTPUT
            CACHED_RUN="${RECENT_RUN_ID}"
        else
            echo "Rebuild required due to source changes"
            echo "CACHE_HIT=false" >> $GITHUB_OUTPUT
            CACHED_RUN=""
        fi
    else
        echo "No recent successful runs found"
        echo "CACHE_HIT=false" >> $GITHUB_OUTPUT
        CACHED_RUN=""
    fi
fi

if [ -n "${CACHED_RUN}" ]; then

    # Download the cached artifacts
    echo "Downloading cached ${ARTIFACT_NAME} from run ${CACHED_RUN}..."

    # First check if the artifact exists
    ARTIFACT_EXISTS=$(gh run view "${CACHED_RUN}" --json "artifacts" | jq -r --arg name "${ARTIFACT_NAME}" '.artifacts[] | select(.name == $name) | .name' | head -1)

    if [ -n "${ARTIFACT_EXISTS}" ]; then
        echo "Found artifact ${ARTIFACT_NAME} in run ${CACHED_RUN}"
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
        echo "Artifact ${ARTIFACT_NAME} not found in run ${CACHED_RUN}, will build from scratch"
        echo "CACHE_HIT=false" >> $GITHUB_OUTPUT
    fi
else
    echo "No cached artifacts found for current commit, will build from scratch"
    echo "CACHE_HIT=false" >> $GITHUB_OUTPUT
fi