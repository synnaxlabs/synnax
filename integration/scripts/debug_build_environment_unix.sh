#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

set -euo pipefail

echo "Build failed - debugging info:"

echo "=== Go version ==="
go version || echo "Go not found"

echo "=== Bazel version ==="
bazel version || bazelisk version || echo "Bazel/Bazelisk not found"

echo "=== Disk space ==="
df -h || echo "Cannot check disk space"

echo "=== Build outputs ==="
ls -la bazel-bin/driver/ || echo "No driver build output"
ls -la synnax/ | grep synnax-v || echo "No server build output"

echo "Debug information collection completed"
