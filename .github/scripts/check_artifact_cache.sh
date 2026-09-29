#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Finds a recent integration run whose live artifacts were built from an unchanged
# path set per product, so the build can be skipped or partly reused. Emits SKIP_BUILD,
# REF_RUN_ID, and the DRIVER, CONSOLE, and CORE REF_RUN_ID maps.
# Usage: check_artifact_cache.sh [linux|windows|all]

set -e

PLATFORM=${1:-linux}

if [ "${PLATFORM}" = "all" ]; then
    OS_NAMES=("linux" "windows")
else
    read -r -a OS_NAMES <<< "${PLATFORM}"
fi

CORE_ARTIFACTS=()
DRIVER_ARTIFACTS=()
for os in "${OS_NAMES[@]}"; do
    CORE_ARTIFACTS+=("synnax-core-${os}")
    DRIVER_ARTIFACTS+=("synnax-driver-${os}")
done
# Binary checks download the driver from the reused run, so a full skip needs both.
FULL_ARTIFACTS=("${CORE_ARTIFACTS[@]}" "${DRIVER_ARTIFACTS[@]}")

# A component's artifacts are reusable from a run that matches on its path set.
FILTERS=.github/filters.yaml
# Reads one *_build list from the change map, aliases expanded.
build_paths() {
    yq "explode(.) | .$1 | flatten | .[]" "${FILTERS}" | grep -v '^\.github/'
}
mapfile -t DRIVER_PATHS < <(build_paths driver_build)
mapfile -t CONSOLE_PATHS < <(build_paths console_build)
mapfile -t CORE_PATHS < <(build_paths core_build)

UNION_PATHS=("${DRIVER_PATHS[@]}" "${CONSOLE_PATHS[@]}" "${CORE_PATHS[@]}")

WORKFLOW_FILES=("ci.yaml" "test.integration.yaml" "build.synnax.yaml")

# Flags of the build this run is about to do. A candidate must have used the same
# ones, or its binaries are not interchangeable with the ones this build would make.
DEBUG=${DEBUG:-false}
SIGN_BINARIES=${SIGN_BINARIES:-false}

# The API does not report a run's inputs, so signing is read back from its step.
# Linux drivers are never signed, so that platform has no signing step.
declare -A SIGN_STEP=(
    [windows]="Sign Driver (Windows)"
)
declare -A BUILD_JOB=(
    [linux]="Build (ubuntu-build-bot)"
    [windows]="Build (windows-build-bot)"
)
declare -A DRIVER_STEP=(
    [linux]="Build Driver (Linux)"
    [windows]="Build Driver (Windows)"
)
declare -A BUILD_STEP=(
    [console]="Build Console web assets"
    [core]="Build Core"
)
declare -A TOOLCHAIN=(
    [driver]="bazel-contrib/setup-bazel actions-rust-lang/setup-rust-toolchain"
    [console]="pnpm/setup"
    [core]="actions/setup-go"
)
WORKFLOW=.github/workflows/build.synnax.yaml

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
        local wf
        for wf in "${WORKFLOW_FILES[@]}"; do
            gh api "repos/:owner/:repo/actions/workflows/${wf}/runs?head_sha=${sha}&per_page=20" \
                --jq '.workflow_runs[].id' 2> /dev/null || true
        done | sort -rn > "${file}"
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

# True when the named step ran to success rather than being skipped by its flag.
step_ran() {
    local file=$1
    local name=$2
    if jq -e --arg n "${name}" \
        '.jobs[]?.steps[]? | select(.name == $n and .conclusion == "success")' \
        "${file}" > /dev/null; then
        echo true
    else
        echo false
    fi
}

# Prints the path of the run's build job log for the platform, timestamps and color
# stripped. Fails when the job or its log cannot be read.
job_log() {
    local run_id=$1
    local os=$2
    local job_id
    job_id=$(jq -r --arg n "${BUILD_JOB[${os}]}" \
        '[.jobs[] | select(.name | endswith($n))][0].id // empty' \
        "${CACHE_DIR}/jobs-${run_id}.json")
    if [ -z "${job_id}" ]; then
        log "run ${run_id}: no ${os} build job"
        return 1
    fi
    local file="${CACHE_DIR}/log-${job_id}"
    if [ -f "${file}.err" ]; then
        return 1
    fi
    if [ ! -f "${file}" ]; then
        if ! gh api --allow-escape-sequences \
            "repos/:owner/:repo/actions/jobs/${job_id}/logs" \
            > "${file}.raw" 2> "${file}.err"; then
            rm -f "${file}.raw"
            log "run ${run_id}: ${os} build log unreadable ($(head -1 "${file}.err"))"
            return 1
        fi
        rm -f "${file}.err"
        sed -E 's/^[^ ]+ //; s/\x1b\[[0-9;]*m//g; s/\r$//' "${file}.raw" > "${file}"
    fi
    echo "${file}"
}

# Prints the script of the first logged step containing the marker, plus any of its
# env lines matching the pattern.
step_block() {
    local file=$1
    local marker=$2
    local env_pat=${3:-^$}
    awk -v m="${marker}" -v e="${env_pat}" '
        /^##\[group\]Run / { script = ""; env = ""; hit = 0; s = 1; next }
        s == 1 && /^shell: / { s = 2; next }
        s == 1 { script = script $0 "\n"; if (index($0, m)) hit = 1; next }
        s == 2 && /^##\[endgroup\]/ {
            if (hit) { printf "%s%s", script, env; exit }
            s = 0; next
        }
        s == 2 && $0 ~ e { sub(/^  /, ""); env = env $0 "\n" }
    ' "${file}"
}

# Resolves the debug ternaries in workflow text for this build. The Go tags are pinned
# because a core candidate must already hold the driver and console artifacts.
resolve_expressions() {
    local plain neg
    if [ "${DEBUG}" = true ]; then
        plain='\1'
        neg='\2'
    else
        plain='\2'
        neg='\1'
    fi
    local t="inputs\\.debug *&& *'([^']*)' *\\|\\| *'([^']*)' *\\}\\}"
    sed -E \
        -e "s/\\$\\{\\{ *!${t}/${neg}/g" \
        -e "s/\\$\\{\\{ *${t}/${plain}/g" \
        -e 's/\$\{\{[^}]*GO_BUILD_TAGS *\}\}/-tags console,driver/g' \
        -e 's/\$\{\{[^}]*\}\}//g'
}

expected_script() {
    yq ".jobs.build.steps[] | select(.name == \"$1\") | .run" "${WORKFLOW}" \
        | resolve_expressions
}

expected_env() {
    yq '.jobs.build.env | to_entries | .[] | select(.key | test("^BAZEL_"))
        | .key + ": " + .value' "${WORKFLOW}" | resolve_expressions
}

# Prints the sorted token set of a block with the per run values masked.
normalize() {
    tr -s ' \t\n' '\n' | grep -v -e '^\\$' -e '^$' \
        | sed -E 's/^--define=SYNNAX_DRIVER_VERSION=.*/--define=SYNNAX_DRIVER_VERSION=/
            s/^--remote_cache=.*/--remote_cache=/
            s/(version\.version|gitCommit)=.*/\1=/
            s/^synnax-v.*/synnax-v/' \
        | sort -u
}

# True when the logged build of one component used the same toolchain, script, and
# flags this build would.
component_matches() {
    local comp=$1
    local os=$2
    local log=$3
    local action ref
    for action in ${TOOLCHAIN[${comp}]}; do
        ref=$(yq ".jobs.build.steps[].uses // \"\" | select(test(\"^${action}@\"))" \
            "${WORKFLOW}")
        if [ -z "${ref}" ] || ! grep -qxF "##[group]Run ${ref}" "${log}"; then
            return 1
        fi
    done
    local step env_pat=""
    if [ "${comp}" = driver ]; then
        step=${DRIVER_STEP[${os}]}
        env_pat='^  BAZEL_[A-Z_]*: '
    else
        step=${BUILD_STEP[${comp}]}
    fi
    local script marker expected actual
    script=$(expected_script "${step}")
    marker=$(awk 'NF { l = $0 } END { sub(/^[ \t]+/, "", l); print l }' <<< "${script}")
    expected=${script}
    if [ -n "${env_pat}" ]; then
        expected+=$'\n'$(expected_env)
    fi
    actual=$(step_block "${log}" "${marker}" "${env_pat}")
    if [ -z "${actual}" ]; then
        return 1
    fi
    diff <(normalize <<< "${expected}") <(normalize <<< "${actual}") > /dev/null
}

# True when the run built the component on every named platform like this build would.
# An unreadable run is rejected rather than assumed to match.
run_build_matches() {
    local run_id=$1
    local component=$2
    local os_list=$3
    local file="${CACHE_DIR}/jobs-${run_id}.json"
    if [ ! -f "${file}" ]; then
        if ! gh api "repos/:owner/:repo/actions/runs/${run_id}/jobs?per_page=100" \
            > "${file}" 2> /dev/null; then
            rm -f "${file}"
            log "jobs unreadable for run ${run_id}"
            return 1
        fi
    fi
    local comps=${component}
    if [ "${component}" = all ]; then
        comps="driver console core"
    fi
    local os log comp sign
    for os in ${os_list}; do
        log=$(job_log "${run_id}" "${os}") || return 1
        for comp in ${comps}; do
            if ! component_matches "${comp}" "${os}" "${log}"; then
                log "run ${run_id}: ${comp} ${os} build differs"
                return 1
            fi
            sign=${SIGN_STEP[${os}]}
            if [ "${comp}" = driver ] && [ -n "${sign}" ] \
                && [ "$(step_ran "${file}" "${sign}")" != "${SIGN_BINARIES}" ]; then
                return 1
            fi
        done
    done
    return 0
}

# Prints the newest run with live artifacts for $2 and no diff on the path set.
# Exact-sha lookups cover this branch. The recent-run scan covers other branches.
find_reusable_run() {
    local label=$1
    local artifact_names=$2
    local component=$3
    local os_list=$4
    shift 4

    local sha run_id
    for sha in ${CANDIDATE_SHAS}; do
        if ! paths_clean "${sha}" "$@"; then
            continue
        fi
        for run_id in $(runs_for_sha "${sha}"); do
            if [ "${run_id}" = "${GITHUB_RUN_ID:-}" ]; then
                continue
            fi
            if run_has_artifacts "${run_id}" "${artifact_names}" \
                && run_build_matches "${run_id}" "${component}" "${os_list}"; then
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
        if run_has_artifacts "${run_id}" "${artifact_names}" \
            && run_build_matches "${run_id}" "${component}" "${os_list}"; then
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
    RECENT_RUNS=$(for wf in "${WORKFLOW_FILES[@]}"; do
        gh run list --workflow="${wf}" --limit=25 \
            --json databaseId,headSha --jq '.[] | "\(.databaseId):\(.headSha)"'
    done | sort -t: -k1 -rn)

    local full_run
    full_run=$(find_reusable_run "full" "${FULL_ARTIFACTS[*]}" all "${OS_NAMES[*]}" \
        "${UNION_PATHS[@]}")
    if [ -n "${full_run}" ]; then
        log "✅ Skipping build. Using artifacts from run ${full_run}"
        emit "SKIP_BUILD=true"
        emit "REF_RUN_ID=${full_run}"
        return 0
    fi

    # Each platform resolves on its own, so a windows-only run can serve the windows
    # half of a build that also does Ubuntu.
    local driver_ids="{}" console_ids="{}" core_ids="{}" os run
    for os in "${OS_NAMES[@]}"; do
        run=$(find_reusable_run "driver ${os}" "synnax-driver-${os}" driver "${os}" \
            "${DRIVER_PATHS[@]}")
        driver_ids=$(jq -c --arg os "${os}" --arg run "${run}" '.[$os] = $run' \
            <<< "${driver_ids}")
        run=$(find_reusable_run "console ${os}" "synnax-console-assets-${os}" console \
            "${os}" "${CONSOLE_PATHS[@]}")
        console_ids=$(jq -c --arg os "${os}" --arg run "${run}" '.[$os] = $run' \
            <<< "${console_ids}")
        run=$(find_reusable_run "core ${os}" \
            "synnax-core-${os} synnax-driver-${os} synnax-console-assets-${os}" \
            all "${os}" "${UNION_PATHS[@]}")
        core_ids=$(jq -c --arg os "${os}" --arg run "${run}" '.[$os] = $run' \
            <<< "${core_ids}")
    done
    emit "SKIP_BUILD=false"
    emit "REF_RUN_ID=${GITHUB_RUN_ID:-}"
    emit "DRIVER_REF_RUN_ID=${driver_ids}"
    emit "CONSOLE_REF_RUN_ID=${console_ids}"
    emit "CORE_REF_RUN_ID=${core_ids}"
}

main
