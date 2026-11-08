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
# success. A pull request needs one tier label and a Sedosaur 5/5 of the head. A pull
# request Dependabot opened and wrote every commit of passes without a review. A fork
# fails, since Sedosaur does not review one. Exits non-zero only when the GitHub API
# fails.

set -euo pipefail

PR=${1:?usage: check_review.sh <pr_number>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}
GH_API="$(dirname "$0")/gh_api.sh"

report() {
    printf '%s\t%s\n' "$1" "$2"
    exit 0
}

PULL=$("$GH_API" "repos/${REPO}/pulls/${PR}" --jq '"\(.head.sha) \(.base.ref) \(
    .head.repo.full_name != .base.repo.full_name) \(.user.login) \(
    [.labels[].name | select(startswith("review/"))] | join(","))"')
read -r HEAD BASE FORK AUTHOR TIER <<< "$PULL"

# Git author emails are free to set, so a commit counts as Dependabot's only when GitHub
# also signed and committed it.
if [ "$AUTHOR" = "dependabot[bot]" ]; then
    OTHERS=$("$GH_API" "repos/${REPO}/pulls/${PR}/commits?per_page=100" --paginate \
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

if [ "$FORK" = true ]; then
    report failure "Sedosaur does not review forks, so an admin merges this one"
fi

# Only a review of the head counts. A push after the review needs a new one, unless it
# leaves the diff the same: the head then gets the last review, and changes of the base
# outside the diff are left to CI. The newest review of this pull request and base
# decides, since a commit can head two pull requests and a rerun must not be shadowed by
# an earlier success. Only the Sedosaur app
# can write its check runs, so the score in the title is its own.
export KEY="${PR}:${BASE}"
RUNS="repos/${REPO}/commits/${HEAD}/check-runs"
REVIEW=$("$GH_API" "${RUNS}?check_name=Sedosaur%20Review&filter=all&per_page=100" \
    --jq '[.check_runs[] | select(.app.slug == "sedosaur" and .external_id == env.KEY)]
        | max_by(.started_at) // {}
        | "\(.conclusion // "none") \(.output.title // ""
            | capture("^Confidence Score: (?<n>[0-9])/5$").n // "none")"')
read -r CONCLUSION SCORE <<< "$REVIEW"
case "$CONCLUSION:$SCORE" in
    none:*) report pending "waiting for the Sedosaur review of the head" ;;
    success:5) report success "Sedosaur scored the head 5/5" ;;
    success:none) report failure "Sedosaur left no score on the head" ;;
    success:*) report failure "Sedosaur scored the head ${SCORE}/5, not 5/5" ;;
    *) report failure "the Sedosaur review of the head ended ${CONCLUSION}" ;;
esac
