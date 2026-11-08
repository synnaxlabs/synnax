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
from typing import Any

from review.post import GH, Target, reviewed, write

LISTED = 300
"""The most files that a comparison of two commits lists."""

_HUNK = re.compile(r"(?m)^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@")


def carry(t: Target, gh: GH, *, key: str) -> bool:
    """Completes the check run of the target with the review of the newest commit that
    the bot reviewed, when that review succeeded for the key and the diff from the base
    to that commit is the diff to the head. The line numbers of the two diffs can
    differ. Returns whether the head got the review.

    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    """
    old = next(iter(reviewed(t, gh)), None)
    if old is None or old == t.head:
        return False
    run = _newest(t, gh, old, key)
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
    return True


def _newest(t: Target, gh: GH, commit: str, key: str) -> dict[str, Any] | None:
    out = gh(
        "api",
        f"repos/{t.repo}/commits/{commit}/check-runs"
        "?check_name=Sedosaur%20Review&filter=all&per_page=100",
        "--jq",
        ".check_runs[] | tojson",
    )
    runs = [json.loads(line) for line in out.splitlines() if line]
    ours = [r for r in runs if r["app"]["slug"] == t.bot and r["external_id"] == key]
    return max(ours, key=lambda r: r["started_at"], default=None)


def _diff(t: Target, gh: GH, commit: str) -> list[tuple[str, ...]] | None:
    """Returns each file of the diff from the base to a commit, without line numbers,
    or None when the comparison cannot list every file. A binary file has no patch, so
    its blob stands in."""
    compare = f"repos/{t.repo}/compare/{t.base}...{commit}"
    out = gh("api", compare, "--jq", ".files[] | tojson")
    files = [json.loads(line) for line in out.splitlines() if line]
    if len(files) >= LISTED:
        return None
    return [
        (
            f["filename"],
            f.get("previous_filename", ""),
            f["status"],
            f["sha"] if f.get("patch") is None else _HUNK.sub("@@", f["patch"]),
        )
        for f in files
    ]
