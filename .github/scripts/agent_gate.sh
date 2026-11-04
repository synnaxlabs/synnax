#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Decides whether an agent workflow may act on a pull request. Usage: agent_gate.sh <pr>
# <kind> <cap> [<sha>]. Prints "run <attempt>", "skip <reason>", or "stop" when the
# attempts reached the cap. A pull request qualifies when it is open, comes from this
# repository, has no agent/needs-human label, and an agent in AGENT_LOGINS (a JSON
# array) opened it or it has the agent/autofix label. With a sha, a head that moved
# skips. An attempt is a GitHub Actions comment holding "<!-- agent:<kind> -->", counted
# from the last removal of the agent/needs-human label.

set -euo pipefail

USAGE="usage: agent_gate.sh <pr> <kind> <cap> [<sha>]"
PR=${1:?$USAGE}
KIND=${2:?$USAGE}
CAP=${3:?$USAGE}
SHA=${4:-}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}
AGENTS=${AGENT_LOGINS:-'["synnax-claude-code[bot]", "synnax-codex[bot]"]'}

skip() {
    echo "skip $1"
    exit 0
}

PULL=$(gh api "repos/${REPO}/pulls/${PR}")
check() { jq -e "$@" <<< "$PULL" > /dev/null; }

check '.state == "open"' || skip "the pull request is closed"
check --arg repo "$REPO" '.head.repo.full_name == $repo' \
    || skip "the pull request comes from a fork"
if [ -n "$SHA" ]; then
    check --arg sha "$SHA" '.head.sha == $sha' || skip "the head moved"
fi
check --argjson agents "$AGENTS" \
    '(.user.login | IN($agents[])) or any(.labels[]; .name == "agent/autofix")' \
    || skip "an agent did not open the pull request"
if check 'any(.labels[]; .name == "agent/needs-human")'; then
    skip "the pull request waits for a person"
fi

SINCE=$(gh api "repos/${REPO}/issues/${PR}/events?per_page=100" --paginate \
    | jq -rs '[add[] | select(.event == "unlabeled"
        and .label.name == "agent/needs-human") | .created_at] | max // ""')
ATTEMPTS=$(gh api "repos/${REPO}/issues/${PR}/comments?per_page=100" --paginate \
    | jq -s --arg marker "<!-- agent:${KIND} -->" --arg since "$SINCE" \
        '[add[] | select(.user.login == "github-actions[bot]"
            and (.body | contains($marker)) and .created_at > $since)] | length')

if [ "$ATTEMPTS" -ge "$CAP" ]; then
    echo stop
else
    echo "run $((ATTEMPTS + 1))"
fi
