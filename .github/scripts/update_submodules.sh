#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Checks out every submodule at the commit its superproject records. A job cancelled
# during a clone can leave a submodule whose HEAD does not resolve on a self-hosted
# runner. git submodule update fails on it, so the script deletes it to clone it again.
# Usage: update_submodules.sh

set -euo pipefail

# foreach enters each submodule after running its command, so delete after listing.
# shellcheck disable=SC2016 # foreach expands the variables in each submodule.
broken=$(git submodule foreach --recursive --quiet \
    'git rev-parse --quiet --verify HEAD > /dev/null ||
        printf "%s\t%s\n" "$toplevel/$sm_path" "$(git rev-parse --absolute-git-dir)"')
if [ -n "$broken" ]; then
    while IFS=$'\t' read -r worktree gitdir; do
        echo "::warning::Deleting half-cloned submodule $worktree"
        rm -rf "$worktree" "$gitdir"
    done <<< "$broken"
fi
git submodule update --init --recursive
