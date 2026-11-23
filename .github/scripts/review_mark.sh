#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints "<mark> <time>" for pull request PR: "human" while it has the review/human
# label, "none" otherwise, and the Unix time just before the labels were read. The
# preview host keeps the newest mark, so runs that finish out of order cannot undo one.
# It fails when the labels cannot be read, rather than guess.

set -euo pipefail

: "${PR:?}" "${GITHUB_REPOSITORY:?}"

at=$(date +%s)
labels=$(gh api "repos/$GITHUB_REPOSITORY/issues/$PR/labels" --jq '.[].name')
mark=none
if grep -qx "review/human" <<< "$labels"; then mark=human; fi
echo "$mark $at"
