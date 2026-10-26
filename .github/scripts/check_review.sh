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
# <pr_number>. Prints "<state>\t<description>" where state is pending, failure, or
# success. A pull request needs one tier label and a Greptile 5/5 of the head. A
# review/human pull request also needs a member of the repository to approve it, and
# fails while one requests changes. A pull request Dependabot opened and wrote every
# commit of passes, since Greptile does not review it. Exits non-zero only when the
# GitHub API fails.

set -euo pipefail

PR=${1:?usage: check_review.sh <pr_number>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}

report() {
    printf '%s\t%s\n' "$1" "$2"
    exit 0
}

PULL=$(gh api "repos/${REPO}/pulls/${PR}" --jq '"\(.head.sha) \(.user.login) \(
    [.labels[].name | select(startswith("review/"))] | join(","))"')
read -r HEAD AUTHOR TIER <<< "$PULL"

# Git author emails are free to set, so a commit counts as Dependabot's only when GitHub
# also signed and committed it.
if [ "$AUTHOR" = "dependabot[bot]" ]; then
    OTHERS=$(gh api "repos/${REPO}/pulls/${PR}/commits?per_page=100" --paginate \
        --jq '.[] | select(.author.login != "dependabot[bot]"
            or .committer.login != "web-flow"
            or .commit.verification.verified != true) | .sha')
    if [ -z "$OTHERS" ]; then
        report success "Dependabot wrote every commit"
    fi
fi

case "$TIER" in
    review/bot | review/human) ;;
    *) report pending "waiting for one label: review/bot or review/human" ;;
esac

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

if [ "$TIER" = review/bot ]; then
    report success "Greptile scored the head 5/5"
fi

# A reviewer's last approval, change request, or dismissal is their verdict. Anyone can
# review a public repository, so only a member's verdict counts.
VERDICTS=$(gh api "repos/${REPO}/pulls/${PR}/reviews?per_page=100" --paginate \
    --jq '.[] | select(.user.type == "User"
            and (.author_association | IN("OWNER", "MEMBER", "COLLABORATOR"))
            and (.state | IN("APPROVED", "CHANGES_REQUESTED", "DISMISSED")))
        | "\(.user.login) \(.state)"' \
    | awk '{ verdict[$1] = $2 } END { for (u in verdict) print verdict[u], u }' \
    | sort)
REJECTER=$(awk '$1 == "CHANGES_REQUESTED" { print $2; exit }' <<< "$VERDICTS")
APPROVER=$(awk '$1 == "APPROVED" { print $2; exit }' <<< "$VERDICTS")
if [ -n "$REJECTER" ]; then
    report failure "${REJECTER} requested changes"
elif [ -z "$APPROVER" ]; then
    report pending "waiting for a human to approve"
fi

report success "Greptile scored the head 5/5 and ${APPROVER} approved"
