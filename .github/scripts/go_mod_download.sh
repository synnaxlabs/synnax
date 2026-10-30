#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Downloads the dependencies of each Go module directory. A failed download runs again
# up to three attempts in total, because the module proxy drops streams at times.
# Usage: go_mod_download.sh <directory>...

set -euo pipefail

if [ "$#" -eq 0 ]; then
    echo "usage: go_mod_download.sh <directory>..." >&2
    exit 2
fi

attempts=3
for directory in "$@"; do
    attempt=1
    until go -C "$directory" mod download; do
        if [ "$attempt" -eq "$attempts" ]; then
            echo "::error::go mod download failed $attempts times in $directory"
            exit 1
        fi
        delay=$((attempt * 15))
        echo "::warning::go mod download failed in $directory, retrying in ${delay}s"
        sleep "$delay"
        attempt=$((attempt + 1))
    done
done
