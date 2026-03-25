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
# growth. Deletes files older than MAX_AGE_HOURS (default 2) from Bazel output base,
# Go build/module caches, and old core binaries. Safe to run — Bazel rebuilds from
# S3 remote cache on miss, Go rebuilds on cache miss.

set -euo pipefail

MAX_AGE_HOURS="${1:-2}"
MAX_AGE_MINUTES=$((MAX_AGE_HOURS * 60))
TOTAL_FREED=0

disk_used_mb() {
    df -BM / --output=used 2> /dev/null | tail -1 | tr -d ' M'
}

clean_dir() {
    local dir="$1"
    local label="$2"
    local real_dir
    real_dir=$(realpath "$dir" 2> /dev/null || echo "$dir")
    if [ -d "$real_dir" ]; then
        local before after freed
        before=$(du -sm "$real_dir" 2> /dev/null | cut -f1)
        find "$real_dir" -type f -mmin +"$MAX_AGE_MINUTES" -delete 2> /dev/null || true
        find "$real_dir" -type d -empty -delete 2> /dev/null || true
        after=$(du -sm "$real_dir" 2> /dev/null | cut -f1)
        freed=$((before - after))
        TOTAL_FREED=$((TOTAL_FREED + freed))
        printf "  %-30s %6dMB -> %6dMB  (freed %dMB)\n" "$label" "$before" "$after" "$freed"
    else
        printf "  %-30s skipped (not found)\n" "$label"
    fi
}

DISK_BEFORE=$(disk_used_mb)
echo "=== Build Cache Cleanup (max age: ${MAX_AGE_HOURS}h) ==="
echo ""

echo "Bazel build outputs (preserving external/):"
for bazel_base in /root/.bazel "$HOME/.bazel"; do
    real_base=$(realpath "$bazel_base" 2> /dev/null || echo "$bazel_base")
    [ "$bazel_base" = "$HOME/.bazel" ] && [ "$HOME" = "/root" ] && continue
    if [ -d "$real_base" ]; then
        # Only clean bazel-out/ dirs, skip external/ (fetched deps break if deleted)
        found_outputs=false
        for output_dir in "$real_base"/_bazel_*/*/bazel-out; do
            [ -d "$output_dir" ] && { clean_dir "$output_dir" "$output_dir"; found_outputs=true; }
        done
        $found_outputs || printf "  %-30s no bazel-out dirs found\n" "$bazel_base"
    else
        printf "  %-30s skipped (not found)\n" "$bazel_base"
    fi
done
echo ""

echo "Go build cache:"
clean_dir "/root/.cache/go-build" "/root/.cache/go-build"
if [ "$HOME" != "/root" ] && [ -d "$HOME/.cache/go-build" ]; then
    clean_dir "$HOME/.cache/go-build" "$HOME/.cache/go-build"
fi
echo ""

echo "Go module cache:"
clean_dir "/root/go/pkg/mod/cache" "/root/go/pkg/mod/cache"
if [ "$HOME" != "/root" ] && [ -d "$HOME/go/pkg/mod/cache" ]; then
    clean_dir "$HOME/go/pkg/mod/cache" "$HOME/go/pkg/mod/cache"
fi
echo ""

echo "Old core binaries:"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CORE_DIR="${REPO_ROOT}/core"
if [ -d "$CORE_DIR" ]; then
    count=$(find "$CORE_DIR" -maxdepth 1 -name "synnax-v*" -mmin +"$MAX_AGE_MINUTES" 2> /dev/null | wc -l)
    find "$CORE_DIR" -maxdepth 1 -name "synnax-v*" -mmin +"$MAX_AGE_MINUTES" -delete 2> /dev/null || true
    printf "  %-30s deleted %d old binaries\n" "core/synnax-v*" "$count"
else
    printf "  %-30s skipped (not found)\n" "core/"
fi
echo ""

DISK_AFTER=$(disk_used_mb)
DISK_FREED=$((DISK_BEFORE - DISK_AFTER))

echo "=== Summary ==="
echo "  Cache freed:  ${TOTAL_FREED}MB"
echo "  Disk before:  ${DISK_BEFORE}MB"
echo "  Disk after:   ${DISK_AFTER}MB"
echo "  Disk freed:   ${DISK_FREED}MB"
df -h / --output=source,size,used,avail,pcent 2> /dev/null || df -h /
