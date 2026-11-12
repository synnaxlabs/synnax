#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Signs a macOS binary with the Developer ID certificate and sends it to Apple for
# notarization. Run import_apple_certificate.sh first: it exports KEYCHAIN_NAME and
# CERT_ID.
#
# Usage: sign_macos_binary.sh <binary> <entitlement>...
#
# Each entitlement is a key below com.apple.security.cs, such as allow-jit.
#
# Required environment variables:
#   KEYCHAIN_NAME     - Keychain that holds the certificate
#   KEYCHAIN_PASSWORD - Password for the keychain
#   CERT_ID           - Signing identity in the keychain
#   APPLE_ID          - Apple ID that submits the binary
#   APPLE_PASSWORD    - App-specific password of the Apple ID
#   APPLE_TEAM_ID     - Developer team of the Apple ID
set -euo pipefail

: "${KEYCHAIN_NAME:?KEYCHAIN_NAME is required}"
: "${KEYCHAIN_PASSWORD:?KEYCHAIN_PASSWORD is required}"
: "${CERT_ID:?CERT_ID is required}"
: "${APPLE_ID:?APPLE_ID is required}"
: "${APPLE_PASSWORD:?APPLE_PASSWORD is required}"
: "${APPLE_TEAM_ID:?APPLE_TEAM_ID is required}"

BINARY=${1:?usage: sign_macos_binary.sh <binary> <entitlement>...}
shift

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

{
    echo '<?xml version="1.0" encoding="UTF-8"?>'
    echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"' \
        '"http://www.apple.com/DTDs/PropertyList-1.0.dtd">'
    echo '<plist version="1.0"><dict>'
    for ENTITLEMENT in "$@"; do
        echo "<key>com.apple.security.cs.$ENTITLEMENT</key><true/>"
    done
    echo '</dict></plist>'
} > "$WORK/entitlements.plist"

codesign --force --options runtime --timestamp \
    --entitlements "$WORK/entitlements.plist" --sign "$CERT_ID" "$BINARY"
# The keychain can lock again before the verification.
security unlock-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN_NAME"
codesign --verify --verbose "$BINARY"

# notarytool accepts only a container format.
zip -j "$WORK/notarize.zip" "$BINARY"
NOTARY=(--apple-id "$APPLE_ID" --password "$APPLE_PASSWORD"
    --team-id "$APPLE_TEAM_ID" --output-format json)
# An upload to the Apple notary service fails at times, so a failed one is sent again.
for ATTEMPT in 1 2 3; do
    echo "Notarization upload attempt $ATTEMPT of 3"
    if SUBMISSION=$(xcrun notarytool submit "$WORK/notarize.zip" "${NOTARY[@]}"); then
        break
    fi
    if [ "$ATTEMPT" = 3 ]; then
        echo "Notarization upload failed after 3 attempts" >&2
        exit 1
    fi
    sleep 10
done
ID=$(jq -r .id <<< "$SUBMISSION")
# A second upload would start again at the end of the queue, so a wait that fails or
# takes too long polls the same submission again.
for ATTEMPT in 1 2 3; do
    echo "Notarization wait attempt $ATTEMPT of 3"
    if RESULT=$(xcrun notarytool wait "$ID" "${NOTARY[@]}" --timeout 6m); then
        break
    fi
    echo "$RESULT"
done
echo "$RESULT"
if [ "$(jq -r .status <<< "$RESULT")" != "Accepted" ]; then
    echo "Notarization of submission $ID did not pass" >&2
    exit 1
fi
