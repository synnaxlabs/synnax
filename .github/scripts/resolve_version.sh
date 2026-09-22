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
# outputs (key=value lines). Usage: resolve_version.sh <product> <bump> [prerelease]
#
# The minor comes from the highest stable product tag reachable from HEAD, the patch
# from the highest stable tag on that minor anywhere in the repo, so a hotfix tag on a
# release branch is never reissued. Candidates never set the base: a candidate counts up
# from the candidates already tagged for the version, and promoting one repeats the same
# bump. The train rule caps a product at one minor ahead of the Core's latest stable.

set -euo pipefail

PRODUCT=${1:?usage: resolve_version.sh <product> <bump> [prerelease]}
BUMP=${2:?usage: resolve_version.sh <product> <bump> [prerelease]}
PRERELEASE=${3:-false}

case "$PRODUCT" in
    console | core | driver) ;;
    *)
        echo "unknown product: $PRODUCT" >&2
        exit 1
        ;;
esac

# Prints the highest X.Y.Z among stable tags of a product on stdin.
highest() {
    sed -nE "s#^$1/v([0-9]+\.[0-9]+\.[0-9]+)\$#\1#p" | sort -t. -k1,1n -k2,2n -k3,3n | tail -1
}

BASE=$(git tag --list "$PRODUCT/v*" --merged HEAD | highest "$PRODUCT")
if [ -z "$BASE" ]; then
    echo "no stable $PRODUCT tag is reachable from HEAD" >&2
    exit 1
fi
IFS=. read -r MAJOR MINOR _ <<< "$BASE"

CURRENT=$(git tag --list "$PRODUCT/v$MAJOR.$MINOR.*" | highest "$PRODUCT")
IFS=. read -r _ _ PATCH <<< "$CURRENT"

case "$BUMP" in
    patch)
        NEXT="$MAJOR.$MINOR.$((PATCH + 1))"
        ;;
    minor)
        NEXT="$MAJOR.$((MINOR + 1)).0"
        ;;
    *)
        echo "unknown bump: $BUMP (expected patch or minor)" >&2
        exit 1
        ;;
esac
IFS=. read -r NEXT_MAJOR NEXT_MINOR _ <<< "$NEXT"

CORE=$(git tag --list 'core/v*' | highest core)
if [ -z "$CORE" ]; then
    echo "no stable core tag exists; the train rule needs one" >&2
    exit 1
fi
IFS=. read -r CORE_MAJOR CORE_MINOR _ <<< "$CORE"
if [ "$NEXT_MAJOR" != "$CORE_MAJOR" ] || [ "$NEXT_MINOR" -gt "$((CORE_MINOR + 1))" ]; then
    echo "train rule: $PRODUCT $NEXT is more than one minor ahead of core $CORE" >&2
    exit 1
fi

VERSION=$NEXT
if [ "$PRERELEASE" = "true" ]; then
    LAST_RC=$(git tag --list "$PRODUCT/v$NEXT-rc.*" \
        | sed -nE "s#^$PRODUCT/v$NEXT-rc\.([0-9]+)\$#\1#p" | sort -n | tail -1)
    VERSION="$NEXT-rc.$((${LAST_RC:-0} + 1))"
fi

echo "$PRODUCT $BASE -> $VERSION (bump $BUMP, core $CORE)" >&2
echo "version=$VERSION"
echo "tag=$PRODUCT/v$VERSION"
echo "minor=$NEXT_MAJOR.$NEXT_MINOR"
echo "previous_tag=$PRODUCT/v$BASE"
