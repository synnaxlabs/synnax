#!/bin/bash
# Generate OS matrix for GitHub Actions workflow based on input flags

BUILD_WINDOWS=$1
BUILD_MACOS=$2
BUILD_UBUNTU=$3
SIGN_BINARIES=$4

OS_LIST=""

if [ "$BUILD_WINDOWS" = "true" ]; then
    OS_LIST="{\"os\":\"windows-build-bot\",\"os-name\":\"windows\",\"executable\":\".exe\"}"
fi

if [ "$BUILD_MACOS" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    # If signing, use specific runner-2; otherwise any macos-build-bot
    if [ "$SIGN_BINARIES" = "true" ]; then
        OS_LIST="${OS_LIST}{\"os\":\"macos-build-bot-2\",\"os-name\":\"macos\",\"executable\":\"\"}"
    else
        OS_LIST="${OS_LIST}{\"os\":\"macos-build-bot\",\"os-name\":\"macos\",\"executable\":\"\"}"
    fi
fi

if [ "$BUILD_UBUNTU" = "true" ]; then
    [ -n "$OS_LIST" ] && OS_LIST="$OS_LIST,"
    OS_LIST="${OS_LIST}{\"os\":\"ubuntu-build-bot\",\"os-name\":\"linux\",\"executable\":\"\"}"
fi

echo "[${OS_LIST}]"
