#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

VERSION=$1
PLATFORM_WINDOWS=$2
PLATFORM_MACOS=$3
PLATFORM_UBUNTU=$4
PLATFORM_DOCKER=$5
BUILD_DRIVER=$6
BUILD_CONSOLE=$7
BUILD_CONSOLE_TAURI=$8
BUILD_CORE=$9

# Verify that at least one platform is selected
if [ "$PLATFORM_WINDOWS" != "true" ] && [ "$PLATFORM_MACOS" != "true" ] && [ "$PLATFORM_UBUNTU" != "true" ] && [ "$PLATFORM_DOCKER" != "true" ]; then
    echo "Error: No platforms selected. Please select at least one platform to build for."
    exit 1
fi

# Verify that at least one build target is selected
if [ "$BUILD_DRIVER" != "true" ] && [ "$BUILD_CONSOLE" != "true" ] && [ "$BUILD_CONSOLE_TAURI" != "true" ] && [ "$BUILD_CORE" != "true" ]; then
    echo "Error: No build targets selected. Please select at least one target to build."
    exit 1
fi

echo "╔═══════════════════════════════════════╗"
echo "║          BUILD CONFIGURATION          ║"
echo "╚═══════════════════════════════════════╝"
echo ""
echo "Version: $VERSION"
echo ""
echo "Platforms:"
if [ "$PLATFORM_WINDOWS" = "true" ]; then
    echo "  ✓ Windows"
else
    echo "  X Windows"
fi
if [ "$PLATFORM_MACOS" = "true" ]; then
    echo "  ✓ macOS"
else
    echo "  X macOS"
fi
if [ "$PLATFORM_UBUNTU" = "true" ]; then
    echo "  ✓ Ubuntu (includes NI Linux RT driver)"
else
    echo "  X Ubuntu"
fi
if [ "$PLATFORM_DOCKER" = "true" ]; then
    echo "  ✓ Docker"
else
    echo "  X Docker"
fi
echo ""
echo "Builds:"
if [ "$BUILD_DRIVER" = "true" ]; then
    echo "  ✓ Driver"
else
    echo "  X Driver"
fi
if [ "$BUILD_CONSOLE" = "true" ]; then
    echo "  ✓ Console Web Assets"
else
    echo "  X Console Web Assets"
fi
if [ "$BUILD_CONSOLE_TAURI" = "true" ]; then
    echo "  ✓ Console Desktop App"
else
    echo "  X Console Desktop App"
fi
if [ "$BUILD_CORE" = "true" ]; then
    echo "  ✓ Core"
else
    echo "  X Core"
fi
echo ""
echo "─────────────────────────────────────────"
