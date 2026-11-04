#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Deletes every draft release under a tag, and fails without deleting any when a
# published release holds the tag. Deletes by ID, because `gh release delete <tag>` can
# pick the published release when one shares the tag with a draft.
# Usage: clear_release_drafts.sh <tag>

set -euo pipefail

TAG=${1:?usage: clear_release_drafts.sh <tag>}
API="repos/$GITHUB_REPOSITORY/releases"

HOLDERS=$(gh api "$API" --paginate --jq '.[] | {id, tag_name, draft}' \
    | jq -c --arg tag "$TAG" 'select(.tag_name == $tag)')

PUBLISHED=$(jq 'select(.draft | not) | .id' <<< "$HOLDERS")
if [ -n "$PUBLISHED" ]; then
    echo "published release $PUBLISHED already holds $TAG" >&2
    exit 1
fi

for ID in $(jq .id <<< "$HOLDERS"); do
    gh api "$API/$ID" --method DELETE
done
