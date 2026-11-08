#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Decides whether Sedosaur skips a request for a review, and prints true or false. Reads
# GITHUB_REPOSITORY, HEAD_SHA, KEY, FORK, REPEAT, and REPLAY. KEY is the external ID of
# the review's check run, `<pr>:<base ref>`. Exits non-zero only when the GitHub API
# fails.
#
# A fork's code never reaches a runner. A replay and a repeat always run, and only a
# manual dispatch asks for one. Sessions often ask again for a review of a head that
# has one. The reviews of one pull request take turns, so a head needs no other review
# when its last review for this pull request and base succeeded.

set -euo pipefail

REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}
GH_API="$(dirname "$0")/gh_api.sh"

if [ "${FORK:?}" = true ]; then
    echo true
    exit 0
fi
if [ "${REPLAY:?}" = true ] || [ "${REPEAT:?}" = true ]; then
    echo false
    exit 0
fi
export KEY=${KEY:?}
RUNS="repos/${REPO}/commits/${HEAD_SHA:?}/check-runs"
"$GH_API" "${RUNS}?check_name=Sedosaur%20Review&filter=all&per_page=100" --jq '
    [.check_runs[] | select(.app.slug == "sedosaur") | select(.external_id == env.KEY)]
    | max_by(.started_at) | .conclusion == "success"'
