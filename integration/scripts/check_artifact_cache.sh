#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
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
    OS_NAMES=("linux" "windows")
else
    OS_NAMES=("${PLATFORM}")
fi

CORE_ARTIFACTS=()
DRIVER_ARTIFACTS=()
CONSOLE_ARTIFACTS=()
for os in "${OS_NAMES[@]}"; do
    CORE_ARTIFACTS+=("synnax-core-${os}")
    DRIVER_ARTIFACTS+=("synnax-driver-${os}")
    CONSOLE_ARTIFACTS+=("synnax-console-assets-${os}")
done
# Binary checks download the driver from the reused run, so a full skip needs both.
FULL_ARTIFACTS=("${CORE_ARTIFACTS[@]}" "${DRIVER_ARTIFACTS[@]}")

# A component's artifacts are reusable from a run that matches on its path set.
# build.synnax.yaml is in every set because a build definition change rebuilds all.
DRIVER_PATHS=(
    ".bazelignore"
    ".bazeliskrc"
    ".bazelrc"
    ".github/workflows/build.synnax.yaml"
    ".gitmodules"
    "arc/cpp/**"
    "client/cpp/**"
    "driver/**"
    "freighter/cpp/**"
    "MODULE.bazel"
    "MODULE.bazel.lock"
    "vendor/**"
    "x/cpp/**"
)

CONSOLE_PATHS=(
    ".github/workflows/build.synnax.yaml"
    "alamos/ts/**"
    "arc/ts/**"
    "client/ts/**"
    "configs/ts/**"
    "configs/vite/**"
    "console/**"
    "drift/**"
    "freighter/ts/**"
    "package.json"
    "pluto/**"
    "pnpm-lock.yaml"
    "pnpm-workspace.yaml"
    "turbo.json"
    "x/media/**"
    "x/ts/**"
)

CORE_PATHS=(
    ".github/workflows/build.synnax.yaml"
    "alamos/go/**"
    "arc/go/**"
    "aspen/**"
    "cesium/**"
    "core/**"
    "freighter/go/**"
    "x/go/**"
)

UNION_PATHS=("${DRIVER_PATHS[@]}" "${CONSOLE_PATHS[@]}" "${CORE_PATHS[@]}")

WORKFLOW_FILE="test.integration.yaml"

CACHE_DIR=$(mktemp -d)
trap 'rm -rf "${CACHE_DIR}"' EXIT

log() {
    echo "[cache] $1" >&2
}

emit() {
    echo "$1" >> "${GITHUB_OUTPUT:-/dev/null}"
}

ensure_history() {
    if [ "$(git rev-parse --is-shallow-repository)" = "true" ]; then
        git fetch --quiet --deepen=30 2> /dev/null || true
    fi
}

# Fetches the commit by sha when the checkout does not already contain it.
ensure_commit() {
    local sha=$1
    if [ -f "${CACHE_DIR}/commit-ok-${sha}" ]; then
        return 0
    fi
    if [ -f "${CACHE_DIR}/commit-missing-${sha}" ]; then
        return 1
    fi
    if git cat-file -e "${sha}^{commit}" 2> /dev/null \
        || { git fetch --quiet --depth=1 origin "${sha}" 2> /dev/null \
            && git cat-file -e "${sha}^{commit}" 2> /dev/null; }; then
        touch "${CACHE_DIR}/commit-ok-${sha}"
        return 0
    fi
    touch "${CACHE_DIR}/commit-missing-${sha}"
    return 1
}

# True when none of the given paths differ between the commit and COMPARE_REF.
paths_clean() {
    local sha=$1
    shift
    local specs=()
    local p
    for p in "$@"; do
        specs+=("${p%'/**'}")
    done
    git diff --quiet "${sha}" "${COMPARE_REF}" -- "${specs[@]}" 2> /dev/null
}

runs_for_sha() {
    local sha=$1
    local file="${CACHE_DIR}/runs-${sha}"
    if [ ! -f "${file}" ]; then
        gh api "repos/:owner/:repo/actions/workflows/${WORKFLOW_FILE}/runs?head_sha=${sha}&per_page=20" \
            --jq '.workflow_runs[].id' > "${file}" 2> /dev/null || true
    fi
    cat "${file}"
}

# Expired artifacts still appear in the API listing but cannot be downloaded.
run_has_artifacts() {
    local run_id=$1
    local names=$2
    local file="${CACHE_DIR}/artifacts-${run_id}.json"
    if [ ! -f "${file}" ]; then
        gh api "repos/:owner/:repo/actions/runs/${run_id}/artifacts?per_page=100" \
            > "${file}" 2> /dev/null || echo '{}' > "${file}"
    fi
    local name
    for name in ${names}; do
        local found=$(jq -r --arg name "${name}" \
            '.artifacts[]? | select(.name == $name and .expired == false) | .name' \
            "${file}" | head -1)
        if [ -z "${found}" ]; then
            return 1
        fi
    done
    return 0
}

# Prints the newest run with live artifacts for $2 and no diff on the path set.
# Exact-sha lookups cover this branch. The recent-run scan covers other branches.
find_reusable_run() {
    local label=$1
    local artifact_names=$2
    shift 2

    local sha run_id
    for sha in ${CANDIDATE_SHAS}; do
        if ! paths_clean "${sha}" "$@"; then
            continue
        fi
        for run_id in $(runs_for_sha "${sha}"); do
            if [ "${run_id}" = "${GITHUB_RUN_ID:-}" ]; then
                continue
            fi
            if run_has_artifacts "${run_id}" "${artifact_names}"; then
                log "${label}: reusing run ${run_id} (${sha:0:8}, exact sha)"
                echo "${run_id}"
                return 0
            fi
        done
    done

    local row
    for row in ${RECENT_RUNS}; do
        run_id="${row%%:*}"
        sha="${row#*:}"
        if [ "${run_id}" = "${GITHUB_RUN_ID:-}" ]; then
            continue
        fi
        if ! ensure_commit "${sha}"; then
            continue
        fi
        if ! paths_clean "${sha}" "$@"; then
            continue
        fi
        if run_has_artifacts "${run_id}" "${artifact_names}"; then
            log "${label}: reusing run ${run_id} (${sha:0:8}, clean diff)"
            echo "${run_id}"
            return 0
        fi
    done
    log "${label}: no reusable run found"
}

main() {
    # A dispatch can force reuse of a specific run and skip the search entirely.
    if [ "${SKIP_BUILD:-false}" = "true" ]; then
        if [ -n "${REF_RUN_ID:-}" ]; then
            log "Skipping build with artifacts from run ${REF_RUN_ID}"
            emit "SKIP_BUILD=true"
            emit "REF_RUN_ID=${REF_RUN_ID}"
            return 0
        fi
        log "Empty REF_RUN_ID. Searching for cached artifacts instead."
    fi

    COMPARE_REF="${GITHUB_HEAD_SHA:-HEAD}"
    ensure_history
    ensure_commit "${COMPARE_REF}" || COMPARE_REF="HEAD"

    # Candidates older than the 7-day artifact retention cannot hit.
    CANDIDATE_SHAS=$(git rev-list --since=7.days --max-count=30 "${COMPARE_REF}" \
        2> /dev/null || true)
    RECENT_RUNS=$(gh run list --workflow="${WORKFLOW_FILE}" --limit=25 \
        --json databaseId,headSha --jq '.[] | "\(.databaseId):\(.headSha)"')

    local full_run
    full_run=$(find_reusable_run "full" "${FULL_ARTIFACTS[*]}" "${UNION_PATHS[@]}")
    if [ -n "${full_run}" ]; then
        log "✅ Skipping build. Using artifacts from run ${full_run}"
        emit "SKIP_BUILD=true"
        emit "REF_RUN_ID=${full_run}"
        return 0
    fi

    local driver_run console_run
    driver_run=$(find_reusable_run "driver" "${DRIVER_ARTIFACTS[*]}" \
        "${DRIVER_PATHS[@]}")
    console_run=$(find_reusable_run "console" "${CONSOLE_ARTIFACTS[*]}" \
        "${CONSOLE_PATHS[@]}")
    emit "SKIP_BUILD=false"
    emit "REF_RUN_ID=${GITHUB_RUN_ID:-}"
    emit "DRIVER_REF_RUN_ID=${driver_run}"
    emit "CONSOLE_REF_RUN_ID=${console_run}"
}

main
