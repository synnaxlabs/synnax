#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs Bazel, and recovers once from an external repo that cannot be loaded.
#
# Self-hosted runners keep the Bazel output base warm between jobs. Each external repo
# there is a symlink into the shared repo contents cache, so a deleted cache entry
# leaves a repo that Bazel records as fetched but cannot load. Every later job on that
# machine then fails in the same way. Wipe the output base and the contents cache, then
# run the command again.
#
# Usage: run_bazel.sh test --test_output=all //x/cpp/...

set -uo pipefail

if [ "$#" -eq 0 ]; then
    echo "Usage: $0 <bazel arguments...>"
    exit 1
fi

log=$(mktemp)
trap 'rm -f "$log"' EXIT

bazel "$@" 2>&1 | tee "$log"
status=${PIPESTATUS[0]}
if [ "$status" -eq 0 ]; then
    exit 0
fi

# Retry an unloadable repo only. A compile error or a failed test ends the job here.
corrupt_repo="Error loading '@@|no such package '@@|@@[^']*' is invalid because"
if ! grep -qE "$corrupt_repo" "$log"; then
    exit "$status"
fi

echo "::warning::Bazel could not load an external repo. Wiping the output base."
repository_cache=$(bazel info repository_cache 2> /dev/null || true)
bazel clean --expunge
if [ -n "$repository_cache" ]; then
    rm -rf "${repository_cache:?}/contents"
fi
bazel "$@"
