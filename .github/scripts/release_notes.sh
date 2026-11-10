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
# whole repo. A PR that an earlier stable release shipped as a cherry-pick is left out
# too. Needs a full clone with tags. Usage: release_notes.sh <tag> <sha> <config>
# [previous_tag] [notes].

set -euo pipefail

USAGE="usage: release_notes.sh <tag> <sha> <config> [previous_tag] [notes]"
TAG=${1:?$USAGE}
SHA=${2:?$USAGE}
CONFIG=${3:?$USAGE}
PREVIOUS_TAG=${4:-}
NOTES=${5:-}

if [ "$(git rev-parse --is-shallow-repository)" = true ]; then
    echo "release_notes.sh needs a full clone, not a shallow one" >&2
    exit 1
fi

PRODUCT=${TAG%%/v*}
VERSION=${TAG#*/v}
VERSION=${VERSION%%-*}

# The stable tags of the product below this release. The commits they hold that SHA
# does not are the cherry-picks of earlier patch releases.
EARLIER=$({
    git tag --list "$PRODUCT/v*" \
        | sed -nE "s#^$PRODUCT/v([0-9]+\.[0-9]+\.[0-9]+)\$#\1#p"
    echo "$VERSION"
} | sort -u -t. -k1,1n -k2,2n -k3,3n | awk -v v="$VERSION" -v p="$PRODUCT" '
    $0 == v { below = 1 }
    !below { print p "/v" $0 }')

# A cherry-pick names its source commit in a `cherry-pick -x` trailer. A squash merge
# drops the trailers, so the commits of the PR it closed are read too.
MESSAGES=""
if [ -n "$EARLIER" ]; then
    # shellcheck disable=SC2086
    for commit in $(git rev-list --no-merges $EARLIER "^$SHA"); do
        MESSAGES+=$(git log -1 --format=%B "$commit")$'\n'
        PR=$(git log -1 --format=%s "$commit" | sed -nE 's/.*\(#([0-9]+)\)$/\1/p')
        if [ -n "$PR" ]; then
            MESSAGES+=$(gh api "repos/$GITHUB_REPOSITORY/pulls/$PR/commits" --paginate \
                --jq '.[].commit.message')$'\n'
        fi
    done
fi

# A source picked from a branch that never merged is not in the clone and has no PR.
SHIPPED=""
SOURCES=$(sed -nE 's/^\(cherry picked from commit ([0-9a-f]+)\)$/\1/p' \
    <<< "$MESSAGES" | sort -u)
for source in $SOURCES; do
    if git cat-file -e "$source^{commit}" 2> /dev/null; then
        SHIPPED+=" $(git log -1 --format=%s "$source" \
            | sed -nE 's/.*\(#([0-9]+)\)$/\1/p')"
    fi
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
