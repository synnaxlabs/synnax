#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Reads the notices of removed previews from the preview host INSTANCE, writes each one
# into its pull request's preview comment, then deletes the notices it handled from the
# host. A comment updated after the removal is from a newer deploy, so it stays. A
# notice that fails to post stays on the host for the next run.

set -euo pipefail

: "${INSTANCE:?}" "${GITHUB_REPOSITORY:?}"

marker="<!-- Sticky Pull Request Commentpreview -->"
issues="repos/$GITHUB_REPOSITORY/issues"
dir=$(dirname "${BASH_SOURCE[0]}")

# SSM prints standard output, a tab, then standard error, so only the first field of
# each line can be a notice.
output=$(COMMAND="/usr/local/bin/preview notices" "$dir/run_preview_command.sh")
time='[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z'
notices=$(cut -f1 <<< "$output" | grep -E "^[0-9]+ $time (evicted [0-9]+|expired)$" \
    || true)

handled=()
# gh reads standard input, so the notices come in on another descriptor.
while read -r pr at reason by <&3; do
    if [[ -z $pr ]]; then continue; fi
    if [[ $reason == evicted ]]; then
        why="Evicted to make room for the preview of #$by"
    else
        why="Removed after a long time with no deploy or view"
    fi
    body=$(printf '%s\n\n%s %s\n%s' "### Preview" "$why, so the link no longer works." \
        "To bring it back, remove and add the **preview** label, or push a commit." \
        "$marker")
    comment=$(gh api --paginate "$issues/$pr/comments" --jq \
        ".[] | select(.body | contains(\"$marker\")) | \"\(.id) \(.updated_at)\"")
    read -r id updated <<< "$comment"
    if [[ -n $id && $updated < $at ]]; then
        gh api -X PATCH "$issues/comments/$id" -f body="$body" > /dev/null
    fi
    handled+=("$pr" "$at")
done 3<<< "$notices"

if ((${#handled[@]})); then
    COMMAND="/usr/local/bin/preview ack ${handled[*]}" "$dir/run_preview_command.sh"
fi
