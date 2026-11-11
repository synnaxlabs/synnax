#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints a release's notes: the given notes, then the notes GitHub generates from the
# config. The new contributors section is left out, because GitHub counts it across the
# whole repo. A PR that the previous tag shipped as a cherry-pick is left out too. Needs
# a full clone. Usage: release_notes.sh <tag> <sha> <config> [previous_tag] [notes].

set -euo pipefail

USAGE="usage: release_notes.sh <tag> <sha> <config> [previous_tag] [notes]"
TAG=${1:?$USAGE}
SHA=${2:?$USAGE}
CONFIG=${3:?$USAGE}
PREVIOUS_TAG=${4:-}
NOTES=${5:-}
GH_API="$(dirname "$0")/gh_api.sh"

if [ "$(git rev-parse --is-shallow-repository)" = true ]; then
    echo "release_notes.sh needs a full clone, not a shallow one" >&2
    exit 1
fi

# Prints the PR number at the end of a commit's subject.
subject_pr() {
    git log -1 --format=%s "$1" | sed -nE 's/.*\(#([0-9]+)\)$/\1/p'
}

# Prints a "pick <commit> <carrier> <source>" line for each `cherry-pick -x` trailer of
# a commit and a "revert <commit>" line for each commit it reverts. A squash merge drops
# these lines from the message, so a commit with none reads the commits of the PR its
# subject names, and each such commit is the carrier of its trailers.
links() {
    local found pr
    found=$(git log -1 --format=%B "$1" | sed -nE \
        -e "s/^\(cherry picked from commit ([0-9a-f]+)\)$/pick $1 $1 \1/p" \
        -e 's/^This reverts commit ([0-9a-f]+).*/revert \1/p') || return
    pr=$(subject_pr "$1") || return
    if [ -n "$found" ] || [ -z "$pr" ]; then
        echo "$found"
        return
    fi
    "$GH_API" "repos/$GITHUB_REPOSITORY/pulls/$pr/commits" --paginate --jq '
        .[] | .sha as $commit | .commit.message
        | (scan("\\(cherry picked from commit ([0-9a-f]+)\\)")
            | "pick \($commit) \(.[0])"),
          (scan("This reverts commit ([0-9a-f]+)") | "revert \(.[0])")' \
        | sed "s/^pick /pick $1 /"
}

# The commits the previous tag holds that SHA does not are its cherry-picks. A source
# off SHA is itself a pick, such as one from an older train's branch, so its own links
# are read too. A pick that a later commit reverts shipped nothing.
LINKS=""
PICKS=""
SEEN=""
if [ -n "$PREVIOUS_TAG" ]; then
    PICKS=$(git rev-list --no-merges "$PREVIOUS_TAG" "^$SHA")
fi
while [ -n "$PICKS" ]; do
    FOUND=""
    for commit in $PICKS; do
        FOUND+=$(links "$commit")$'\n'
    done
    LINKS+=$FOUND
    PICKS=""
    for source in $(awk '$1 == "pick" { print $4 }' <<< "$FOUND"); do
        if [[ " $SEEN " == *" $source "* ]]; then continue; fi
        SEEN+=" $source"
        if git cat-file -e "$source^{commit}" 2> /dev/null \
            && ! git merge-base --is-ancestor "$source" "$SHA"; then
            PICKS+=" $source"
        fi
    done
done
SOURCES=$(awk '
    $1 == "revert" { reverted[$2] = 1 }
    $1 == "pick" { picks[NR] = $0 }
    END {
        for (i in picks) {
            split(picks[i], f, " ")
            if (!(f[2] in reverted) && !(f[3] in reverted)) print f[4]
        }
    }' <<< "$LINKS" | sort -u | tr '\n' ' ')

# A source with a PR in its subject is that PR's squash merge. GitHub resolves the
# merged PR of any other source, such as a commit of a PR branch. That PR counts as
# shipped only when all of its commits but merges were picked. A source picked from a
# commit that was never pushed has no PR.
SHIPPED=""
ERR=$(mktemp)
trap 'rm -f "$ERR"' EXIT
for source in $SOURCES; do
    PR=""
    if git cat-file -e "$source^{commit}" 2> /dev/null; then
        PR=$(subject_pr "$source")
    fi
    if [ -n "$PR" ]; then
        SHIPPED+=" $PR"
        continue
    fi
    if ! PRS=$("$GH_API" "repos/$GITHUB_REPOSITORY/commits/$source/pulls" \
        --jq '.[] | select(.merged_at) | .number' 2> "$ERR"); then
        if ! grep -q "No commit found for SHA" "$ERR"; then
            cat "$ERR" >&2
            exit 1
        fi
        continue
    fi
    for pr in $PRS; do
        COMMITS=$("$GH_API" "repos/$GITHUB_REPOSITORY/pulls/$pr/commits" --paginate \
            --jq '.[] | select(.parents | length == 1) | .sha')
        ALL=true
        for commit in $COMMITS; do
            if [[ " $SOURCES " != *" $commit "* ]]; then ALL=false; fi
        done
        if [ "$ALL" = true ]; then SHIPPED+=" $pr"; fi
    done
done

ARGS=(-f tag_name="$TAG" -f target_commitish="$SHA")
ARGS+=(-f configuration_file_path="$CONFIG")
if [ -n "$PREVIOUS_TAG" ]; then ARGS+=(-f previous_tag_name="$PREVIOUS_TAG"); fi
if [ -n "$NOTES" ]; then printf '%s\n\n' "$NOTES"; fi
gh api "repos/$GITHUB_REPOSITORY/releases/generate-notes" "${ARGS[@]}" --jq .body \
    | awk -v shipped="$SHIPPED" '
        BEGIN { split(shipped, prs, " "); for (i in prs) drop["/pull/" prs[i]] = 1 }
        /^## New Contributors/ { skip = 1 }
        /^\*\*Full Changelog\*\*/ { skip = 0 }
        skip { next }
        /^\* / && match($0, /\/pull\/[0-9]+$/) && substr($0, RSTART) in drop { next }
        # A category whose PRs all shipped earlier is left out.
        /^### / { header = $0; next }
        header != "" { if (/^\* /) print header; header = "" }
        { print }'
