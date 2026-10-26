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
# A missing formatter is skipped with a warning. With --staged, the script fails when it
# changes a file that also has unstaged changes, because it cannot stage only the fix.

set -euo pipefail

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

GO=()
PY=()
CPP=()
for f in "${FILES[@]}"; do
    case "$f" in
        *.go) GO+=("$f") ;;
        *.py | *.pyi) PY+=("$f") ;;
        *.cpp | *.hpp | *.h | *.cc) CPP+=("$f") ;;
    esac
done

BEFORE=$(git hash-object -- "${FILES[@]}")
UNSTAGED=$(git diff --name-only)

skip() {
    echo "fix.sh: $1 not found, so $2 files are not formatted." >&2
}

scripts/update_copyrights.sh "${FILES[@]}" > /dev/null

if [ -x node_modules/.bin/prettier ]; then
    node_modules/.bin/prettier --write --ignore-unknown --log-level warn \
        -- "${FILES[@]}"
else
    skip "Prettier (run pnpm install)" "TypeScript, Markdown, and config"
fi

if [ ${#GO[@]} -gt 0 ]; then
    if command -v golangci-lint > /dev/null; then
        golangci-lint fmt "${GO[@]}"
    else
        skip "golangci-lint" "Go"
    fi
fi

if [ ${#PY[@]} -gt 0 ]; then
    if command -v uv > /dev/null; then
        ruff=(uv run --quiet --frozen --package synnax-scripts ruff)
        "${ruff[@]}" check --quiet --select I --fix --exit-zero -- "${PY[@]}"
        "${ruff[@]}" format --quiet -- "${PY[@]}"
    else
        skip "uv" "Python"
    fi
fi

if [ ${#CPP[@]} -gt 0 ]; then
    if command -v clang-format > /dev/null; then
        scripts/clang_format.sh --files "${CPP[@]}" > /dev/null
    else
        skip "clang-format (run scripts/install_clang_format.sh)" "C++"
    fi
fi

AFTER=$(git hash-object -- "${FILES[@]}")

PARTIAL=()
i=0
while IFS=$'\t' read -r before after; do
    f="${FILES[i]}"
    i=$((i + 1))
    [ "$before" = "$after" ] && continue
    echo "fix.sh: fixed $f"
    [ "$STAGED" = "1" ] || continue
    if grep -qxF -- "$f" <<< "$UNSTAGED"; then
        PARTIAL+=("$f")
    else
        git add -- "$f"
    fi
done < <(paste <(echo "$BEFORE") <(echo "$AFTER"))

if [ ${#PARTIAL[@]} -gt 0 ]; then
    echo "fix.sh: these files have unstaged changes. Stage the fixes, then commit:" >&2
    printf '  %s\n' "${PARTIAL[@]}" >&2
    exit 1
fi
