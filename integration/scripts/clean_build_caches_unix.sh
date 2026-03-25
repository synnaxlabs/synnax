#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Cleans stale build caches on self-hosted Linux runners to prevent unbounded disk
# growth. Deletes files older than MAX_AGE_HOURS (default 6) from Bazel output base,
# Go build/module caches, and old core binaries. Safe to run — Bazel rebuilds from
# S3 remote cache on miss, Go rebuilds on cache miss.

set -euo pipefail

MAX_AGE_HOURS="${1:-2}"
MAX_AGE_MINUTES=$((MAX_AGE_HOURS * 60))

echo "Cleaning build caches older than ${MAX_AGE_HOURS}h..."

clean_dir() {
    local dir="$1"
    local label="$2"
    if [ -d "$dir" ]; then
        local before
        before=$(du -sm "$dir" 2> /dev/null | cut -f1)
        find "$dir" -type f -mmin +"$MAX_AGE_MINUTES" -delete 2> /dev/null || true
        find "$dir" -type d -empty -delete 2> /dev/null || true
        local after
        after=$(du -sm "$dir" 2> /dev/null | cut -f1)
        echo "  ${label}: ${before}MB -> ${after}MB (freed $((before - after))MB)"
    else
        echo "  ${label}: not found, skipping"
    fi
}

echo "--- Bazel output base ---"
clean_dir "$HOME/.bazel" "~/.bazel"
clean_dir "/root/.bazel" "/root/.bazel"

echo "--- Go build cache ---"
clean_dir "$HOME/.cache/go-build" "~/.cache/go-build"
clean_dir "/root/.cache/go-build" "/root/.cache/go-build"

echo "--- Go module cache ---"
clean_dir "$HOME/go/pkg/mod/cache" "~/go/pkg/mod/cache"
clean_dir "/root/go/pkg/mod/cache" "/root/go/pkg/mod/cache"

echo "--- Old core binaries ---"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CORE_DIR="${REPO_ROOT}/core"
if [ -d "$CORE_DIR" ]; then
    count=$(find "$CORE_DIR" -maxdepth 1 -name "synnax-v*" -mmin +"$MAX_AGE_MINUTES" 2> /dev/null | wc -l)
    find "$CORE_DIR" -maxdepth 1 -name "synnax-v*" -mmin +"$MAX_AGE_MINUTES" -delete 2> /dev/null || true
    echo "  core/synnax-v*: deleted $count old binaries"
else
    echo "  core/: not found, skipping"
fi

echo "--- Disk usage ---"
df -h / --output=source,size,used,avail,pcent 2> /dev/null || df -h /

echo "✅ Build cache cleanup complete"
