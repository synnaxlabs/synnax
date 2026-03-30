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
# growth. Deletes files older than MAX_AGE_HOURS (default 2) from Bazel build outputs,
# Go build/module caches, and old core binaries. Safe to run — Bazel rebuilds from
# S3 remote cache on miss, Go rebuilds on cache miss.
#
# This script must never fail the build, so we use set +e (best-effort cleanup).

set +e

MAX_AGE_HOURS="${1:-2}"
MAX_AGE_MINUTES=$((MAX_AGE_HOURS * 60))
TOTAL_FREED=0

disk_used_mb() {
    df -BM / --output=used 2> /dev/null | tail -1 | tr -d ' M'
}

clean_dir() {
    local dir="$1"
    local label="$2"
    if [ ! -d "$dir" ]; then
        printf "  %-35s skipped (not found)\n" "$label"
        return
    fi
    local before after freed
    before=$(du -sm "$dir" 2> /dev/null | cut -f1 || echo 0)
    before=${before:-0}
    find "$dir" -type f -mmin +"$MAX_AGE_MINUTES" -delete 2> /dev/null
    find "$dir" -type d -empty -delete 2> /dev/null
    after=$(du -sm "$dir" 2> /dev/null | cut -f1 || echo 0)
    after=${after:-0}
    freed=$((before - after))
    TOTAL_FREED=$((TOTAL_FREED + freed))
    printf "  %-35s %6dMB -> %6dMB  (freed %dMB)\n" "$label" "$before" "$after" "$freed"
}

DISK_BEFORE=$(disk_used_mb)
echo "=== Build Cache Cleanup (max age: ${MAX_AGE_HOURS}h) ==="
echo ""

echo "Bazel (clean only if no recent builds within ${MAX_AGE_HOURS}h):"
REPO_ROOT="$(cd "$(dirname "$0")/../.." 2> /dev/null && pwd)"
BAZEL_BASE="/root/.bazel"
if [ -d "$BAZEL_BASE" ]; then
    recent=$(find "$BAZEL_BASE" -name "bazel-out" -type d -exec find {} -type f -mmin -"$MAX_AGE_MINUTES" -print -quit \; 2> /dev/null)
    if [ -n "$recent" ]; then
        printf "  %-35s skipped (recent build found)\n" "bazel clean"
    elif [ -d "$REPO_ROOT" ]; then
        before_bazel=$(du -sm "$BAZEL_BASE" 2> /dev/null | cut -f1 || echo 0)
        before_bazel=${before_bazel:-0}
        cd "$REPO_ROOT" && bazel clean 2> /dev/null
        after_bazel=$(du -sm "$BAZEL_BASE" 2> /dev/null | cut -f1 || echo 0)
        after_bazel=${after_bazel:-0}
        freed_bazel=$((before_bazel - after_bazel))
        TOTAL_FREED=$((TOTAL_FREED + freed_bazel))
        printf "  %-35s %6dMB -> %6dMB  (freed %dMB)\n" "bazel clean" "$before_bazel" "$after_bazel" "$freed_bazel"
    else
        printf "  %-35s skipped (repo not found)\n" "bazel clean"
    fi
else
    printf "  %-35s skipped (not found)\n" "$BAZEL_BASE"
fi
echo ""

echo "Go build cache:"
clean_dir "/root/.cache/go-build" "/root/.cache/go-build"
[ "$HOME" != "/root" ] && clean_dir "$HOME/.cache/go-build" "$HOME/.cache/go-build"
echo ""

echo "Go module cache:"
clean_dir "/root/go/pkg/mod/cache" "/root/go/pkg/mod/cache"
[ "$HOME" != "/root" ] && clean_dir "$HOME/go/pkg/mod/cache" "$HOME/go/pkg/mod/cache"
echo ""

echo "Old core binaries:"
REPO_ROOT="$(cd "$(dirname "$0")/../.." 2> /dev/null && pwd)"
CORE_DIR="${REPO_ROOT}/core"
if [ -d "$CORE_DIR" ]; then
    count=$(find "$CORE_DIR" -maxdepth 1 -name "synnax-v*" -mmin +"$MAX_AGE_MINUTES" 2> /dev/null | wc -l)
    find "$CORE_DIR" -maxdepth 1 -name "synnax-v*" -mmin +"$MAX_AGE_MINUTES" -delete 2> /dev/null
    printf "  %-35s deleted %d old binaries\n" "core/synnax-v*" "$count"
else
    printf "  %-35s skipped (not found)\n" "core/"
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
