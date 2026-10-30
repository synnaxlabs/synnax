#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Turns auto-merge on or off for a pull request. Usage: auto_merge.sh <pr_number>. It
# turns auto-merge on for a pull request into main or a release branch that is labeled
# review/bot, not review/human, and is not a draft. When a pull request stops
# qualifying, it turns off only auto-merge that the bot turned on; auto-merge that a
# person turned on stays on. GitHub does not allow auto-merge on a pull request in a
# stack, so the script leaves a stacked pull request unchanged. Prints the decision.

set -euo pipefail

PR=${1:?usage: auto_merge.sh <pr_number>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}
BOT=${BOT:-synnax-claude-code[bot]}

# shellcheck disable=SC2016 # The jq program uses jq variables.
read -r stacked want enabled ours < <(
    gh api "repos/$REPO/pulls/$PR" --jq '[.labels[].name] as $l
        | (.draft == false and any($l[]; . == "review/bot")
            and all($l[]; . != "review/human")
            and (.base.ref == "main" or (.base.ref | startswith("release/"))))
          as $want
        | (.auto_merge.enabled_by.login == "'"$BOT"'") as $ours
        | "\(.stack != null) \($want) \(.auto_merge != null) \($ours)"'
)
if [ "$stacked" = true ]; then
    echo "unchanged: the pull request is in a stack"
elif [ "$want" = true ] && [ "$enabled" = false ]; then
    gh pr merge "$PR" --repo "$REPO" --auto --squash
    echo enabled
elif [ "$want" = false ] && [ "$ours" = true ]; then
    gh pr merge "$PR" --repo "$REPO" --disable-auto
    echo disabled
else
    echo unchanged
fi
