#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Gives a new head the newest review of its pull request when the diff did not
change, as after a merge of the base, so the head needs no review of its own."""

import json
import re

from review.post import GH, Target, newest, recheck, reviewed, write

LISTED = 300
"""The most files that a comparison of two commits lists."""

_HUNK = re.compile(r"(?m)^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@")


def carry(t: Target, gh: GH, *, key: str) -> bool:
    """Completes the check run of the target with the review of the newest commit that
    the bot reviewed, when that review succeeded for the key and the diff from the base
    to that commit is the diff to the head. The line numbers of the two diffs can
    differ. The head scores 5/5 when a member dismissed every blocking finding of the
    review. Returns whether the head got the review.

    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    """
    old = next(iter(reviewed(t, gh)), None)
    if old is None or old == t.head:
        return False
    run = newest(t, gh, old, key)
    if run is None or run["conclusion"] != "success":
        return False
    before = _diff(t, gh, old)
    if before is None or before != _diff(t, gh, t.head):
        return False
    link = f"[{old[:7]}](https://github.com/{t.repo}/commit/{old})"
    output = {
        "title": run["output"]["title"],
        "summary": f"The diff is the same as at {link}, so this is its review.\n\n"
        + run["output"]["summary"],
    }
    if run["output"]["text"]:
        output["text"] = run["output"]["text"]
    done = {"status": "completed", "conclusion": "success", "output": output}
    write(gh, "PATCH", f"repos/{t.repo}/check-runs/{t.check_run}", done)
    recheck(t, gh)
    return True


def changed(t: Target, gh: GH, old: str) -> tuple[set[str], set[str]] | None:
    """Compares the diff from the base to an older commit with the diff to the head.
    Line numbers do not count. Returns each file of the head's diff whose changes
    differ, and each file that only the older diff changes, or None when GitHub cannot
    compare every file."""
    before, after = _diff(t, gh, old), _diff(t, gh, t.head)
    if before is None or after is None:
        return None
    kept = {f[0] for f in after}
    return {f[0] for f in set(after) - set(before)}, {f[0] for f in before} - kept


def _diff(t: Target, gh: GH, commit: str) -> list[tuple[str, ...]] | None:
    """Returns each file of the diff from the base to a commit, without line numbers,
    or None when the comparison cannot show every file. GitHub leaves out the patch of a
    binary file and of a text diff too large to show."""
    compare = f"repos/{t.repo}/compare/{t.base}...{commit}"
    out = gh("api", compare, "--jq", ".files[] | tojson")
    files = [json.loads(line) for line in out.splitlines() if line]
    if len(files) >= LISTED or any(f.get("patch") is None for f in files):
        return None
    return [
        (
            f["filename"],
            f.get("previous_filename", ""),
            f["status"],
            _HUNK.sub("@@", f["patch"]),
        )
        for f in files
    ]
