#!/bin/bash
# Generate OS matrix for GitHub Actions workflow based on input flags

BUILD_WINDOWS=$1
BUILD_MACOS=$2
BUILD_UBUNTU=$3
BUILD_UBUNTU_22_04=$4

OS_LIST=""

if [ "$BUILD_WINDOWS" = "true" ]; then
    OS_LIST="{\"os\":\"windows-latest\",\"artifact-name\":\"synnax-core-windows\",\"binary-suffix\":\"windows.exe\"}"
fi

if [ "$BUILD_MACOS" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"macos-latest\",\"artifact-name\":\"synnax-core-macos\",\"binary-suffix\":\"macos\"}"
fi

if [ "$BUILD_UBUNTU" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"ubuntu-latest\",\"artifact-name\":\"synnax-core-linux\",\"binary-suffix\":\"linux\"}"
fi

if [ "$BUILD_UBUNTU_22_04" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"ubuntu-22.04\",\"artifact-name\":\"synnax-core-nilinuxrt\",\"binary-suffix\":\"nilinuxrt\"}"
fi

echo "[${OS_LIST}]"
