#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the review threads on a pull request that need action, one JSON object per
# line. Usage: review_threads.sh <pr>. A thread needs action when it is unresolved and
# its last comment is from Greptile or a person other than the pull request author.
# Replies from other bots, such as agents, end the need.

set -euo pipefail

PR=${1:?usage: review_threads.sh <pr>}
REPO=${GITHUB_REPOSITORY:-synnaxlabs/synnax}

gh api graphql --paginate -F p="$PR" -f o="${REPO%/*}" -f r="${REPO#*/}" -f query='
query($o: String!, $r: String!, $p: Int!, $endCursor: String) {
  repository(owner: $o, name: $r) {
    pullRequest(number: $p) {
      author { login }
      reviewThreads(first: 100, after: $endCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id isResolved isOutdated path line
          comments(last: 100) { nodes { author { login __typename } body url } }
        }
      }
    }
  }
}' --jq '.data.repository.pullRequest as $pr | $pr.reviewThreads.nodes[]
    | select((.isResolved | not) and (.comments.nodes[-1].author
        | .login != $pr.author.login
            and (.__typename == "User" or .login == "greptile-apps")))
    | {id, isOutdated, path, line, comments: .comments.nodes} | tojson'
