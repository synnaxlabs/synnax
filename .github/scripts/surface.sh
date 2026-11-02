#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Compares one user-facing surface between two checkouts of the repository.
#
#   surface.sh <surface> <base-dir> <head-dir> <out-dir>
#
# Each checkout makes its report with its own generator. The script writes
# <out-dir>/status (same, changed, or error) and <out-dir>/diff (the unified diff of the
# two reports, or the tail of the failed generator's output). It exits non-zero on
# error.

set -euo pipefail

surface=$1
base=$(cd "$2" && pwd)
head=$(cd "$3" && pwd)
mkdir -p "$4"
out=$(cd "$4" && pwd)

report() {
    case "$surface" in
        oracle)
            (cd oracle && go build -buildvcs=false -o "$1/.oracle-surface" .) >&2 \
                && ./.oracle-surface surface
            ;;
        client-ts)
            pnpm install --frozen-lockfile >&2 \
                && pnpm exec turbo build --filter='@synnaxlabs/client^...' >&2 \
                && client/ts/scripts/surface.sh
            ;;
        client-py)
            uv run --project scripts scripts/surface_py.py client/py synnax
            ;;
        core)
            cd core && go run -buildvcs=false ./cmd/surface/dump
            ;;
        driver)
            uv run --project scripts scripts/surface_driver.py .
            ;;
        *)
            echo "unknown surface: $surface" >&2
            return 2
            ;;
    esac
}

for side in base head; do
    dir=$base
    [ "$side" = head ] && dir=$head
    if ! (cd "$dir" && report "$dir") > "$out/$side.txt" 2> "$out/$side.log"; then
        echo error > "$out/status"
        {
            echo "The $side report failed:"
            tail -n 40 "$out/$side.log"
        } > "$out/diff"
        cat "$out/diff" >&2
        exit 1
    fi
done

if diff -u --label base --label head "$out/base.txt" "$out/head.txt" > "$out/diff"; then
    echo same > "$out/status"
else
    echo changed > "$out/status"
fi
cat "$out/status"
