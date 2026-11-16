#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the Docker channel, latest or next, that a Core version may move, or nothing
# when a Core tag on stdin outranks the version. A candidate ranks below its stable
# version, and latest counts only stable tags. Usage: image_channel.sh <channel>
# <version> < tags, where each line ends in a core/vX.Y.Z[-rc.N] tag.

set -euo pipefail

CHANNEL=${1:-}
VERSION=${2:?usage: image_channel.sh <channel> <version>}

rank() {
    sed -nE 's#.*core/v([0-9]+)\.([0-9]+)\.([0-9]+)(-rc\.([0-9]+))?$#\1 \2 \3 \5#p' \
        | awk -v channel="$1" '$4 == "" { $4 = 999999 }
            channel == "latest" && $4 != 999999 { next }
            { print }'
}

[ -n "$CHANNEL" ] || exit 0
HIGHEST=$(rank "$CHANNEL" | sort -k1,1n -k2,2n -k3,3n -k4,4n | tail -n 1)
OURS=$(rank "" <<< "core/v$VERSION")
if [ "$HIGHEST" = "$OURS" ]; then echo "$CHANNEL"; fi
