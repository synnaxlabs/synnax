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
# <sha>. Judges the pull request the queue ref names, if GitHub's ref points at the
# commit. A merge group cannot wait, so anything short of success is a failure. When the
# gate cannot judge, posts a failure, so the queue does not stall until its timeout, and
# exits non-zero.

set -euo pipefail

QUEUE_REF=${1:?usage: report_queue_gate.sh <queue_ref> <sha>}
SHA=${2:?usage: report_queue_gate.sh <queue_ref> <sha>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}

if [[ ! $QUEUE_REF =~ ^gh-readonly-queue/.+/pr-([0-9]+)-[0-9a-f]{40}$ ]]; then
    echo "not a merge queue ref: $QUEUE_REF" >&2
    exit 1
fi
PR=${BASH_REMATCH[1]}

JUDGED=true
QUEUED=$("$(dirname "$0")/gh_api.sh" "repos/${REPO}/git/ref/heads/${QUEUE_REF}" \
    --jq .object.sha) || QUEUED=
if [ "$QUEUED" = "$SHA" ] && VERDICT=$("$(dirname "$0")/check_review.sh" "$PR"); then
    IFS=$'\t' read -r STATE DESCRIPTION <<< "$VERDICT"
    if [ "$STATE" != success ]; then
        STATE=failure
    fi
else
    JUDGED=false
    STATE=failure
    DESCRIPTION="could not judge pull request ${PR}"
fi

echo "$STATE: $DESCRIPTION"
RUN_URL="$GITHUB_SERVER_URL/$REPO/actions/runs/$GITHUB_RUN_ID"
"$(dirname "$0")/gh_api.sh" "repos/${REPO}/statuses/${SHA}" \
    -f state="$STATE" \
    -f context="Review gate" \
    -f description="$DESCRIPTION" \
    -f target_url="$RUN_URL" > /dev/null

[ "$JUDGED" = true ]
