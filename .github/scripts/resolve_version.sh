#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Resolves the next version of a product from its Git tags and prints GitHub Actions
# outputs (key=value lines).
# Usage: resolve_version.sh <product> <prerelease> [ref] [train] [bump]
#
# Console, Core, and Driver: a release/<product>-X.Y ref resolves the next patch on X.Y,
# or X.Y.0 when the train has no tag. Any other ref resolves the minor after the
# product's newest stable tag. Packages (ts, py): the next patch on the train, or X.Y.0.
# The train's newest tag at HEAD resolves to itself, so a rerun finishes its publish.
# Desktop: the bump applies to the highest Desktop tag reachable from HEAD, and a patch
# counts from the highest tag on that minor. The first Desktop release must be a major
# bump from 0.0.0, so it outranks the 1.0.0 candidates.
#
# A candidate counts up from the candidates already tagged for the version. The train
# rule caps every product but Desktop at one minor ahead of the Core's newest stable.
# latest is true when no stable tag of the product outranks the result. previous_tag is
# the highest stable product tag below the result, reachable or not.

set -euo pipefail

USAGE="usage: resolve_version.sh <product> <prerelease> [ref] [train] [bump]"
PRODUCT=${1:?$USAGE}
PRERELEASE=${2:?$USAGE}
REF=${3:-}
TRAIN=${4:-}
BUMP=${5:-}

# Prints the highest X.Y.Z among stable tags of a product on stdin.
highest() {
    sed -nE "s#^$1/v([0-9]+\.[0-9]+\.[0-9]+)\$#\1#p" | sort -t. -k1,1n -k2,2n -k3,3n | tail -1
}

# Prints the next patch on train $1, or $1.0 when the train has no stable tag.
next_patch() {
    local current patch
    current=$(git tag --list "$PRODUCT/v$1.*" | highest "$PRODUCT")
    if [ -z "$current" ]; then
        echo "$1.0"
        return
    fi
    IFS=. read -r _ _ patch <<< "$current"
    echo "$1.$((patch + 1))"
}

case "$PRODUCT" in
    console | core | driver)
        if [ -n "$TRAIN$BUMP" ]; then
            echo "$PRODUCT takes no train or bump; the ref sets its version" >&2
            exit 1
        fi
        if [[ $REF =~ ^release/(console|core|driver)-([0-9]+\.[0-9]+)$ ]]; then
            BASE=$REF
            NEXT=$(next_patch "${BASH_REMATCH[2]}")
        else
            BASE=$(git tag --list "$PRODUCT/v*" | highest "$PRODUCT")
            if [ -z "$BASE" ]; then
                echo "no stable $PRODUCT tag exists" >&2
                exit 1
            fi
            IFS=. read -r MAJOR MINOR _ <<< "$BASE"
            NEXT="$MAJOR.$((MINOR + 1)).0"
        fi
        ;;
    ts | py)
        if ! [[ $TRAIN =~ ^[0-9]+\.[0-9]+$ ]]; then
            echo "malformed train: '$TRAIN' (expected X.Y)" >&2
            exit 1
        fi
        if [ -n "$BUMP" ]; then
            echo "$PRODUCT takes no bump; it releases the next patch on its train" >&2
            exit 1
        fi
        BASE="train $TRAIN"
        NEXT=$(next_patch "$TRAIN")
        CURRENT=$(git tag --list "$PRODUCT/v$TRAIN.*" | highest "$PRODUCT")
        # A tag at HEAD claims its version for this commit, so a rerun finishes it.
        if [ -n "$CURRENT" ] && [ "$(git rev-parse "$PRODUCT/v$CURRENT^{commit}")" \
            = "$(git rev-parse HEAD)" ]; then
            NEXT=$CURRENT
        fi
        ;;
    desktop)
        BASE=$(git tag --list "$PRODUCT/v*" --merged HEAD | highest "$PRODUCT")
        if [ -z "$BASE" ]; then
            if [ "$BUMP" != major ]; then
                echo "no stable desktop tag exists, so bump must be major, not $BUMP" >&2
                exit 1
            fi
            BASE=0.0.0
        fi
        IFS=. read -r MAJOR MINOR _ <<< "$BASE"
        case "$BUMP" in
            patch) NEXT=$(next_patch "$MAJOR.$MINOR") ;;
            minor) NEXT="$MAJOR.$((MINOR + 1)).0" ;;
            major) NEXT="$((MAJOR + 1)).0.0" ;;
            *)
                echo "unknown bump: '$BUMP' (expected patch, minor, or major)" >&2
                exit 1
                ;;
        esac
        ;;
    *)
        echo "unknown product: $PRODUCT" >&2
        exit 1
        ;;
esac
IFS=. read -r NEXT_MAJOR NEXT_MINOR _ <<< "$NEXT"

CORE=""
if [ "$PRODUCT" != desktop ]; then
    CORE=$(git tag --list 'core/v*' | highest core)
    if [ -z "$CORE" ]; then
        echo "no stable core tag exists; the train rule needs one" >&2
        exit 1
    fi
    IFS=. read -r CORE_MAJOR CORE_MINOR _ <<< "$CORE"
    if [ "$NEXT_MAJOR" != "$CORE_MAJOR" ] \
        || [ "$NEXT_MINOR" -gt "$((CORE_MINOR + 1))" ]; then
        echo "train rule: $PRODUCT $NEXT is more than one minor ahead of core $CORE" >&2
        exit 1
    fi
fi

VERSION=$NEXT
if [ "$PRERELEASE" = "true" ]; then
    LAST_RC=$(git tag --list "$PRODUCT/v$NEXT-rc.*" \
        | sed -nE "s#^$PRODUCT/v$NEXT-rc\.([0-9]+)\$#\1#p" | sort -n | tail -1)
    VERSION="$NEXT-rc.$((${LAST_RC:-0} + 1))"
fi

RANKED=$({
    git tag --list "$PRODUCT/v*"
    echo "$PRODUCT/v$NEXT"
} | sed -nE "s#^$PRODUCT/v([0-9]+\.[0-9]+\.[0-9]+)\$#\1#p" \
    | sort -u -t. -k1,1n -k2,2n -k3,3n)
PREVIOUS=$(grep -B1 -xF "$NEXT" <<< "$RANKED" | head -1)
PREVIOUS_TAG=""
if [ "$PREVIOUS" != "$NEXT" ]; then PREVIOUS_TAG="$PRODUCT/v$PREVIOUS"; fi

# A hotfix on an old train must not become the product's latest release.
HIGHEST=$(tail -1 <<< "$RANKED")
LATEST=false
if [ "$HIGHEST" = "$NEXT" ]; then LATEST=true; fi

echo "$PRODUCT $BASE -> $VERSION${CORE:+ (core $CORE)}" >&2
echo "version=$VERSION"
echo "tag=$PRODUCT/v$VERSION"
echo "minor=$NEXT_MAJOR.$NEXT_MINOR"
echo "previous_tag=$PREVIOUS_TAG"
echo "latest=$LATEST"
