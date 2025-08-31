#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# run-test-conductor-unix.sh
# Runs the integration test conductor on Unix systems (Linux/macOS)
# Used by GitHub Actions workflow: test.integration.yaml

set -euo pipefail

echo "Running integration test conductor..."

# Change to the integration test directory
cd integration/test/py

# Set up PATH for Poetry
export PATH="$HOME/.local/bin:$PATH"

# Run the test conductor
poetry run test-conductor --name test-conductor-unix --sequence testcases/basic_tests.json

echo "Integration test conductor completed successfully"