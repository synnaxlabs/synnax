#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Fails unless the GitHub Actions checks on a commit passed. Usage: verify_checks.sh
# <sha> <run_id>. Runs of the given workflow run are ignored, so a release can verify
# its own commit. Every merged commit carries the CI merge-group run, so the commit's
# own check runs are the whole verdict.

set -euo pipefail

USAGE="usage: verify_checks.sh <sha> <run_id>"
SHA=${1:?$USAGE}
RUN_ID=${2:?$USAGE}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}

RUNS=$(gh api "repos/${REPO}/commits/${SHA}/check-runs?per_page=100" \
    --paginate \
    --jq '.check_runs[]
        | select(.app.slug == "github-actions")
        | select(.html_url | contains("/actions/runs/'"$RUN_ID"'/") | not)
        | "\(.status)\t\(.conclusion)\t\(.name)"')

if [ -z "$RUNS" ]; then
    echo "no check runs found on $SHA" >&2
    exit 1
fi

PENDING=$(awk -F'\t' '$1 != "completed" { print "  " $3 " (" $1 ")" }' <<< "$RUNS")
if [ -n "$PENDING" ]; then
    printf 'checks still running on %s:\n%s\n' "$SHA" "$PENDING" >&2
    exit 1
fi

FAILED=$(awk -F'\t' '$2 != "success" && $2 != "skipped" && $2 != "neutral" \
    { print "  " $3 " (" $2 ")" }' <<< "$RUNS")
if [ -n "$FAILED" ]; then
    printf 'checks failed on %s:\n%s\n' "$SHA" "$FAILED" >&2
    exit 1
fi

echo "$(wc -l <<< "$RUNS" | tr -d ' ') checks passed on $SHA"
