#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Defines the integration test matrix, validates that all *_tests.json files are
# covered, and outputs the matrix JSON for GitHub Actions.
#
# Usage:
#   integration/scripts/validate_test_coverage.sh
#
# Outputs (GitHub Actions):
#   TEST_MATRIX — JSON object consumable by strategy.matrix via fromJSON()
#   BOT_MATRIX — the same suites for a run with no hosted runners. A hosted entry is
#   one job there with no args, as its shards and args fit a hosted runner.

set -euo pipefail

TESTS_DIR="integration/tests"
# Test prefixes exempt from the integration matrix — these run in dedicated workflows.
EXEMPT="example"

# Matrix definition (single source of truth)
# Format: "name:target:hosted:shards:args" where target can be comma-separated file
# prefixes. hosted is true when the suite runs on a GitHub-hosted runner. A suite that
# needs NI software, the timer precision of a test bot, or more than 4 CPUs is not
# hosted. An entry with more than one shard becomes the jobs name-1 to name-N, each
# running its part of target. args are more flags for the test conductor. The Driver
# suite waits on its devices more than it computes, so it runs its sequences on four
# Cores at once. A hosted runner has the CPU for one browser, so the Console suite
# runs as shards.
MATRIX_ENTRIES=(
    "arc:arc,control:true:1:"
    "console:console:true:5:--browsers 1"
    "driver:driver,latency:false:1:--cores 4 --browsers 1"
)

covered=""
for entry in "${MATRIX_ENTRIES[@]}"; do
    IFS=: read -r _ target _ <<< "$entry"
    for prefix in $(echo "$target" | tr ',' ' '); do
        covered="$covered $prefix"
    done
done

# Forward validation: every *_tests.json on disk must be covered
missing=""
for file in "$TESTS_DIR"/*_tests.json; do
    [ -f "$file" ] || continue
    prefix=$(basename "$file" | sed 's/_tests\.json$//')

    if echo "$EXEMPT" | grep -qw "$prefix"; then
        continue
    fi

    if ! echo "$covered" | grep -qw "$prefix"; then
        missing="$missing $prefix"
    fi
done

if [ -n "$missing" ]; then
    echo "::error::Test files not covered by the matrix:$missing"
    echo "Update MATRIX_ENTRIES in integration/scripts/validate_test_coverage.sh"
    exit 1
fi

# Reverse validation: every prefix in the matrix must have a *_tests.json file
invalid=""
for prefix in $covered; do
    if [ ! -f "$TESTS_DIR/${prefix}_tests.json" ]; then
        invalid="$invalid $prefix"
    fi
done

if [ -n "$invalid" ]; then
    echo "::error::Matrix references missing test files:$invalid"
    echo "Update MATRIX_ENTRIES in integration/scripts/validate_test_coverage.sh"
    exit 1
fi

echo "All test files covered:"
for prefix in $covered; do
    echo "  ✓ ${prefix}_tests.json"
done
if [ -n "$EXEMPT" ]; then
    echo "Exempt: $EXEMPT"
fi

# Build Matrix Output
job() {
    printf '{"name":"%s","target":"%s","hosted":%s,"args":"%s"}' "$@"
}

test_jobs=()
bot_jobs=()
for entry in "${MATRIX_ENTRIES[@]}"; do
    IFS=: read -r name target hosted shards args <<< "$entry"
    if [ "$hosted" = true ]; then
        bot_jobs+=("$(job "$name" "$target" "$hosted" "")")
    else
        bot_jobs+=("$(job "$name" "$target" "$hosted" "$args")")
    fi
    if [ "$shards" -eq 1 ]; then
        test_jobs+=("$(job "$name" "$target" "$hosted" "$args")")
        continue
    fi
    for shard in $(seq 1 "$shards"); do
        shard_args="$args --shard $shard/$shards"
        test_jobs+=("$(job "$name-$shard" "$target" "$hosted" "$shard_args")")
    done
done

matrix() {
    local IFS=,
    echo "{\"include\":[$*]}"
}
test_matrix=$(matrix "${test_jobs[@]}")
bot_matrix=$(matrix "${bot_jobs[@]}")

echo "Matrix: $test_matrix"
echo "Bot matrix: $bot_matrix"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
    echo "TEST_MATRIX=$test_matrix" >> "$GITHUB_OUTPUT"
    echo "BOT_MATRIX=$bot_matrix" >> "$GITHUB_OUTPUT"
fi
