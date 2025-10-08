#!/bin/bash
# Generate OS matrix for GitHub Actions workflow based on input flags

BUILD_UBUNTU=$1
BUILD_WINDOWS=$2
BUILD_MACOS=$3

OS_LIST=""

if [ "$BUILD_UBUNTU" = "true" ]; then
    OS_LIST="\"ubuntu-latest\""
fi

if [ "$BUILD_WINDOWS" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}\"windows-latest\""
fi

if [ "$BUILD_MACOS" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}\"macos-latest\""
fi

echo "matrix=[${OS_LIST}]"
