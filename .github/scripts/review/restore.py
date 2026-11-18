#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Puts the bot's last summary back in a pull request body after an edit with an old
copy of the body dropped or changed it."""

import json

from review import render
from review.post import GH, Target, newest, summaries, write

CHANGED = "The head or the body changed during the restore. Run it again."


class Unrestorable(Exception):
    """The summary cannot go back in the body."""


def restore(t: Target, gh: GH, *, key: str, writer: GH) -> bool:
    """Writes the summary that the bot wrote last back in the pull request body, and
    keeps the rest of the body. Returns False when the body holds it already.

    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    :param writer: Runs `gh` for the write of the body.
    :raises Unrestorable: The bot has no successful review of the head or never wrote a
        summary, or the head or the body changed during the restore.
    """
    before = _body(t, gh)
    run = newest(t, gh, t.head, key)
    if run is None or run["conclusion"] != "success":
        raise Unrestorable(
            f"Sedosaur has no finished review of {t.head}. Comment `@sedosaur review` "
            "to get one."
        )
    block = next(iter(summaries(t, gh)), None)
    if block is None:
        raise Unrestorable(f"Sedosaur never wrote a summary in the body of #{t.pr}.")
    restored = render.splice(before, block)
    if restored == before:
        return False
    # A review that posts during the reads writes a newer summary.
    if _body(t, gh) != before:
        raise Unrestorable(CHANGED)
    write(writer, "PATCH", f"repos/{t.repo}/pulls/{t.pr}", {"body": restored})
    return True


def _body(t: Target, gh: GH) -> str:
    out = gh(
        "api",
        f"repos/{t.repo}/pulls/{t.pr}",
        "--jq",
        '{head: .head.sha, body: (.body // "")} | tojson',
    )
    pr = json.loads(out)
    if pr["head"] != t.head:
        raise Unrestorable(CHANGED)
    body: str = pr["body"]
    return body
