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
# during a clone or checkout can leave an unfinished submodule on a self-hosted runner,
# and git submodule update fails on it. The script deletes those to clone them again.
# Usage: update_submodules.sh

set -euo pipefail

# A clone killed in its fetch leaves a HEAD that does not resolve. One killed in its
# checkout leaves no index, or a stale index.lock. foreach enters each submodule after
# running its command, so the deletes wait until the list is done.
# shellcheck disable=SC2016 # foreach expands the variables in each submodule.
unfinished=$(git submodule foreach --recursive --quiet '
    gitdir=$(git rev-parse --absolute-git-dir)
    if ! git rev-parse --quiet --verify HEAD > /dev/null ||
        [ ! -f "$gitdir/index" ] || [ -e "$gitdir/index.lock" ]; then
        printf "%s\t%s\n" "$toplevel/$sm_path" "$gitdir"
    fi')
if [ -n "$unfinished" ]; then
    while IFS=$'\t' read -r worktree gitdir; do
        echo "::warning::Deleting unfinished submodule $worktree"
        rm -rf "$worktree" "$gitdir"
    done <<< "$unfinished"
fi
git submodule update --init --recursive
