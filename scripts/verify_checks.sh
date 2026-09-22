#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Fails unless the GitHub Actions checks on a commit have passed. Usage:
# verify_checks.sh <sha> <run_id> <branch>. Every check run on the commit must have
# passed; runs of the given workflow run are ignored, so a release workflow can verify
# the commit it runs on. A path filter skips a workflow on a push that leaves its paths
# alone, so a commit with only passing check runs can still sit on a broken branch. The
# newest push run of every test, lint, and check workflow on the branch must therefore
# have passed too. A workflow with no push run on the branch is skipped: a hotfix
# branch cut from a tag only runs the workflows a cherry-pick touches. Runs from the
# checkout root, which supplies the workflow files.

set -euo pipefail

USAGE="usage: verify_checks.sh <sha> <run_id> <branch>"
SHA=${1:?$USAGE}
RUN_ID=${2:?$USAGE}
BRANCH=${3:?$USAGE}
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

shopt -s nullglob
FILES=(.github/workflows/{test,lint,check}.*.yaml)
WORKFLOWS=""
if [ "${#FILES[@]}" -gt 0 ]; then
    WORKFLOWS=$(grep -lE '^  push:' "${FILES[@]}" || true)
fi
STALE=""
CHECKED=0
for FILE in $WORKFLOWS; do
    NAME=$(basename "$FILE")
    LATEST=$(gh api --method GET "repos/${REPO}/actions/workflows/${NAME}/runs" \
        -f branch="$BRANCH" -f event=push -F per_page=1 \
        --jq '.workflow_runs[0] // {} | "\(.status // "missing") \(.conclusion // "")"')
    read -r STATUS CONCLUSION <<< "$LATEST"
    if [ "$STATUS" = "missing" ]; then
        echo "$NAME never ran on $BRANCH, skipped"
        continue
    fi
    CHECKED=$((CHECKED + 1))
    if [ "$STATUS" != "completed" ]; then
        STALE+="  $NAME ($STATUS)"$'\n'
    elif [ "$CONCLUSION" != "success" ] && [ "$CONCLUSION" != "skipped" ] \
        && [ "$CONCLUSION" != "neutral" ]; then
        STALE+="  $NAME ($CONCLUSION)"$'\n'
    fi
done
if [ -n "$STALE" ]; then
    printf 'newest push run on %s did not pass:\n%s' "$BRANCH" "$STALE" >&2
    exit 1
fi

echo "$CHECKED workflows pass on $BRANCH"
