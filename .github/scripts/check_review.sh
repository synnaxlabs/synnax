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
# <pr_number>. Prints "<state>\t<description>" where state is pending (waiting on the
# Greptile review or its score), failure (a Greptile score below 5/5), or success. Exits
# non-zero only when the GitHub API fails. Human review is a convention documented in
# CONTRIBUTING.md, not a gate.

set -euo pipefail

PR=${1:?usage: check_review.sh <pr_number>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}

report() {
    printf '%s\t%s\n' "$1" "$2"
    exit 0
}

HEAD=$(gh api "repos/${REPO}/pulls/${PR}" --jq '.head.sha')

# Only a review of the head counts: a push after the review needs a new one. The latest
# filter keeps a rerun's verdict from being shadowed by an earlier success.
REVIEW=$(gh api "repos/${REPO}/commits/${HEAD}/check-runs?filter=latest&per_page=100" \
    --paginate \
    --jq '.check_runs[]
        | select(.name == "Greptile Review" and .conclusion == "success") | .id')
if [ -z "$REVIEW" ]; then
    report pending "waiting for the Greptile review of the head"
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
    report pending "waiting for Greptile to score the head"
elif [ "$SCORE" != 5 ]; then
    report failure "Greptile scored the head ${SCORE}/5, not 5/5"
fi

report success "Greptile scored the head 5/5"
