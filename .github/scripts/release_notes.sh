#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints a release's notes: the given notes, then the notes GitHub generates from the
# config. The new contributors section is left out, because GitHub counts it across the
# whole repo. A PR that the previous tag shipped as a cherry-pick is left out too. Needs
# a full clone. Usage: release_notes.sh <tag> <sha> <config> [previous_tag] [notes].

set -euo pipefail

USAGE="usage: release_notes.sh <tag> <sha> <config> [previous_tag] [notes]"
TAG=${1:?$USAGE}
SHA=${2:?$USAGE}
CONFIG=${3:?$USAGE}
PREVIOUS_TAG=${4:-}
NOTES=${5:-}
GH_API="$(dirname "$0")/gh_api.sh"

if [ "$(git rev-parse --is-shallow-repository)" = true ]; then
    echo "release_notes.sh needs a full clone, not a shallow one" >&2
    exit 1
fi

ERR=$(mktemp)
trap 'rm -f "$ERR"' EXIT

# Prints the PR number at the end of the subject of the message on stdin.
subject_pr() {
    sed -nE '1s/.*\(#([0-9]+)\)$/\1/p'
}

# Prints a commit's message. A commit outside the clone, such as one of a deleted PR
# branch, is read from GitHub, and one that GitHub does not know prints nothing.
message() {
    if git cat-file -e "$1^{commit}" 2> /dev/null; then
        git log -1 --format=%B "$1"
        return
    fi
    if ! "$GH_API" "repos/$GITHUB_REPOSITORY/commits/$1" --jq .commit.message \
        2> "$ERR"; then
        grep -q "No commit found for SHA" "$ERR" || { cat "$ERR" >&2 && return 1; }
    fi
}

# Prints each commit of a PR as a line of JSON. The endpoint lists at most 250 commits,
# so a PR that reaches that count fails.
pr_commits() {
    local commits
    commits=$("$GH_API" "repos/$GITHUB_REPOSITORY/pulls/$1/commits" --paginate \
        --jq '.[] | @json') || return
    if [ "$(grep -c . <<< "$commits")" -ge 250 ]; then
        echo "PR #$1 has 250 or more commits, and GitHub lists only 250" >&2
        return 1
    fi
    echo "$commits"
}

# Prints a "pick <commit> <carrier> <source>" line for each `cherry-pick -x` trailer of
# a commit and a "revert <commit> <carrier> <target>" line for each commit it reverts.
# A squash merge loses the order and SHAs of its PR's commits, so a commit whose subject
# names a PR reads that PR's commits, and each is the carrier of its lines. A direct
# pick of a squash merge keeps its subject, so it reads its own message.
links() {
    local msg found pr source
    msg=$(message "$1") || return
    found=$(sed -nE \
        -e "s/^\(cherry picked from commit ([0-9a-f]+)\)$/pick $1 $1 \1/p" \
        -e "s/^This reverts commit ([0-9a-f]+).*/revert $1 $1 \1/p" <<< "$msg")
    pr=$(subject_pr <<< "$msg")
    while read -r source; do
        if git cat-file -e "$source^{commit}" 2> /dev/null \
            && [ "$(git log -1 --format=%s "$source" | subject_pr)" = "$pr" ]; then
            pr=""
        fi
    done < <(awk '$1 == "pick" { print $4 }' <<< "$found")
    if [ -z "$pr" ]; then
        echo "$found"
        return
    fi
    pr_commits "$pr" | jq -r --arg carrier "$1" '
        .sha as $commit | .commit.message
        | (scan("\\(cherry picked from commit ([0-9a-f]+)\\)")
            | "pick \($commit) \($carrier) \(.[0])"),
          (scan("This reverts commit ([0-9a-f]+)")
            | "revert \($commit) \($carrier) \(.[0])")'
}

# The commits the previous tag holds that SHA does not are its cherry-picks, oldest
# first. A source off SHA is itself a pick, such as one from an older train's branch, so
# its own links are read too, as are those of a source outside the clone.
LINKS=""
OPS=""
SEEN=""
if [ -n "$PREVIOUS_TAG" ]; then
    OPS=$(git rev-list --reverse --no-merges "$PREVIOUS_TAG" "^$SHA")
fi
PICKS=$OPS
while [ -n "$PICKS" ]; do
    FOUND=""
    for commit in $PICKS; do
        FOUND+=$(links "$commit")$'\n'
    done
    LINKS+=$FOUND
    PICKS=""
    while read -r source; do
        if [[ " $SEEN " == *" $source "* ]]; then continue; fi
        SEEN+=" $source"
        if ! git cat-file -e "$source^{commit}" 2> /dev/null \
            || ! git merge-base --is-ancestor "$source" "$SHA"; then
            PICKS+=" $source"
        fi
    done < <(awk '$1 == "pick" { print $4 }' <<< "$FOUND")
done
# Applies the picks in commit order, so each source keeps its last state. A pick ships
# its source, and a revert flips the state of its target. A squash merge applies its
# PR's commits in order, and a revert of it undoes them in reverse order. A revert ships
# with the picked fix it undoes or restores, so a source counts as shipped when the
# picked fix at the end of its revert chain does.
SOURCES=$(awk -v ops="$(tr '\n' ' ' <<< "$OPS")" '
    function add(list, key, value) {
        if (index(list[key] " ", " " value " ") == 0) list[key] = list[key] " " value
    }
    function apply(c, on,   n, i, list) {
        if (c in subs) {
            n = split(subs[c], list, " ")
            for (i = 1; i <= n; i++) apply(list[on ? i : n + 1 - i], on)
            return
        }
        if (c in picks) {
            n = split(picks[c], list, " ")
            for (i = 1; i <= n; i++) apply(list[i], on)
        } else {
            state[c] = on
        }
        n = split(reverts[c], list, " ")
        for (i = 1; i <= n; i++) apply(list[i], !on)
    }
    function base(c,   list) {
        while (1) {
            if (reverts[c] != "") split(reverts[c], list, " ")
            else if (c in picks) split(picks[c], list, " ")
            else return c
            if (!(list[1] in sources)) return c
            c = list[1]
        }
    }
    $1 == "pick" { add(picks, $2, $4); sources[$4] = 1 }
    $1 == "revert" { n++; commit[n] = $2; target[n] = $4 }
    ($1 == "pick" || $1 == "revert") && $2 != $3 { add(subs, $3, $2) }
    END {
        # A pick of a revert copies its message, so the revert belongs to its source.
        for (i = 1; i <= n; i++) {
            if (!(commit[i] in picks)) {
                add(reverts, commit[i], target[i])
                continue
            }
            m = split(picks[commit[i]], from, " ")
            for (j = 1; j <= m; j++) add(reverts, from[j], target[i])
        }
        m = split(ops, list, " ")
        for (i = 1; i <= m; i++) apply(list[i], 1)
        for (s in sources) if (state[base(s)]) print s
    }' <<< "$LINKS" | sort -u | tr '\n' ' ')

# A source with a PR in its subject is that PR's squash merge. GitHub resolves the
# merged PR of any other source, such as a commit of a PR branch. That PR counts as
# shipped only when all of its commits but merges were picked. A source picked from a
# commit that was never pushed has no PR.
SHIPPED=""
for source in $SOURCES; do
    PR=""
    if git cat-file -e "$source^{commit}" 2> /dev/null; then
        PR=$(git log -1 --format=%s "$source" | subject_pr)
    fi
    if [ -n "$PR" ]; then
        SHIPPED+=" $PR"
        continue
    fi
    if ! PRS=$("$GH_API" "repos/$GITHUB_REPOSITORY/commits/$source/pulls" \
        --jq '.[] | select(.merged_at) | .number' 2> "$ERR"); then
        if ! grep -q "No commit found for SHA" "$ERR"; then
            cat "$ERR" >&2
            exit 1
        fi
        continue
    fi
    for pr in $PRS; do
        COMMITS=$(pr_commits "$pr" \
            | jq -r 'select(.parents | length == 1) | .sha')
        ALL=true
        for commit in $COMMITS; do
            if [[ " $SOURCES " != *" $commit "* ]]; then ALL=false; fi
        done
        if [ "$ALL" = true ]; then SHIPPED+=" $pr"; fi
    done
done

ARGS=(-f tag_name="$TAG" -f target_commitish="$SHA")
ARGS+=(-f configuration_file_path="$CONFIG")
if [ -n "$PREVIOUS_TAG" ]; then ARGS+=(-f previous_tag_name="$PREVIOUS_TAG"); fi
# GitHub rejects a release body over 125,000 characters. A longer body is cut at a line,
# and keeps its Full Changelog link. awk may count bytes, which only cuts sooner.
cap() {
    awk -v limit=125000 '
        { lines[NR] = $0; total += length($0) + 1 }
        /^\*\*Full Changelog\*\*/ { link = $0; last = NR - 1 }
        END {
            if (total <= limit) {
                for (i = 1; i <= NR; i++) print lines[i]
                exit
            }
            if (link == "") last = NR
            note = "_The list is cut short to fit the size limit of a GitHub release._"
            budget = limit - length(link) - length(note) - 4
            for (i = 1; i <= last; i++) {
                if (used + length(lines[i]) + 1 > budget) break
                used += length(lines[i]) + 1
                if (lines[i] ~ /^\* /) keep = i
            }
            for (i = 1; i <= keep; i++) print lines[i]
            print ""
            print note
            if (link != "") { print ""; print link }
        }'
}

{
    if [ -n "$NOTES" ]; then printf '%s\n\n' "$NOTES"; fi
    gh api "repos/$GITHUB_REPOSITORY/releases/generate-notes" "${ARGS[@]}" --jq .body \
        | awk -v shipped="$SHIPPED" '
            BEGIN { split(shipped, prs, " "); for (i in prs) drop["/pull/" prs[i]] = 1 }
            /^## New Contributors/ { skip = 1 }
            /^\*\*Full Changelog\*\*/ { skip = 0 }
            skip { next }
            /^\* / && match($0, /\/pull\/[0-9]+$/) && substr($0, RSTART) in drop { next }
            # A section or category whose PRs all shipped earlier is left out.
            /^## / { section = $0; category = ""; next }
            /^### / { category = $0; next }
            /^$/ && section category != "" { next }
            /^\* / && section != "" { print section }
            /^\* / && category != "" { print category }
            { section = ""; category = ""; print }'
} | cap
