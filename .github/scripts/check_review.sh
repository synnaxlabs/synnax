#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Fails unless a pull request carries exactly one review tier label, the tier is at or
# above the floor its changed paths set, and a tier that needs a human has an approval
# from someone other than the author. Usage: check_review.sh <pr_number>. The tiers and
# the paths that set a floor are documented in CONTRIBUTING.md.

set -euo pipefail

PR=${1:?usage: check_review.sh <pr_number>}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}

TIERS="review/thorough review/light review/bot"

# Non-test files under these paths can never be reviewed by the bot alone.
floor() {
    case "$1" in
        *_test.go | *_test.py | *_test.cpp | *.spec.ts | *.spec.tsx) echo bot ;;
        schemas/* | oracle/* | cesium/* | aspen/* | x/go/gorp/* | x/go/kv/* | \
            core/pkg/storage/* | core/pkg/distribution/* | \
            core/pkg/service/framer/* | core/pkg/service/channel/* | \
            core/pkg/service/ontology/* | driver/pipeline/*)
            echo light
            ;;
        *) echo bot ;;
    esac
}

rank() {
    case "$1" in
        bot) echo 0 ;;
        light) echo 1 ;;
        thorough) echo 2 ;;
    esac
}

PULL=$(gh api "repos/${REPO}/pulls/${PR}" \
    --jq '{author: .user.login, labels: [.labels[].name]}')
AUTHOR=$(jq -r .author <<< "$PULL")
LABELS=$(jq -r '.labels[]' <<< "$PULL")

FOUND=()
for tier in $TIERS; do
    if grep -qx "$tier" <<< "$LABELS"; then FOUND+=("$tier"); fi
done
if [ "${#FOUND[@]}" -ne 1 ]; then
    printf 'add exactly one review tier label (%s); found: %s\n' \
        "${TIERS// /, }" "${FOUND[*]:-none}" >&2
    exit 1
fi
TIER=${FOUND[0]#review/}

FILES=$(gh api "repos/${REPO}/pulls/${PR}/files?per_page=100" --paginate \
    --jq '.[].filename')
FLOOR=bot
FLOOR_FILE=""
while IFS= read -r file; do
    [ -n "$file" ] || continue
    f=$(floor "$file")
    if [ "$(rank "$f")" -gt "$(rank "$FLOOR")" ]; then
        FLOOR=$f
        FLOOR_FILE=$file
    fi
done <<< "$FILES"
if [ "$(rank "$TIER")" -lt "$(rank "$FLOOR")" ]; then
    printf 'review/%s is below the floor review/%s set by %s\n' \
        "$TIER" "$FLOOR" "$FLOOR_FILE" >&2
    exit 1
fi

if [ "$TIER" = bot ]; then
    echo "review/bot: no human approval required"
    exit 0
fi

# The latest review by each human decides; a later request for changes or a dismissal
# retires an earlier approval.
APPROVERS=$(gh api "repos/${REPO}/pulls/${PR}/reviews?per_page=100" --paginate \
    --jq '[.[] | select(.user.type != "Bot") | select(.state != "COMMENTED")]
        | group_by(.user.login) | map(last)
        | map(select(.state == "APPROVED" and .user.login != "'"$AUTHOR"'"))
        | .[].user.login')
if [ -z "$APPROVERS" ]; then
    printf 'review/%s needs an approval from someone other than %s\n' \
        "$TIER" "$AUTHOR" >&2
    exit 1
fi
echo "review/${TIER}: approved by $(tr '\n' ' ' <<< "$APPROVERS")"
