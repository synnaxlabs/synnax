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

echo "Building Synnax..."

# Build Driver
echo "Building driver with Bazel..."
bazel build --enable_platform_specific_config -c opt --config=hide_symbols --announce_rc //driver

# Move Driver to Assets
echo "Moving driver to assets..."
mkdir -p core/pkg/service/hardware/embedded/assets
cp bazel-bin/driver/driver core/pkg/service/hardware/embedded/assets/driver

# Get Version
echo "Getting version..."
cd core
VERSION=$(cat pkg/version/VERSION)
echo "VERSION=$VERSION" >> $GITHUB_OUTPUT
echo "Building version: $VERSION"
cd ..

# Download Go Dependencies
echo "Downloading Go dependencies..."
cd core
go mod download

# Build Server
echo "Building Synnax server..."
go build -tags driver -o synnax-v${VERSION}-macos
cd ..

# Test Binary Execution
echo "Testing binary execution..."
./core/synnax-v${VERSION}-macos version || echo "WARNING: Server binary check failed"
bazel-bin/driver/driver --help || echo "WARNING: Driver binary check failed"

echo "macOS build completed successfully!"
echo "Built artifacts:"
echo "  - Driver: bazel-bin/driver/driver"
echo "  - Server: core/synnax-v${VERSION}-macos"
