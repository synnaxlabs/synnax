#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Decides the Review gate commit status for a pull request. Usage:
# check_review.sh <pr_number>. Prints "<state>\t<description>" where state is pending
# (waiting on a tier label, the Greptile review, or a human approval), failure (more
# than one tier label), or success. Exits non-zero only when the GitHub API fails. The
# tiers are documented in CONTRIBUTING.md.

set -euo pipefail

PR=${1:?usage: check_review.sh <pr_number>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}

TIERS="review/thorough review/light review/bot"

report() {
    printf '%s\t%s\n' "$1" "$2"
    exit 0
}

# Prints the id of a successful Greptile check run on the pull request, or nothing.
# Greptile skips a push that adds no new changes, such as a merge from the base branch,
# so its check run can sit on an ancestor of the head rather than on the head itself.
# The walk over the commits stops at the first success.
greptile_review() {
    local commits sha found
    commits=$(gh api "repos/${REPO}/pulls/${PR}/commits?per_page=100" --paginate \
        --jq '.[].sha')
    while read -r sha; do
        found=$(gh api "repos/${REPO}/commits/${sha}/check-runs?per_page=100" \
            --paginate --jq '.check_runs[]
                | select(.name == "Greptile Review" and .conclusion == "success")
                | .id')
        if [ -n "$found" ]; then
            echo "$found"
            return 0
        fi
    done <<< "$commits"
}

PULL=$(gh api "repos/${REPO}/pulls/${PR}" \
    --jq '{author: .user.login, labels: [.labels[].name]}')
AUTHOR=$(jq -r .author <<< "$PULL")
LABELS=$(jq -r '.labels[]' <<< "$PULL")

FOUND=()
for tier in $TIERS; do
    if grep -qx "$tier" <<< "$LABELS"; then FOUND+=("$tier"); fi
done
if [ "${#FOUND[@]}" -eq 0 ]; then
    report pending "add one review tier label: ${TIERS// /, }"
elif [ "${#FOUND[@]}" -gt 1 ]; then
    report failure "keep one review tier label, found ${FOUND[*]}"
fi
TIER=${FOUND[0]#review/}

REVIEW=$(greptile_review)
if [ -z "$REVIEW" ]; then
    report pending "review/${TIER}: waiting for the Greptile review"
fi

if [ "$TIER" = bot ]; then report success "review/bot: no human approval required"; fi

# The latest review by each human decides; a later request for changes or a dismissal
# retires an earlier approval. Pages are joined before grouping, since gh applies a
# --jq filter to each page on its own.
APPROVERS=$(gh api "repos/${REPO}/pulls/${PR}/reviews?per_page=100" --paginate \
    | jq -rs 'add
        | map(select(.user.type != "Bot" and .state != "COMMENTED"))
        | group_by(.user.login) | map(last)
        | map(select(.state == "APPROVED" and .user.login != "'"$AUTHOR"'"))
        | .[].user.login')
if [ -z "$APPROVERS" ]; then
    report pending "review/${TIER}: waiting for an approval from someone but $AUTHOR"
fi
NAMES=$(tr '\n' ' ' <<< "$APPROVERS" | sed 's/ $//')
report success "review/${TIER}: approved by $NAMES"
