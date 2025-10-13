#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Verify and display build configuration

VERSION=$1
BUILD_WINDOWS=$2
BUILD_MACOS=$3
BUILD_UBUNTU=$4
BUILD_UBUNTU_22_04=$5
BUILD_DRIVER=$6
BUILD_CONSOLE=$7
BUILD_CORE=$8

# Verify that at least one OS is selected
if [ "$BUILD_WINDOWS" != "true" ] && [ "$BUILD_MACOS" != "true" ] && [ "$BUILD_UBUNTU" != "true" ] && [ "$BUILD_UBUNTU_22_04" != "true" ]; then
    echo "Error: No operating systems selected. Please select at least one OS to build for."
    exit 1
fi

# Verify that at least one binary is selected
if [ "$BUILD_DRIVER" != "true" ] && [ "$BUILD_CONSOLE" != "true" ] && [ "$BUILD_CORE" != "true" ]; then
    echo "Error: No binaries selected. Please select at least one binary to build."
    exit 1
fi

echo "╔═══════════════════════════════════════╗"
echo "║          BUILD CONFIGURATION          ║"
echo "╚═══════════════════════════════════════╝"
echo ""
echo "Version: $VERSION"
echo ""
echo "Operating Systems:"
if [ "$BUILD_WINDOWS" = "true" ]; then
    echo "  ✓ Windows"
else
    echo "  X Windows"
fi
if [ "$BUILD_MACOS" = "true" ]; then
    echo "  ✓ macOS"
else
    echo "  X macOS"
fi
if [ "$BUILD_UBUNTU" = "true" ]; then
    echo "  ✓ Ubuntu"
else
    echo "  X Ubuntu"
fi
if [ "$BUILD_UBUNTU_22_04" = "true" ]; then
    echo "  ✓ Ubuntu 22.04 (NI Linux RT)"
else
    echo "  X Ubuntu 22.04 (NI Linux RT)"
fi
echo ""
echo "Binaries:"
if [ "$BUILD_DRIVER" = "true" ]; then
    echo "  ✓ Driver"
else
    echo "  X Driver"
fi
if [ "$BUILD_CONSOLE" = "true" ]; then
    echo "  ✓ Console"
else
    echo "  X Console"
fi
if [ "$BUILD_CORE" = "true" ]; then
    echo "  ✓ Core"
else
    echo "  X Core"
fi
echo ""
echo "─────────────────────────────────────────"
