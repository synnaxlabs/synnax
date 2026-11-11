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

# Prints the sources a commit names in `cherry-pick -x` trailers. A squash merge drops
# the trailers, so a commit with none reads the commits of the PR its subject names.
sources() {
    local trailer='s/^\(cherry picked from commit ([0-9a-f]+)\)$/\1/p'
    local found pr
    found=$(git log -1 --format=%B "$1" | sed -nE "$trailer") || return
    pr=$(subject_pr "$1") || return
    if [ -n "$found" ] || [ -z "$pr" ]; then
        echo "$found"
        return
    fi
    "$GH_API" "repos/$GITHUB_REPOSITORY/pulls/$pr/commits" --paginate \
        --jq '.[].commit.message' | sed -nE "$trailer"
}

# The commits the previous tag holds that SHA does not are its cherry-picks. A source
# off SHA is itself a pick, such as one from an older train's branch, so its own
# sources are read too.
SOURCES=""
PICKS=""
if [ -n "$PREVIOUS_TAG" ]; then
    PICKS=$(git rev-list --no-merges "$PREVIOUS_TAG" "^$SHA")
fi
while [ -n "$PICKS" ]; do
    FOUND=""
    for commit in $PICKS; do
        FOUND+=$(sources "$commit")$'\n'
    done
    PICKS=""
    for source in $FOUND; do
        if [[ " $SOURCES " == *" $source "* ]]; then continue; fi
        SOURCES+=" $source"
        if git cat-file -e "$source^{commit}" 2> /dev/null \
            && ! git merge-base --is-ancestor "$source" "$SHA"; then
            PICKS+=" $source"
        fi
    done
done

# A source with a PR in its subject is that PR's squash merge. GitHub resolves the
# merged PR of any other source, such as a commit of a PR branch. That PR counts as
# shipped only when all of its commits were picked. A source picked from a commit that
# was never pushed has no PR.
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
            --jq '.[].sha')
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
