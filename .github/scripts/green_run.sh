#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Finds a run on the commit in which the full CI passed. Prints the run ID and emits
# green=true and run_id. With no such run, prints nothing and emits green=false. An
# API error counts as no green run, so the exit code is 0 in both cases.
# Usage: green_run.sh <full commit SHA>

set -uo pipefail

SHA=${1:-}
if ! [[ ${SHA} =~ ^[0-9a-f]{40}$ ]]; then
    echo "usage: green_run.sh <full commit SHA>" >&2
    exit 2
fi
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}

# The workflows that call ci.yaml with full: true, in a job named CI.
WORKFLOWS=(nightly.yaml release.yaml)
JOB="CI / OK"
# A page of 100 jobs of a large run at times gets an HTTP 502.
PER_PAGE=50
API="$(dirname "$0")/gh_api.sh"

log() {
    echo "[green-run] $1" >&2
}

emit() {
    echo "$1" >> "${GITHUB_OUTPUT:-/dev/null}"
}

# Prints every item under the key of a paged listing, one JSON object per line. Fails
# when a page cannot be read.
list() {
    local url=$1 key=$2 page=1 body count
    while :; do
        body=$("${API}" "${url}&per_page=${PER_PAGE}&page=${page}") || return 1
        jq -c --arg k "${key}" '.[$k][]' <<< "${body}" || return 1
        count=$(jq --arg k "${key}" '.[$k] | length' <<< "${body}") || return 1
        if ((count < PER_PAGE)); then
            return 0
        fi
        page=$((page + 1))
    done
}

# True when the CI job passed in any attempt of the run. A rerun of all jobs starts a
# new attempt with no CI result, and the default listing holds only that attempt.
passed() {
    local jobs
    if ! jobs=$(list "repos/${REPO}/actions/runs/$1/jobs?filter=all" jobs); then
        log "run $1: cannot read the jobs"
        return 1
    fi
    jq -e -s --arg n "${JOB}" 'any(.[];
        .name == $n and .status == "completed" and .conclusion == "success")' \
        <<< "${jobs}" > /dev/null
}

for workflow in "${WORKFLOWS[@]}"; do
    url="repos/${REPO}/actions/workflows/${workflow}/runs?head_sha=${SHA}"
    if ! runs=$(list "${url}" workflow_runs); then
        log "${workflow}: cannot read the runs"
        continue
    fi
    ids=$(jq -r --arg sha "${SHA}" --arg repo "${REPO}" \
        'select(.head_sha == $sha and .head_repository.full_name == $repo) | .id' \
        <<< "${runs}")
    for id in ${ids}; do
        if passed "${id}"; then
            log "${JOB} passed in ${workflow} run ${id}"
            emit "green=true"
            emit "run_id=${id}"
            echo "${id}"
            exit 0
        fi
    done
done

log "no green run of ${SHA}"
emit "green=false"
