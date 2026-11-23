#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints true when a maintainer approved a commit of a pull request, else false. Usage:
# fork_approved.sh <pr_number> <sha>. A maintainer is a person with write access. Only
# the last approval or request for changes of each person on the commit counts. Exits
# non-zero only when the GitHub API fails.

set -euo pipefail

PR=${1:?usage: fork_approved.sh <pr_number> <sha>}
export SHA=${2:?usage: fork_approved.sh <pr_number> <sha>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}
GH_API="$(dirname "$0")/gh_api.sh"

# Pages come oldest first, and jq sees one page at a time, so awk keeps the last
# verdict.
VERDICTS=$("$GH_API" "repos/${REPO}/pulls/${PR}/reviews?per_page=100" --paginate \
    --jq '.[] | select(.commit_id == env.SHA and .user.type == "User"
        and (.state == "APPROVED" or .state == "CHANGES_REQUESTED"))
        | "\(.user.login) \(.state)"')
APPROVERS=$(awk '{ last[$1] = $2 } END { for (l in last) if (last[l] == "APPROVED")
    print l }' <<< "$VERDICTS")

for login in $APPROVERS; do
    PERMISSION=$("$GH_API" "repos/${REPO}/collaborators/${login}/permission" \
        --jq .permission)
    case $PERMISSION in
        admin | write)
            echo true
            exit 0
            ;;
    esac
done
echo false
