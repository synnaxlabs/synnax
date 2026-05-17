#!/bin/sh

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Audit every `uses:` reference across .github/workflows and .github/actions.
# Prints, for each distinct action, the versions/refs it is pinned to and the
# number of occurrences. Local `./...` action references are skipped. Actions
# pinned to more than one version are listed in a detail block with file:line
# locations.

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
GITHUB_DIR=$(dirname "$SCRIPT_DIR")
REPO_DIR=$(dirname "$GITHUB_DIR")

TMP=$(mktemp -t audit_actions.XXXXXX)
trap 'rm -f "$TMP"' EXIT

# Extract `uses:` references as TAB-separated: name<TAB>version<TAB>file:line
find "$GITHUB_DIR/workflows" "$GITHUB_DIR/actions" \
    \( -name '*.yaml' -o -name '*.yml' \) -type f 2> /dev/null \
    | sort \
    | while IFS= read -r file; do
        rel=${file#"$REPO_DIR"/}
        grep -nE '^[[:space:]]*-?[[:space:]]*uses:[[:space:]]*' "$file" \
            | awk -v rel="$rel" '
            {
                line = $0
                sub(/^[0-9]+:/, "", line)
                sub(/^[[:space:]]*-?[[:space:]]*uses:[[:space:]]*/, "", line)
                gsub(/["\x27]/, "", line)
                sub(/[[:space:]]+#.*$/, "", line)
                sub(/[[:space:]]+$/, "", line)
                if (line == "" || line ~ /^#/) next

                match($0, /^[0-9]+/)
                lineno = substr($0, RSTART, RLENGTH)

                if (line ~ /^\.\.?\//) {
                    next
                } else if (index(line, "@") > 0) {
                    at = index(line, "@")
                    name = substr(line, 1, at - 1)
                    version = substr(line, at + 1)
                } else {
                    name = line; version = "(unpinned)"
                }
                printf "%s\t%s\t%s:%s\n", name, version, rel, lineno
            }'
    done > "$TMP"

total=$(wc -l < "$TMP" | tr -d ' ')
files=$(cut -f3 "$TMP" | cut -d: -f1 | sort -u | wc -l | tr -d ' ')

if [ "$total" -eq 0 ]; then
    echo "No \`uses:\` references found."
    exit 0
fi

name_w=$(cut -f1 "$TMP" | awk '{ if (length > m) m = length } END { print m }')

printf "Found %s \`uses:\` references across %s files.\n\n" "$total" "$files"
printf "%-*s  VERSIONS\n" "$name_w" "ACTION"
printf "%s  %s\n" \
    "$(printf '%*s' "$name_w" '' | tr ' ' '-')" \
    "$(printf '%*s' 40 '' | tr ' ' '-')"

# Per-name summary: collapse versions into "v1 (3), v2 (1)" form.
cut -f1,2 "$TMP" | sort | uniq -c \
    | awk '{ count = $1; name = $2; version = $3;
           printf "%s\t%s (%d)\n", name, version, count }' \
    | sort \
    | awk -F'\t' -v w="$name_w" '
    {
        if ($1 != prev && prev != "") {
            marker = (n > 1) ? "*" : " "
            printf "%-*s %s%s\n", w, prev, marker, joined
            joined = ""
            n = 0
        }
        prev = $1
        joined = (joined == "") ? $2 : joined ", " $2
        n++
    }
    END {
        if (prev != "") {
            marker = (n > 1) ? "*" : " "
            printf "%-*s %s%s\n", w, prev, marker, joined
        }
    }'

# Detail block for any action pinned to more than one version.
multi=$(cut -f1,2 "$TMP" | sort -u | cut -f1 | uniq -c \
    | awk '$1 > 1 { print $2 }')

if [ -n "$multi" ]; then
    printf "\nActions with multiple pinned versions (marked with *):\n"
    echo "$multi" | while IFS= read -r name; do
        [ -z "$name" ] && continue
        printf "\n  %s\n" "$name"
        awk -F'\t' -v target="$name" '$1 == target { print $2 "\t" $3 }' "$TMP" \
            | sort \
            | awk -F'\t' '
            {
                if ($1 != prev) {
                    printf "    %s\n", $1
                    prev = $1
                }
                printf "      %s\n", $2
            }'
    done
fi
