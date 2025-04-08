#!/bin/sh

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# This debug script runs run-integration.sh with a few differences:
#   - Test data is not deleted after a run
#   - Test is run in verbose mode by default

if [ $# -lt 1 ]; then
    echo "Usage: $0 <test config name>"
    exit 1
fi


# Build the server binary
echo "--Compiling"
(cd ../synnax && go build -tags="development" -o ../integration/bin/synnax.exe)

# Run the Go program with the provided arguments
go run . -v "$1"

# Capture the exit code of the Go program
exit_code=$?

# Check if the exit code is not 0
if [ $exit_code -ne 0 ]; then
    echo "Test failed, see stdout for more info"
    exit $exit_code
fi

printf "\n"

# Find the last occurrence of "Test Started" and print everything after that
last_occurrence=$(grep -n "Test Started" ./timing.log | tail -n 1 | cut -d: -f1)

# Check if the last occurrence is found
if [ -z "$last_occurrence" ]; then
    echo "No occurrences of 'Test Started' found in timing.log"
    exit 1
fi

# We want everything after the last occurrence
result=$(sed -n "${last_occurrence},\$p" ./timing.log)

echo "$result"

# Check if any assertions failed
if echo "$result" | grep -q "FAIL!!"; then
    echo "Integration test failed: at least one assertion failed"
    exit 1
fi
