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
# and a plain git submodule update fails on it. The script repairs those first.
# Usage: update_submodules.sh

set -euo pipefail

# The scan cannot see a half-deleted submodule, so the deletes go through a journal. A
# job cancelled during them leaves the journal, and the next run finishes them first.
journal="$(git rev-parse --absolute-git-dir)/unfinished-submodules"

delete_unfinished() {
    [ -f "$journal" ] || return 0
    while IFS=$'\t' read -r worktree gitdir; do
        echo "::warning::Deleting unfinished submodule $worktree"
        rm -rf "$worktree" "$gitdir"
    done < "$journal"
    rm -f "$journal"
}

delete_unfinished
# A clone killed in its fetch leaves a HEAD that does not resolve, and a checkout killed
# with no time to clean up leaves a stale index.lock. Those are cloned again. foreach
# enters each submodule after running its command, so the deletes wait for the list.
# shellcheck disable=SC2016 # foreach expands the variables in each submodule.
unfinished=$(git submodule foreach --recursive --quiet '
    gitdir=$(git rev-parse --absolute-git-dir)
    if ! git rev-parse --quiet --verify HEAD > /dev/null ||
        [ -e "$gitdir/index.lock" ]; then
        printf "%s\t%s\n" "$toplevel/$sm_path" "$gitdir"
    fi')
if [ -n "$unfinished" ]; then
    printf '%s\n' "$unfinished" > "$journal.tmp"
    mv "$journal.tmp" "$journal"
    delete_unfinished
fi
# --force checks out with -f, which overwrites the files a killed checkout left behind.
git submodule update --init --recursive --force
