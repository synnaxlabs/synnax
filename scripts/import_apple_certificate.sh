#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Imports an Apple Developer Certificate into a permanent, per-runner keychain.
# Creates the keychain if it doesn't exist, reuses it if it does.
#
# Required environment variables:
#   APPLE_CERTIFICATE          - Base64-encoded .p12 certificate
#   APPLE_CERTIFICATE_PASSWORD - Password for the .p12 certificate
#   KEYCHAIN_PASSWORD          - Password for the keychain
#   KEYCHAIN_NAME              - Keychain filename (e.g. build-actions-runner-2.keychain)
#   GITHUB_ENV                 - Path to GitHub Actions env file
set -euo pipefail

: "${APPLE_CERTIFICATE:?APPLE_CERTIFICATE is required}"
: "${APPLE_CERTIFICATE_PASSWORD:?APPLE_CERTIFICATE_PASSWORD is required}"
: "${KEYCHAIN_PASSWORD:?KEYCHAIN_PASSWORD is required}"
: "${KEYCHAIN_NAME:?KEYCHAIN_NAME is required}"
: "${GITHUB_ENV:?GITHUB_ENV is required}"

echo "=== Keychain Debug Info ==="
echo "Target keychain: $KEYCHAIN_NAME"
echo "Current keychains:"
security list-keychains -d user
KEYCHAIN_COUNT=$(security list-keychains -d user | wc -l | tr -d ' ')
echo "Total keychains: $KEYCHAIN_COUNT"
echo "=========================="

echo "$APPLE_CERTIFICATE" | base64 --decode > certificate.p12

# Create keychain if it doesn't exist
if ! security list-keychains -d user | grep -q "$KEYCHAIN_NAME"; then
    echo "Creating new keychain: $KEYCHAIN_NAME"
    security create-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN_NAME"
    CURRENT_KEYCHAINS=$(security list-keychains -d user | tr -d '"' | tr '\n' ' ')
    security list-keychains -d user -s $CURRENT_KEYCHAINS "$KEYCHAIN_NAME"
else
    echo "Found existing keychain: $KEYCHAIN_NAME (reusing)"
fi

security default-keychain -s "$KEYCHAIN_NAME"
security unlock-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN_NAME"
security set-keychain-settings "$KEYCHAIN_NAME"

# Import certificate (may already exist in keychain, that's ok)
echo "Importing certificate into $KEYCHAIN_NAME..."
security import certificate.p12 -k "$KEYCHAIN_NAME" \
    -P "$APPLE_CERTIFICATE_PASSWORD" -T /usr/bin/codesign 2> /dev/null || true
security set-key-partition-list -S apple-tool:,apple:,codesign: \
    -s -k "$KEYCHAIN_PASSWORD" "$KEYCHAIN_NAME"

rm -f certificate.p12

# Extract signing identity and export to GITHUB_ENV
security find-identity -v -p codesigning "$KEYCHAIN_NAME"
CERT_INFO=$(security find-identity -v -p codesigning "$KEYCHAIN_NAME" \
    | grep "Developer ID Application")
CERT_ID=$(echo "$CERT_INFO" | awk -F'"' '{print $2}')
echo "Certificate imported: $CERT_ID"

echo "KEYCHAIN_NAME=$KEYCHAIN_NAME" >> "$GITHUB_ENV"
echo "CERT_ID=$CERT_ID" >> "$GITHUB_ENV"
