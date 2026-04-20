#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Cleans build caches on self-hosted Linux runners to prevent unbounded disk growth.
# - Bazel: always runs `bazel clean` (unconditional — remote cache serves next build)
# - Go/binaries: deletes oldest files first until MIN_FREE_GB of disk space is available
#
# Usage: clean_build_caches_unix.sh [MIN_FREE_GB]
#
# This script must never fail the build, so we use set +e (best-effort cleanup).

set +e

MIN_FREE_GB="${1:-25}"
MIN_FREE_MB=$((MIN_FREE_GB * 1024))
TOTAL_FREED=0
BATCH_SIZE=100

REPO_ROOT="$(cd "$(dirname "$0")/../.." 2> /dev/null && pwd)"

get_free_mb() {
    df -BM / --output=avail 2> /dev/null | tail -1 | tr -d ' M'
}

get_used_mb() {
    df -BM / --output=used 2> /dev/null | tail -1 | tr -d ' M'
}

has_enough_space() {
    [ "$(get_free_mb)" -ge "$MIN_FREE_MB" ] 2> /dev/null
}

DISK_BEFORE=$(get_used_mb)
echo "=== Build Cache Cleanup (target: ${MIN_FREE_GB}GB free) ==="
echo "  Current free space: $(get_free_mb)MB (target: ${MIN_FREE_MB}MB)"
echo ""

# --- Bazel clean (unconditional) ---
echo "Bazel clean:"
    BAZEL_BASE=$(bazel info output_user_root 2>/dev/null || echo "/root/.bazel")
if [ -d "$BAZEL_BASE" ] && [ -d "$REPO_ROOT" ]; then
    before_bazel=$(du -sm "$BAZEL_BASE" 2> /dev/null | cut -f1 || echo 0)
    before_bazel=${before_bazel:-0}
    (cd "$REPO_ROOT" && bazel clean 2>&1) || echo "  bazel clean failed (exit $?)"
    after_bazel=$(du -sm "$BAZEL_BASE" 2> /dev/null | cut -f1 || echo 0)
    after_bazel=${after_bazel:-0}
    freed_bazel=$((before_bazel - after_bazel))
    TOTAL_FREED=$((TOTAL_FREED + freed_bazel))
    printf "  %-35s %6dMB -> %6dMB  (freed %dMB)\n" \
        "bazel clean" "$before_bazel" "$after_bazel" "$freed_bazel"
else
    printf "  %-35s skipped (not found)\n" "bazel clean"
fi
echo ""

# --- Check if we already have enough space after bazel clean ---
if has_enough_space; then
    echo "Free space $(get_free_mb)MB >= target ${MIN_FREE_MB}MB — skipping cache cleanup."
    echo ""
    DISK_AFTER=$(get_used_mb)
    DISK_FREED=$((DISK_BEFORE - DISK_AFTER))
    echo "=== Summary ==="
    echo "  Cache freed:  ${TOTAL_FREED}MB"
    echo "  Disk before:  ${DISK_BEFORE}MB"
    echo "  Disk after:   ${DISK_AFTER}MB"
    echo "  Disk freed:   ${DISK_FREED}MB"
    df -h / --output=source,size,used,avail,pcent 2> /dev/null || df -h /
    exit 0
fi

# --- Collect all cache files sorted oldest-first, delete until target met ---
echo "Deleting oldest cache files until ${MIN_FREE_GB}GB free..."

CACHE_DIRS=()
for dir in \
    "/root/.cache/go-build" \
    "/root/go/pkg/mod/cache" \
    "$HOME/.cache/go-build" \
    "$HOME/go/pkg/mod/cache"; do
    [ -d "$dir" ] && CACHE_DIRS+=("$dir")
done

CORE_DIR="${REPO_ROOT}/core"
DELETED=0

while IFS=' ' read -r _mtime filepath; do
    [ -z "$filepath" ] && continue
    size=$(stat --format='%s' "$filepath" 2> /dev/null || echo 0)
    rm -f "$filepath" 2> /dev/null
    DELETED=$((DELETED + 1))
    TOTAL_FREED=$((TOTAL_FREED + size / 1048576))
    if [ $((DELETED % BATCH_SIZE)) -eq 0 ]; then
        if has_enough_space; then
            break
        fi
    fi
done < <(
    {
        if [ ${#CACHE_DIRS[@]} -gt 0 ]; then
            find "${CACHE_DIRS[@]}" -type f 2> /dev/null
        fi
        if [ -d "$CORE_DIR" ]; then
            find "$CORE_DIR" -maxdepth 1 -name "synnax-v*" -type f 2> /dev/null
        fi
    } | xargs -r stat --format='%Y %n' 2> /dev/null | sort -n
)

printf "  Deleted %d files\n" "$DELETED"

# Clean up empty directories left behind
for dir in "${CACHE_DIRS[@]}"; do
    find "$dir" -type d -empty -delete 2> /dev/null
done
echo ""

CURRENT_FREE=$(get_free_mb)
if [ "$CURRENT_FREE" -lt "$MIN_FREE_MB" ]; then
    echo "WARNING: Caches exhausted but free space ${CURRENT_FREE}MB" \
        "< target ${MIN_FREE_MB}MB"
    echo ""
fi

DISK_AFTER=$(get_used_mb)
DISK_FREED=$((DISK_BEFORE - DISK_AFTER))

echo "=== Summary ==="
echo "  Cache freed:  ~${TOTAL_FREED}MB"
echo "  Disk before:  ${DISK_BEFORE}MB"
echo "  Disk after:   ${DISK_AFTER}MB"
echo "  Disk freed:   ${DISK_FREED}MB"
df -h / --output=source,size,used,avail,pcent 2> /dev/null || df -h /
