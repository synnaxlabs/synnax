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
# Usage: resolve_version.sh <product> <prerelease> <ref> [bump] [core]
#
# Every product releases only from main or a release branch: release/X.Y, the one
# branch of train X.Y, or release/desktop-X.Y, which releases only Desktop.
#
# Console, Core, and Driver: main resolves the minor after the product's newest stable
# tag. A release branch resolves the next patch on X.Y, or X.Y.0 when the product has
# no tag on a train that another product opened from main. Both fail when a stable tag
# of the product names HEAD, and a release branch fails when it lacks the tag it counts
# from.
#
# ts and py share the Core's minor, since the clients connect only to a Core with the
# same major.minor. From main, they release only with the Core: core is the stable
# version the same run releases, and they take the next patch on its minor. From a
# release branch, they take the next patch on X.Y, and fail when no Core is released on
# X.Y. For a stable run, a package tag at HEAD on that train resolves to itself, so a
# release branch cut at a run's commit finishes that run's publish.
#
# Desktop: main takes a minor or major bump on the highest Desktop tag reachable from
# HEAD. release/desktop-X.Y takes only a patch, the next one on X.Y, and needs the
# newest stable tag on X.Y. The first Desktop release must be a major bump from 0.0.0,
# so it outranks the 1.0.0 candidates.
#
# A candidate counts up from the candidates already tagged for the version. The train
# rule caps every product but Desktop at one minor ahead of the Core's newest stable.
# latest is true when no stable tag of the product outranks the result. previous_tag is
# the highest stable product tag below the result, reachable or not.

set -euo pipefail

USAGE="usage: resolve_version.sh <product> <prerelease> <ref> [bump] [core]"
PRODUCT=${1:?$USAGE}
PRERELEASE=${2:?$USAGE}
REF=${3:-}
BUMP=${4:-}
RUN_CORE=${5:-}

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

# Fails when HEAD lacks the train's newest tag, CURRENT, since a patch built without it
# would drop its fix.
require_last_patch() {
    if [ -n "$CURRENT" ] \
        && ! git merge-base --is-ancestor "$PRODUCT/v$CURRENT" HEAD; then
        echo "$PRODUCT/v$CURRENT is not on $REF. Release $PRODUCT from the branch" \
            "that holds it" >&2
        exit 1
    fi
}

# Prints X.Y when the ref is release/, then the prefix $1, then X.Y.
release_train() {
    if [[ $REF =~ ^release/$1((0|[1-9][0-9]*)\.(0|[1-9][0-9]*))$ ]]; then
        echo "${BASH_REMATCH[1]}"
    fi
}

case "$PRODUCT" in
    console | core | driver | ts | py)
        if [ -n "$BUMP" ]; then
            echo "$PRODUCT takes no bump; the ref sets its version" >&2
            exit 1
        fi
        TRAIN=$(release_train "")
        if [ -n "$(release_train desktop-)" ]; then
            echo "$REF releases only Desktop. Leave $PRODUCT out of the run" >&2
            exit 1
        fi
        if [ -z "$TRAIN" ] && [ "$REF" != main ]; then
            echo "releases run only from main or release/X.Y, not '$REF'" >&2
            exit 1
        fi
        if [ -z "$TRAIN" ] && [[ $PRODUCT =~ ^(ts|py)$ ]]; then
            if [ -z "$RUN_CORE" ]; then
                echo "$PRODUCT releases from main only with the Core. To release it" \
                    "alone, run from release/X.Y" >&2
                exit 1
            fi
            if ! [[ $RUN_CORE =~ ^([0-9]+\.[0-9]+)\.[0-9]+$ ]]; then
                echo "core $RUN_CORE is not a stable version" >&2
                exit 1
            fi
            TRAIN=${BASH_REMATCH[1]}
        fi
        if [[ $PRODUCT =~ ^(ts|py)$ ]] && [[ $RUN_CORE != "$TRAIN".[0-9]* ]] \
            && [ -z "$(git tag --list "core/v$TRAIN.*" | highest core)" ]; then
            echo "no Core is released on $TRAIN, and $PRODUCT shares the Core's minor" >&2
            exit 1
        fi
        if [ -n "$TRAIN" ]; then
            BASE=$TRAIN
            CURRENT=$(git tag --list "$PRODUCT/v$TRAIN.*" | highest "$PRODUCT")
            NEXT=$(next_patch "$TRAIN")
            if [ "$REF" != main ]; then require_last_patch; fi
            if [ "$REF" != main ] && [ -z "$CURRENT" ] \
                && [ -z "$(git tag --list "console/v$TRAIN.*" "core/v$TRAIN.*" \
                    "driver/v$TRAIN.*" | highest '[a-z]+')" ]; then
                echo "no stable release exists on $TRAIN; open a train from main" >&2
                exit 1
            fi
        else
            CURRENT=$(git tag --list "$PRODUCT/v*" | highest "$PRODUCT")
            BASE=$CURRENT
            if [ -z "$BASE" ]; then
                echo "no stable $PRODUCT tag exists" >&2
                exit 1
            fi
            IFS=. read -r MAJOR MINOR _ <<< "$BASE"
            NEXT="$MAJOR.$((MINOR + 1)).0"
        fi
        # A package tag at HEAD keeps its version, so a rerun finishes the publish. Any
        # other product would release the same commit twice.
        if [[ $PRODUCT =~ ^(ts|py)$ ]]; then
            if [ "$PRERELEASE" != true ] && [ -n "$CURRENT" ] \
                && [ "$(git rev-parse "$PRODUCT/v$CURRENT^{commit}")" \
                    = "$(git rev-parse HEAD)" ]; then
                NEXT=$CURRENT
            fi
        else
            SHIPPED=$(git tag --points-at HEAD --list "$PRODUCT/v*" \
                | highest "$PRODUCT")
            if [ -n "$SHIPPED" ]; then
                echo "$PRODUCT/v$SHIPPED already names HEAD. Rerun the failed jobs," \
                    "or leave $PRODUCT out of the run" >&2
                exit 1
            fi
        fi
        ;;
    desktop)
        TRAIN=$(release_train desktop-)
        if [ "$REF" != main ] && [ -z "$TRAIN" ]; then
            echo "desktop releases only from main or release/desktop-X.Y, not '$REF'" >&2
            exit 1
        fi
        if [ -n "$TRAIN" ] && [ "$BUMP" != patch ]; then
            echo "$REF releases only patches, not a $BUMP bump" >&2
            exit 1
        fi
        if [ -z "$TRAIN" ] && [ "$BUMP" = patch ]; then
            echo "desktop releases patches only from release/desktop-X.Y" >&2
            exit 1
        fi
        if [ -n "$TRAIN" ]; then
            CURRENT=$(git tag --list "desktop/v$TRAIN.*" | highest desktop)
            if [ -z "$CURRENT" ]; then
                echo "no stable desktop tag exists on $TRAIN; open a train from main" >&2
                exit 1
            fi
            require_last_patch
        fi
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
            patch) NEXT=$(next_patch "$TRAIN") ;;
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
