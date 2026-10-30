#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Adds copyright headers to files and formats them with the tools CI checks.
#
#   scripts/fix.sh            files changed against the merge base with origin/main,
#                             untracked files included
#   scripts/fix.sh --staged   staged files, then stages the fixes again
#   scripts/fix.sh <file...>  the files given
#
# A missing formatter is skipped with a warning. With --staged, a partially staged file
# has its staged copy fixed and staged, and its working copy fixed but not staged. A
# fixer that fails on the working copy does not fail the run.

set -euo pipefail

PREFIX=$(git rev-parse --show-prefix)
cd "$(git rev-parse --show-toplevel)"

FILES=()
add_files() {
    local f
    while IFS= read -r -d '' f; do
        [ -f "$f" ] && FILES+=("$f")
    done
}

STAGED=0
if [ "${1:-}" = "--staged" ]; then
    STAGED=1
    add_files < <(git diff --cached --name-only --diff-filter=ACMR -z)
elif [ $# -gt 0 ]; then
    for f in "$@"; do
        [[ $f == /* ]] || f="$PREFIX$f"
        [ -f "$f" ] && FILES+=("$f")
    done
else
    base=$(git merge-base HEAD origin/main)
    add_files < <(git diff --name-only --diff-filter=ACMR -z "$base")
    add_files < <(git ls-files --others --exclude-standard -z)
fi

if [ ${#FILES[@]} -eq 0 ]; then
    exit 0
fi

WARNED=0
skip() {
    [ "$WARNED" = "1" ] || echo "fix.sh: $1 not found; $2 files are not formatted." >&2
}

# Fixers write through a symlink to its target, so links are not fixed.
fix() {
    local f files=() go=() py=() cpp=()
    for f in "$@"; do
        [ -L "$f" ] && continue
        files+=("$f")
        case "$f" in
            *.go) go+=("$f") ;;
            *.py | *.pyi) py+=("$f") ;;
            *.cpp | *.hpp | *.h | *.cc) cpp+=("$f") ;;
        esac
    done
    [ ${#files[@]} -gt 0 ] || return 0
    scripts/update_copyrights.sh "${files[@]}" > /dev/null
    if [ -x node_modules/.bin/prettier ]; then
        node_modules/.bin/prettier --write --ignore-unknown --log-level warn -- \
            "${files[@]}"
    else
        skip "Prettier (run pnpm install)" "TypeScript, Markdown, and config"
    fi
    if [ ${#go[@]} -gt 0 ]; then
        if command -v golangci-lint > /dev/null; then
            golangci-lint fmt "${go[@]}"
        else
            skip "golangci-lint" "Go"
        fi
    fi
    if [ ${#py[@]} -gt 0 ]; then
        if command -v uv > /dev/null; then
            local ruff=(uv run --quiet --frozen --package synnax-scripts ruff)
            "${ruff[@]}" check --quiet --select I --fix --exit-zero -- "${py[@]}"
            "${ruff[@]}" format --quiet -- "${py[@]}"
        else
            skip "uv" "Python"
        fi
    fi
    if [ ${#cpp[@]} -gt 0 ]; then
        if command -v clang-format > /dev/null; then
            scripts/clang_format.sh --files "${cpp[@]}" > /dev/null
        else
            skip "clang-format (run scripts/install_clang_format.sh)" "C++"
        fi
    fi
    WARNED=1
}

# The working copies of partially staged files wait in SAVED while their staged copies
# are fixed. The trap puts them back if a fixer fails.
PARTIAL=()
SAVED=$(mktemp -d "$(git rev-parse --git-dir)/fix.XXXXXX")
restore() {
    local i
    for ((i = 0; i < ${#PARTIAL[@]}; i++)); do
        rm -f -- "${PARTIAL[i]}"
        mv -- "$SAVED/$i" "${PARTIAL[i]}"
    done
    rm -r "$SAVED"
}
trap restore EXIT

if [ "$STAGED" = "1" ]; then
    for f in "${FILES[@]}"; do
        git diff --quiet -- "$f" && continue
        mv -- "$f" "$SAVED/${#PARTIAL[@]}"
        PARTIAL+=("$f")
        git checkout -- "$f"
    done
fi

BEFORE=$(git hash-object -- "${FILES[@]}")
fix "${FILES[@]}"
AFTER=$(git hash-object -- "${FILES[@]}")

i=0
while IFS=$'\t' read -r before after; do
    f="${FILES[i]}"
    i=$((i + 1))
    [ "$before" = "$after" ] && continue
    echo "fix.sh: fixed $f"
    if [ "$STAGED" = "1" ]; then
        git add -- "$f"
    fi
done < <(paste <(echo "$BEFORE") <(echo "$AFTER"))

if [ ${#PARTIAL[@]} -gt 0 ]; then
    trap - EXIT
    restore
    fix "${PARTIAL[@]}" || true
fi
