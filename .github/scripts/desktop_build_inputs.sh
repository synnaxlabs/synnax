#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the inputs of a Synnax Desktop build as GitHub Actions outputs (key=value
# lines).
# Usage: desktop_build_inputs.sh [core] [driver]
#
# core and driver are the versions that the same run releases, or empty.
#
# version is the one version that the bundled Core, the clients, and a Driver built
# from source carry: the Core version of the run, else the newest stable Core tag
# reachable from HEAD.
#
# driver_release is the Driver tag of the run when the run releases the Driver and the
# Core on one minor. The Driver accepts a Core only with its own major.minor, so the
# Driver of the run pairs only with the Core version of the run. When it is empty,
# Desktop builds the Driver from source at HEAD with the version above.

set -euo pipefail

CORE=${1:-}
DRIVER=${2:-}

minor() {
    cut -d. -f1,2 <<< "$1"
}

VERSION=$CORE
if [ -z "$VERSION" ]; then
    VERSION=$(git tag --list --merged HEAD 'core/v*' \
        | sed -nE 's#^core/v([0-9]+\.[0-9]+\.[0-9]+)$#\1#p' \
        | sort -t. -k1,1n -k2,2n -k3,3n | tail -n 1)
    if [ -z "$VERSION" ]; then
        echo "no stable core tag is reachable from HEAD" >&2
        exit 1
    fi
fi

DRIVER_RELEASE=""
if [ -n "$CORE" ] && [ -n "$DRIVER" ] && [ "$(minor "$CORE")" = "$(minor "$DRIVER")" ]; then
    DRIVER_RELEASE="driver/v$DRIVER"
fi

echo "version=$VERSION"
echo "driver_release=$DRIVER_RELEASE"
