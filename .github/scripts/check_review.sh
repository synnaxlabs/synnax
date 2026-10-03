#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Decides the Review gate commit status for a pull request. Usage: check_review.sh
# <pr_number>. Prints "<state>\t<description>" where state is pending (waiting on a tier
# label, the Greptile review, or a human approval), failure (more than one tier label,
# or a Greptile score below 5/5), or success. Exits non-zero only when the GitHub API
# fails. The tiers are documented in CONTRIBUTING.md.

set -euo pipefail

PR=${1:?usage: check_review.sh <pr_number>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}

TIERS="review/thorough review/light review/bot"

report() {
    printf '%s\t%s\n' "$1" "$2"
    exit 0
}

PULL=$(gh api "repos/${REPO}/pulls/${PR}" \
    --jq '{author: .user.login, head: .head.sha, labels: [.labels[].name]}')
AUTHOR=$(jq -r .author <<< "$PULL")
HEAD=$(jq -r .head <<< "$PULL")
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

# Only a review of the head counts: a push after the review needs a new one. The latest
# filter keeps a rerun's verdict from being shadowed by an earlier success.
REVIEW=$(gh api "repos/${REPO}/commits/${HEAD}/check-runs?filter=latest&per_page=100" \
    --paginate \
    --jq '.check_runs[]
        | select(.name == "Greptile Review" and .conclusion == "success") | .id')
if [ -z "$REVIEW" ]; then
    report pending "review/${TIER}: waiting for the Greptile review of the head"
fi

# The check run succeeds at any score. The score lives only in the description, which
# the author can edit, so only the latest revision Greptile wrote counts, and only when
# it names the head. Every score marker must be a 5: a commit title can quote one. The
# diff field holds the whole description at that revision, not a delta.
SCORE=$(gh api graphql \
    -f query='query($owner: String!, $name: String!, $number: Int!) {
        repository(owner: $owner, name: $name) {
            pullRequest(number: $number) {
                userContentEdits(first: 100) {
                    nodes { editor { __typename login } diff }
                }
            }
        }
    }' \
    -f owner="${REPO%/*}" -f name="${REPO#*/}" -F number="$PR" \
    --jq '[.data.repository.pullRequest.userContentEdits.nodes[]
            | select(.editor | .__typename == "Bot" and .login == "greptile-apps")]
        | (.[0].diff // "")
        | select(contains("/commit/'"$HEAD"')"))
        | [scan("greptile_confidence_score:([0-9]+)")[]] | unique | join(" ")')
if [ -z "$SCORE" ]; then
    report pending "review/${TIER}: waiting for Greptile to score the head"
elif [ "$SCORE" != 5 ]; then
    report failure "review/${TIER}: Greptile scored the head ${SCORE}/5, not 5/5"
fi

if [ "$TIER" = bot ]; then report success "review/bot: no human approval required"; fi

# The latest review by each human decides; a later request for changes or a dismissal
# retires an earlier approval. Pages are joined before grouping, since gh applies a --jq
# filter to each page on its own.
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
