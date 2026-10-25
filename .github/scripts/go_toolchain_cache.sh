#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the tool cache path and the Actions cache key of the Go toolchain that a go.mod
# selects, as GitHub step outputs.
# Usage: go_toolchain_cache.sh <go.mod>

set -euo pipefail

version=$(sed -n 's/^go //p' "$1")
echo "path=${RUNNER_TOOL_CACHE}/go/${version}"
echo "key=go-toolchain-${RUNNER_OS}-${RUNNER_ARCH}-${version}"
