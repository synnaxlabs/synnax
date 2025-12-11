#!/bin/bash
# Generate OS matrix for GitHub Actions workflow based on input flags

BUILD_WINDOWS=$1
BUILD_MACOS=$2
BUILD_UBUNTU=$3
BUILD_UBUNTU_22_04=$4

OS_LIST=""

if [ "$BUILD_WINDOWS" = "true" ]; then
    OS_LIST="{\"os\":\"windows-build-bot\",\"artifact-name\":\"windows-synnax-core\",\"binary-suffix\":\"windows\",\"executable\":\".exe\"}"
fi

if [ "$BUILD_MACOS" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"macos-build-bot\",\"artifact-name\":\"macos-synnax-core\",\"binary-suffix\":\"macos\",\"executable\":\"\"}"
fi

if [ "$BUILD_UBUNTU" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"ubuntu-build-bot\",\"artifact-name\":\"linux-synnax-core\",\"binary-suffix\":\"linux\",\"executable\":\"\"}"
fi

if [ "$BUILD_UBUNTU_22_04" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"ubuntu-2204-build-bot\",\"artifact-name\":\"nilinuxrt-synnax-core\",\"binary-suffix\":\"nilinuxrt\",\"executable\":\"\"}"
fi

echo "[${OS_LIST}]"
