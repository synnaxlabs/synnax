#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints Sedosaur's "Comments Outside Diff" comment on a pull request when it lists
# findings that need action, and nothing otherwise. Usage: outside_diff.sh <pr>.
# Sedosaur keeps these findings in one top-level comment that it edits on each review. A
# bot comment holding "<!-- agent:outside-diff -->" after Sedosaur's last edit answers
# them until Sedosaur reviews again.

set -euo pipefail

PR=${1:?usage: outside_diff.sh <pr>}
REPO=${GITHUB_REPOSITORY:-synnaxlabs/synnax}

gh api "repos/${REPO}/issues/${PR}/comments?per_page=100" --paginate | jq -rs '
    add as $all
    | [$all[] | select(.user.login == "sedosaur[bot]"
        and (.body | contains("<!-- sedosaur_outside_diff -->")))] | last
    | select(. != null and (.body | test("(?m)^- ")))
    | . as $review
    | [$all[] | select(.user.type == "Bot"
        and (.body | contains("<!-- agent:outside-diff -->"))) | .created_at]
    | select((max // "") < $review.updated_at)
    | $review.body'
