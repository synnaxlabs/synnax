#!/bin/sh

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Builds the Core binary of the Docker image into the directory $1. Run it in core/.
# DRIVER_ENABLED and CONSOLE_ENABLED embed the Driver and the Console, and VERSION,
# GIT_COMMIT, and BUILD_DATE go into the version package.

set -eu

TAGS=""
if [ "${DRIVER_ENABLED:-false}" = true ]; then
    TAGS=driver
fi
if [ "${CONSOLE_ENABLED:-false}" = true ]; then
    TAGS="${TAGS:+$TAGS,}console"
fi
PKG=github.com/synnaxlabs/synnax/pkg/version
LDFLAGS="-w -s -X $PKG.version=${VERSION:-} -X $PKG.gitCommit=${GIT_COMMIT:-unknown}"
LDFLAGS="$LDFLAGS -X $PKG.buildDate=${BUILD_DATE:-unknown}"
CGO_ENABLED=0 go build -trimpath -buildvcs=false -tags "$TAGS" -ldflags "$LDFLAGS" \
    -o "$1/" .
