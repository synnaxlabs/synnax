#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Downloads the artifacts of a workflow run that match a pattern, replacing the
# directory. A transient failure runs again with backoff for up to ten minutes, because
# GitHub's artifact storage throttles egress under load. Any other failure, such as a
# missing artifact, exits at once. The caller's gh_cmd overrides the gh executable.
# Usage: download_run_artifacts.sh <run-id> <pattern> <directory>

set -euo pipefail

if [ "$#" -ne 3 ]; then
    echo "usage: download_run_artifacts.sh <run-id> <pattern> <directory>" >&2
    exit 2
fi
run_id=$1
pattern=$2
directory=$3

gh_cmd=${gh_cmd:-gh}
retry_window=600
max_delay=120
transient='HTTP (408|429|5[0-9]{2})|rate limit|timeout|timed out|connection reset'
transient+='|connection refused|unexpected EOF|stream error|no such host'

download=("$gh_cmd" run download "$run_id" -p "$pattern" --dir "$directory")
delay=15
waited=0
while true; do
    rm -rf "$directory"
    if output=$("${download[@]}" 2>&1); then
        echo "$output"
        exit 0
    fi
    echo "$output"
    if ! grep -Eiq "$transient" <<< "$output"; then
        echo "::error::Failed to download $pattern from run $run_id"
        exit 1
    fi
    # Jitter keeps the jobs of a throttled merge queue from retrying in step.
    wait_seconds=$((delay + RANDOM % (delay / 2 + 1)))
    if [ "$((waited + wait_seconds))" -gt "$retry_window" ]; then
        echo "::error::Download of $pattern still failing after ${waited}s of retries"
        exit 1
    fi
    echo "::warning::Download of $pattern failed, retrying in ${wait_seconds}s"
    sleep "$wait_seconds"
    waited=$((waited + wait_seconds))
    delay=$((delay * 2 > max_delay ? max_delay : delay * 2))
done
