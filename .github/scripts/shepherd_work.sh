#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the work a pull request needs before it is done, one item per line, and
# nothing when it is done. Usage: shepherd_work.sh <pr>. The items are "conflict", "ci
# <run id>" for a failed CI run on the head, "threads <count>" for review threads that
# need action, and "outside-diff" for open Greptile findings outside the diff.

set -euo pipefail

PR=${1:?usage: shepherd_work.sh <pr>}
REPO=${GITHUB_REPOSITORY:-synnaxlabs/synnax}
HERE=$(dirname "$0")

# GitHub computes the mergeable state on the first read after a change.
for _ in 1 2 3; do
    PULL=$(gh api "repos/${REPO}/pulls/${PR}" --jq '"\(.head.sha) \(.mergeable_state)"')
    read -r SHA STATE <<< "$PULL"
    [ "$STATE" = unknown ] || break
    sleep "${SHEPHERD_RETRY_SECONDS:-5}"
done
if [ "$STATE" = dirty ]; then
    echo conflict
fi

RUNS="repos/${REPO}/actions/workflows/ci.yaml/runs"
RUN=$(gh api "${RUNS}?head_sha=${SHA}&event=pull_request&per_page=1" \
    --jq '.workflow_runs[0] | select(.conclusion == "failure") | .id')
if [ -n "$RUN" ]; then
    echo "ci $RUN"
fi

THREADS=$("$HERE/review_threads.sh" "$PR" | wc -l | tr -d ' ')
if [ "$THREADS" -gt 0 ]; then
    echo "threads $THREADS"
fi

OUTSIDE=$("$HERE/greptile_outside_diff.sh" "$PR")
if [ -n "$OUTSIDE" ]; then
    echo outside-diff
fi
