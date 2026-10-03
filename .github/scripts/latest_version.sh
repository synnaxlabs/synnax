#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the highest version of a product on a train, or nothing when the train has
# none. Usage: latest_version.sh <product> <major.minor> [candidates]. With
# candidates=true, X.Y.Z-rc.N tags rank below the stable X.Y.Z but above X.Y.(Z-1).

set -euo pipefail

PRODUCT=${1:?usage: latest_version.sh <product> <major.minor> [candidates]}
MINOR=${2:?usage: latest_version.sh <product> <major.minor> [candidates]}
CANDIDATES=${3:-false}

git tag --list "$PRODUCT/v$MINOR.*" \
    | sed -nE "s#^$PRODUCT/v$MINOR\.([0-9]+)(-rc\.([0-9]+))?\$#\1\t\3\t$MINOR.\1\2#p" \
    | awk -F '\t' -v OFS='\t' -v candidates="$CANDIDATES" '
        $2 == "" { $2 = 999999 }
        $2 != 999999 && candidates != "true" { next }
        { print }' \
    | sort -t "$(printf '\t')" -k1,1n -k2,2n | tail -1 | awk -F '\t' '{ print $3 }'
