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
# Reads one *_build list from the change map as Git pathspecs, aliases expanded. Git
# reads a directory as every file below it.
build_paths() {
    yq "explode(.) | .$1 | flatten | .[]" "${FILTERS}" | grep -v '^\.github/' \
        | sed 's|/\*\*$||'
}
mapfile -t DRIVER_PATHS < <(build_paths driver_build)
mapfile -t CONSOLE_PATHS < <(build_paths console_build)
mapfile -t CORE_PATHS < <(build_paths core_build)

# shellcheck disable=SC2034 # Read through a nameref in paths_clean and record_clean.
UNION_PATHS=("${DRIVER_PATHS[@]}" "${CONSOLE_PATHS[@]}" "${CORE_PATHS[@]}")

# Flags of the build this run is about to do. A candidate must have used the same
# ones, or its binaries are not interchangeable with the ones this build would make.
DEBUG=${DEBUG:-false}
SIGN_BINARIES=${SIGN_BINARIES:-false}
FLAGS_ENABLED=${FLAGS_ENABLED:-false}

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
declare -A ARTIFACT_RUNS
# 1 or 0 per set:sha, for whether the set's paths match COMPARE_REF.
declare -A CLEAN

CACHE_DIR=$(mktemp -d)
trap 'rm -rf "${CACHE_DIR}"' EXIT

log() {
    echo "[cache] $1" >&2
}

emit() {
    echo "$1" >> "${GITHUB_OUTPUT:-/dev/null}"
}

# Only the history of COMPARE_REF gives candidates, so no other ref is deepened.
ensure_history() {
    if [ "$(git rev-parse --is-shallow-repository)" = "true" ]; then
        git fetch --quiet --deepen=30 origin "${COMPARE_REF}" 2> /dev/null || true
    fi
}

# Fetches the listed commits the checkout lacks in one request. GitHub at times refuses
# a commit that no branch reaches, which fails the whole request, so the refused commit
# is dropped and the request sent again. ensure_commit tries the dropped ones later.
fetch_missing() {
    local todo="${CACHE_DIR}/missing"
    local err sha
    git cat-file --batch-check < "$1" | awk '$2 == "missing" { print $1 }' > "${todo}"
    while [ -s "${todo}" ]; do
        if err=$(xargs git fetch --quiet --depth=1 origin < "${todo}" 2>&1); then
            return 0
        fi
        sha=$(sed -nE 's/.*not our ref ([0-9a-f]{40}).*/\1/p' <<< "${err}" | head -1)
        if [ -z "${sha}" ] || ! grep -qxF "${sha}" "${todo}"; then
            return 0
        fi
        grep -vxF "${sha}" "${todo}" > "${todo}.next" || true
        mv "${todo}.next" "${todo}"
    done
}

# Writes the listed commits the checkout holds to the present file, and marks them for
# ensure_commit.
record_present() {
    local sha
    git cat-file --batch-check < "$1" | awk '$2 == "commit" { print $1 }' \
        > "${CACHE_DIR}/present"
    while read -r sha; do
        : > "${CACHE_DIR}/commit-ok-${sha}"
    done < "${CACHE_DIR}/present"
}

# Records in CLEAN whether each present commit matches COMPARE_REF on the set's paths,
# with one diff-tree for every commit.
record_clean() {
    local set=$1
    local -n set_paths="${set}_PATHS"
    local dirty="${CACHE_DIR}/dirty-${set}"
    local sha
    if ! sed "s/\$/ ${COMPARE_REF}/" "${CACHE_DIR}/present" \
        | git diff-tree --stdin -r -s -- "${set_paths[@]}" \
            > "${dirty}" 2> /dev/null; then
        return 0
    fi
    while read -r sha; do
        CLEAN["${set}:${sha}"]=1
    done < "${CACHE_DIR}/present"
    while read -r sha; do
        CLEAN["${set}:${sha}"]=0
    done < "${dirty}"
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

# True when none of the set's paths differ between the commit and COMPARE_REF.
paths_clean() {
    local set=$1
    local sha=$2
    local known=${CLEAN["${set}:${sha}"]:-}
    if [ -n "${known}" ]; then
        [ "${known}" = 1 ]
        return
    fi
    local -n set_paths="${set}_PATHS"
    git diff --quiet "${sha}" "${COMPARE_REF}" -- "${set_paths[@]}" 2> /dev/null
}

# Prints run_id:sha for every live artifact of the name, newest first.
# Expired artifacts still appear in the API listing but cannot be downloaded.
artifact_rows() {
    gh api --paginate "repos/:owner/:repo/actions/artifacts?name=$1&per_page=100" \
        --jq '.artifacts[] | select(.expired == false)
            | "\(.workflow_run.id):\(.workflow_run.head_sha)"' 2> /dev/null || true
}

runs_at_sha() {
    awk -F: -v s="$1" '$2 == s { print $1 }' <<< "${RECENT_RUNS}"
}

# True when the run holds a live artifact for every name.
run_has_artifacts() {
    local run_id=$1
    local names=$2
    local name
    for name in ${names}; do
        case " ${ARTIFACT_RUNS[${name}]} " in
            *" ${run_id} "*) ;;
            *) return 1 ;;
        esac
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

# Prints the workflow step that builds the component on the platform.
build_step() {
    if [ "$1" = driver ]; then
        echo "${DRIVER_STEP[$2]}"
    else
        echo "${BUILD_STEP[$1]}"
    fi
}

# True when the platform's build job skipped the named step, as it does when it reuses
# a cached build. The job log then holds no build to compare.
step_skipped() {
    local file=$1
    local os=$2
    local name=$3
    jq -e --arg j "${BUILD_JOB[${os}]}" --arg n "${name}" \
        '[.jobs[] | select(.name | endswith($j))][0].steps[]?
            | select(.name == $n and .conclusion == "skipped")' \
        "${file}" > /dev/null 2>&1
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
        ref=$("$(dirname "$0")/toolchain_ref.sh" "${WORKFLOW}" "${action}")
        if [ -z "${ref}" ] || ! grep -qxF "##[group]Run ${ref}" "${log}"; then
            return 1
        fi
    done
    local step env_pat="" env=""
    step=$(build_step "${comp}" "${os}")
    case "${comp}" in
        driver)
            env_pat='^  BAZEL_[A-Z_]*: '
            env=$(expected_env)
            ;;
        console)
            env_pat='^  VITE_FLAGS_ENABLED: '
            env="VITE_FLAGS_ENABLED: ${FLAGS_ENABLED}"
            ;;
    esac
    local script marker expected actual
    script=$(expected_script "${step}")
    marker=$(awk 'NF { l = $0 } END { sub(/^[ \t]+/, "", l); print l }' <<< "${script}")
    expected=${script}
    if [ -n "${env}" ]; then
        expected+=$'\n'${env}
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
        for comp in ${comps}; do
            if step_skipped "${file}" "${os}" "$(build_step "${comp}" "${os}")"; then
                log "run ${run_id}: ${comp} ${os} build skipped"
                return 1
            fi
        done
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

# Prints the newest run with live artifacts for $2 and no diff on the path set $5.
# Exact-sha lookups cover this branch. The recent-run scan covers other branches.
find_reusable_run() {
    local label=$1
    local artifact_names=$2
    local component=$3
    local os_list=$4
    local set=$5

    local sha run_id
    for sha in ${CANDIDATE_SHAS}; do
        if ! paths_clean "${set}" "${sha}"; then
            continue
        fi
        for run_id in $(runs_at_sha "${sha}"); do
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
        if [ "${run_id}" = "${GITHUB_RUN_ID:-}" ] \
            || ! run_has_artifacts "${run_id}" "${artifact_names}" \
            || ! ensure_commit "${sha}" \
            || ! paths_clean "${set}" "${sha}"; then
            continue
        fi
        if run_build_matches "${run_id}" "${component}" "${os_list}"; then
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
    ensure_commit "${COMPARE_REF}" || COMPARE_REF="HEAD"
    COMPARE_REF=$(git rev-parse "${COMPARE_REF}^{commit}")
    ensure_history

    # Candidates older than the 7-day artifact retention cannot hit.
    CANDIDATE_SHAS=$(git rev-list --since=7.days --max-count=30 "${COMPARE_REF}" \
        2> /dev/null || true)

    local os name rows
    RECENT_RUNS=""
    for os in "${OS_NAMES[@]}"; do
        for name in "synnax-core-${os}" "synnax-driver-${os}" \
            "synnax-console-assets-${os}"; do
            rows=$(artifact_rows "${name}")
            ARTIFACT_RUNS[${name}]=$(cut -d: -f1 <<< "${rows}" | tr '\n' ' ')
            RECENT_RUNS+="${rows}"$'\n'
        done
    done
    RECENT_RUNS=$(sed '/^$/d' <<< "${RECENT_RUNS}" | sort -t: -k1,1 -rnu)

    local shas="${CACHE_DIR}/shas"
    { cut -d: -f2 <<< "${RECENT_RUNS}" && echo "${CANDIDATE_SHAS}"; } \
        | sed '/^$/d' | sort -u > "${shas}"
    fetch_missing "${shas}"
    record_present "${shas}"
    record_clean DRIVER
    record_clean CONSOLE
    record_clean UNION

    local full_run
    full_run=$(find_reusable_run "full" "${FULL_ARTIFACTS[*]}" all "${OS_NAMES[*]}" \
        UNION)
    if [ -n "${full_run}" ]; then
        log "✅ Skipping build. Using artifacts from run ${full_run}"
        emit "SKIP_BUILD=true"
        emit "REF_RUN_ID=${full_run}"
        return 0
    fi

    # Each platform resolves on its own, so a windows-only run can serve the windows
    # half of a build that also does Ubuntu.
    local driver_ids="{}" console_ids="{}" core_ids="{}" run
    for os in "${OS_NAMES[@]}"; do
        run=$(find_reusable_run "driver ${os}" "synnax-driver-${os}" driver "${os}" \
            DRIVER)
        driver_ids=$(jq -c --arg os "${os}" --arg run "${run}" '.[$os] = $run' \
            <<< "${driver_ids}")
        run=$(find_reusable_run "console ${os}" "synnax-console-assets-${os}" console \
            "${os}" CONSOLE)
        console_ids=$(jq -c --arg os "${os}" --arg run "${run}" '.[$os] = $run' \
            <<< "${console_ids}")
        run=$(find_reusable_run "core ${os}" \
            "synnax-core-${os} synnax-driver-${os} synnax-console-assets-${os}" \
            all "${os}" UNION)
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
