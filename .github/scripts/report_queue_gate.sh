#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Posts the Review gate status of a merge group. Usage: report_queue_gate.sh <queue_ref>
# <sha>. Judges every pull request in the group, read from the "(#N)" that ends each
# commit title between the target branch and the commit, if GitHub's ref points at the
# commit. Under HEADGREEN only the last entry's status gates a merge, so it must cover
# the entries ahead. A merge group cannot wait, so anything short of success is a
# failure. When the gate cannot judge, posts a failure, so the queue does not stall
# until its timeout, and exits non-zero. When the queue ref is gone, the queue destroyed
# the group, so there is nothing to judge, and the script exits zero.

set -euo pipefail

QUEUE_REF=${1:?usage: report_queue_gate.sh <queue_ref> <sha>}
SHA=${2:?usage: report_queue_gate.sh <queue_ref> <sha>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}
API="$(dirname "$0")/gh_api.sh"

if [[ ! $QUEUE_REF =~ ^gh-readonly-queue/(.+)/pr-([0-9]+)-[0-9a-f]{40}$ ]]; then
    echo "not a merge queue ref: $QUEUE_REF" >&2
    exit 1
fi
TARGET=${BASH_REMATCH[1]}
PR=${BASH_REMATCH[2]}

REF_ERR=$(mktemp)
trap 'rm -f "$REF_ERR"' EXIT
QUEUED=$("$API" "repos/${REPO}/git/ref/heads/${QUEUE_REF}" --jq .object.sha \
    2> "$REF_ERR") || {
    cat "$REF_ERR" >&2
    if grep -q 'HTTP 404' "$REF_ERR"; then
        echo "::notice::The queue destroyed merge group ${QUEUE_REF}."
        exit 0
    fi
    QUEUED=
}

# Prints the verdict for the group, or fails when a pull request cannot be judged.
judge() {
    local prs pr verdict state description own=
    [ "$QUEUED" = "$SHA" ] || return 1
    local filter='.commits[].commit.message | split("\n")[0]'
    filter+=' | capture("\\(#(?<n>[0-9]+)\\)$").n'
    prs=$("$API" "repos/${REPO}/compare/${TARGET}...${SHA}" --jq "$filter")
    grep -qx "$PR" <<< "$prs" || return 1
    for pr in $prs; do
        verdict=$("$(dirname "$0")/check_review.sh" "$pr") || {
            echo "could not judge pull request ${pr}" >&2
            return 1
        }
        IFS=$'\t' read -r state description <<< "$verdict"
        if [ "$state" != success ]; then
            [ "$pr" = "$PR" ] || description="pull request ${pr}: ${description}"
            printf 'failure\t%s\n' "$description"
            return 0
        fi
        [ "$pr" = "$PR" ] && own=$description
    done
    printf 'success\t%s\n' "$own"
}

JUDGED=true
if VERDICT=$(judge); then
    IFS=$'\t' read -r STATE DESCRIPTION <<< "$VERDICT"
else
    JUDGED=false
    STATE=failure
    DESCRIPTION="could not judge pull request ${PR}"
fi

echo "$STATE: $DESCRIPTION"
RUN_URL="$GITHUB_SERVER_URL/$REPO/actions/runs/$GITHUB_RUN_ID"
"$API" "repos/${REPO}/statuses/${SHA}" \
    -f state="$STATE" \
    -f context="Review gate" \
    -f description="$DESCRIPTION" \
    -f target_url="$RUN_URL" > /dev/null

[ "$JUDGED" = true ]
