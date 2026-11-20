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
# <sha>. Judges the pull request the group adds, if GitHub's ref points at the commit.
# Under ALLGREEN each entry ahead passes its own gate. A merge group cannot wait, so
# anything short of success is a failure. When the gate cannot judge, posts a failure, so the queue does not stall
# until its timeout, and exits non-zero. When the queue ref is gone or points at another
# commit, the queue destroyed the group, so there is nothing to judge, and the script
# exits zero.

set -euo pipefail

QUEUE_REF=${1:?usage: report_queue_gate.sh <queue_ref> <sha>}
SHA=${2:?usage: report_queue_gate.sh <queue_ref> <sha>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}
API="$(dirname "$0")/gh_api.sh"

if [[ ! $QUEUE_REF =~ ^gh-readonly-queue/.+/pr-([0-9]+)-[0-9a-f]{40}$ ]]; then
    echo "not a merge queue ref: $QUEUE_REF" >&2
    exit 1
fi
PR=${BASH_REMATCH[1]}

REF_ERR=$(mktemp)
trap 'rm -f "$REF_ERR"' EXIT
DESTROYED=false
QUEUED=$("$API" "repos/${REPO}/git/ref/heads/${QUEUE_REF}" --jq .object.sha \
    2> "$REF_ERR") || {
    cat "$REF_ERR" >&2
    if grep -q 'HTTP 404' "$REF_ERR"; then DESTROYED=true; fi
}
# A group rebuilt on the same base reuses the ref at a new commit.
if [ -n "$QUEUED" ] && [ "$QUEUED" != "$SHA" ]; then DESTROYED=true; fi
if [ "$DESTROYED" = true ]; then
    echo "::notice::The queue destroyed merge group ${QUEUE_REF} at ${SHA}."
    exit 0
fi

# Prints the verdict for the group, or fails when the pull request cannot be judged.
judge() {
    local verdict state description
    [ "$QUEUED" = "$SHA" ] || return 1
    verdict=$("$(dirname "$0")/check_review.sh" "$PR") || return 1
    IFS=$'\t' read -r state description <<< "$verdict"
    [ "$state" = success ] || state=failure
    printf '%s\t%s\n' "$state" "$description"
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
