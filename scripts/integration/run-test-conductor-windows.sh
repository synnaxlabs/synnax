#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# run-test-conductor-windows.sh
# Runs the test conductor on Windows using bash
# Used by GitHub Actions workflow: test.integration.yaml

set -euo pipefail

echo "Running test conductor on Windows..."

# Add Poetry to PATH for current session
export PATH="$HOME/.local/bin:$PATH"

# Change to test directory
cd "integration/test/py"

# Run test conductor
poetry run test-conductor --name test-conductor-windows --sequence testcases/basic_tests.json

echo "Test conductor completed"