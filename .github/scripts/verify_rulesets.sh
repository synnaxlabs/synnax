#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Fails unless the repository rulesets on GitHub match the files in .github/rulesets,
# one ruleset per file, matched by name. Prints a diff for each mismatch. Run from the
# repository root with a token that can read bypass actors, which takes admin access.

set -euo pipefail

REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}
DIR=.github/rulesets
# Keeps the fields a file sets. GitHub returns the rules in no fixed order.
NORMALIZE='{name, target, enforcement, conditions, bypass_actors,
    rules: (.rules | sort_by(.type))}'

LIVE=$(gh api "repos/${REPO}/rulesets?includes_parents=false" --paginate \
    --jq '.[] | "\(.id)\t\(.name)"')
LIVE_NAMES=$(cut -f2 <<< "$LIVE" | sed '/^$/d' | sort)
FILE_NAMES=$(jq -r .name "$DIR"/*.json | sort)

STATUS=0

MISSING=$(comm -23 <(echo "$FILE_NAMES") <(echo "$LIVE_NAMES") | sed '/^$/d')
if [ -n "$MISSING" ]; then
    printf 'rulesets not on GitHub:\n%s\n' "$(sed 's/^/  /' <<< "$MISSING")" >&2
    STATUS=1
fi

UNTRACKED=$(comm -13 <(echo "$FILE_NAMES") <(echo "$LIVE_NAMES") | sed '/^$/d')
if [ -n "$UNTRACKED" ]; then
    printf 'rulesets on GitHub with no file in %s:\n%s\n' "$DIR" \
        "$(sed 's/^/  /' <<< "$UNTRACKED")" >&2
    STATUS=1
fi

for FILE in "$DIR"/*.json; do
    NAME=$(jq -r .name "$FILE")
    ID=$(awk -F'\t' -v name="$NAME" '$2 == name { print $1 }' <<< "$LIVE")
    [ -n "$ID" ] || continue
    if ! diff -u --label "$FILE" --label "GitHub ruleset $NAME" \
        <(jq -S "$NORMALIZE" "$FILE") \
        <(gh api "repos/${REPO}/rulesets/${ID}" | jq -S "$NORMALIZE") >&2; then
        STATUS=1
    fi
done

if [ "$STATUS" -eq 0 ]; then
    echo "$(wc -l <<< "$FILE_NAMES" | tr -d ' ') rulesets match $DIR"
fi
exit "$STATUS"
