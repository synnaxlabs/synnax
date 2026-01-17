#!/bin/bash
# Generate OS matrix for GitHub Actions workflow based on input flags

BUILD_WINDOWS=$1
BUILD_MACOS=$2
BUILD_UBUNTU=$3

OS_LIST=""

if [ "$BUILD_WINDOWS" = "true" ]; then
    OS_LIST="{\"os\":\"windows-build-bot\",\"os-name\":\"windows\",\"executable\":\".exe\"}"
fi

# TODO: Revert to macos-build-bot when self-hosted runner is available
if [ "$BUILD_MACOS" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"macos-latest\",\"os-name\":\"macos\",\"executable\":\"\"}"
fi

if [ "$BUILD_UBUNTU" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"ubuntu-build-bot\",\"os-name\":\"linux\",\"executable\":\"\"}"
fi

echo "[${OS_LIST}]"
