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
# <run id>" for a failed CI run on the head, "codeql <path>:<line> <rule> <message>"
# for each open CodeQL alert the pull request adds, "queue [<run id>]" for a merge
# queue removal after the head commit, with the failed merge group run when found,
# "threads <count>" for review threads that need action, and "outside-diff" for open
# Sedosaur findings outside the diff.

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
QUERY="head_sha=${SHA}&event=pull_request&per_page=1"
read -r RUN CONCLUSION < <(gh api "${RUNS}?${QUERY}" \
    --jq '.workflow_runs[0] | "\(.id) \(.conclusion)"')
# CI wakes the shepherd from inside the run, once its OK job fails.
if [ "$RUN" != null ] && [ "$CONCLUSION" = null ]; then
    CONCLUSION=$(gh api "repos/${REPO}/actions/runs/${RUN}/jobs?per_page=100" \
        --paginate --jq '.jobs[] | select(.name == "OK") | .conclusion')
fi
if [ "$CONCLUSION" = failure ]; then
    echo "ci $RUN"
fi

# The alerts come from the last finished analysis, so they wait for the head's scan.
# CodeQL on a pull request reports only the alerts on lines it changed.
SCANS="repos/${REPO}/actions/workflows/codeql.yaml/runs?${QUERY}"
if [ "$(gh api "$SCANS" --jq '.workflow_runs[0].status')" = completed ]; then
    ALERTS="code-scanning/alerts?ref=refs/pull/${PR}/merge&state=open&per_page=100"
    # shellcheck disable=SC2016 # The jq program uses jq variables.
    gh api "repos/${REPO}/${ALERTS}" --paginate --jq '.[]
        | .most_recent_instance as $at
        | "codeql \($at.location.path):\($at.location.start_line) \(.rule.id)"
            + " \($at.message.text | gsub("\\s+"; " "))"'
fi

# The span from the head commit to a later merge queue removal for failed checks that
# no new enqueue followed.
# shellcheck disable=SC2016 # The GraphQL query uses GraphQL variables.
SPAN=$(gh api graphql -F p="$PR" -f o="${REPO%/*}" -f r="${REPO#*/}" -f query='
query($o: String!, $r: String!, $p: Int!) {
  repository(owner: $o, name: $r) {
    pullRequest(number: $p) {
      commits(last: 1) { nodes { commit { committedDate } } }
      timelineItems(
        last: 1
        itemTypes: [ADDED_TO_MERGE_QUEUE_EVENT, REMOVED_FROM_MERGE_QUEUE_EVENT]
      ) {
        nodes { ... on RemovedFromMergeQueueEvent { createdAt reason } }
      }
    }
  }
}' --jq '.data.repository.pullRequest
    | .commits.nodes[0].commit.committedDate as $head
    | .timelineItems.nodes[0] as $removal
    | select($removal.reason == "failed_checks" and $removal.createdAt > $head)
    | "\($head)..\($removal.createdAt)"')
# The removal cancels the runs of its merge group, so a failed run can end cancelled.
if [ -n "$SPAN" ]; then
    GROUP_RUNS=$(gh api "repos/${REPO}/actions/runs?event=merge_group&created=${SPAN}" \
        --paginate --jq ".workflow_runs[]
            | select(.head_branch | test(\"^gh-readonly-queue/.+/pr-${PR}-\"))
            | \"\(.conclusion) \(.id)\"")
    GROUP=$(grep -m1 '^failure ' <<< "$GROUP_RUNS" \
        || grep -m1 '^cancelled ' <<< "$GROUP_RUNS" || true)
    echo "queue${GROUP:+ ${GROUP#* }}"
fi

THREADS=$("$HERE/review_threads.sh" "$PR" | wc -l | tr -d ' ')
if [ "$THREADS" -gt 0 ]; then
    echo "threads $THREADS"
fi

OUTSIDE=$("$HERE/outside_diff.sh" "$PR")
if [ -n "$OUTSIDE" ]; then
    echo outside-diff
fi
