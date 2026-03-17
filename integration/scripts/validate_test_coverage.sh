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

set -euo pipefail

TESTS_DIR="integration/tests"
EXEMPT="example"

# Matrix definition (single source of truth)
# Format: "name:target" where target can be comma-separated file prefixes
MATRIX_ENTRIES=(
    "arc:arc,control,latency"
    "console:console"
    "driver:driver"
)

covered=""
for entry in "${MATRIX_ENTRIES[@]}"; do
    target="${entry#*:}"
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
json='{"include":['
first=true
for entry in "${MATRIX_ENTRIES[@]}"; do
    name="${entry%%:*}"
    target="${entry#*:}"
    if [ "$first" = true ]; then
        first=false
    else
        json+=","
    fi
    json+="{\"name\":\"$name\",\"target\":\"$target\"}"
done
json+=']}'

echo "Matrix: $json"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
    echo "TEST_MATRIX=$json" >> "$GITHUB_OUTPUT"
fi
