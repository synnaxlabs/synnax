#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

set -euo pipefail

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <path>"
    exit 1
fi

directory="$1"
git_root=$(git rev-parse --show-toplevel)

if [ ! -d "$git_root/$directory" ]; then
    echo "Error: Path '$directory' does not exist relative to repo root."
    exit 1
fi

(cd "$git_root/$directory" && go generate ./...)

# --intent-to-add marks new files as added without staging their content, so they show
# up in `git diff` alongside modifications. Without this, files generated for the first
# time would slip past the diff check.
git -C "$git_root" add --intent-to-add "$directory"

"$git_root/scripts/update_copyrights.sh" "$git_root/$directory" > /dev/null

if ! git -C "$git_root" diff --exit-code -- "$directory"; then
    echo "::error::go generate produced changes. Run 'go generate ./...' and './scripts/update_copyrights.sh $directory' locally, then commit the result."
    exit 1
fi
