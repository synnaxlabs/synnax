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
# whole repo. Usage: release_notes.sh <tag> <sha> <config> [previous_tag] [notes].

set -euo pipefail

USAGE="usage: release_notes.sh <tag> <sha> <config> [previous_tag] [notes]"
TAG=${1:?$USAGE}
SHA=${2:?$USAGE}
CONFIG=${3:?$USAGE}
PREVIOUS_TAG=${4:-}
NOTES=${5:-}

ARGS=(-f tag_name="$TAG" -f target_commitish="$SHA")
ARGS+=(-f configuration_file_path="$CONFIG")
if [ -n "$PREVIOUS_TAG" ]; then ARGS+=(-f previous_tag_name="$PREVIOUS_TAG"); fi
if [ -n "$NOTES" ]; then printf '%s\n\n' "$NOTES"; fi
gh api "repos/$GITHUB_REPOSITORY/releases/generate-notes" "${ARGS[@]}" --jq .body \
    | awk '
        /^## New Contributors/ { skip = 1 }
        /^\*\*Full Changelog\*\*/ { skip = 0 }
        !skip'
