#!/bin/bash
# Generate OS matrix for GitHub Actions workflow based on input flags

BUILD_UBUNTU=$1
BUILD_WINDOWS=$2
BUILD_MACOS=$3

OS_LIST=""

if [ "$BUILD_UBUNTU" = "true" ]; then
    OS_LIST="{\"os\":\"ubuntu-latest\",\"artifact-name\":\"synnax-core-linux\",\"binary-suffix\":\"linux\"}"
fi

if [ "$BUILD_WINDOWS" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"windows-latest\",\"artifact-name\":\"synnax-core-windows\",\"binary-suffix\":\"windows.exe\"}"
fi

if [ "$BUILD_MACOS" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"macos-latest\",\"artifact-name\":\"synnax-core-macos\",\"binary-suffix\":\"macos\"}"
fi

echo "[${OS_LIST}]"
