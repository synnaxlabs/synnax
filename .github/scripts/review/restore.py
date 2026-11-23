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
from enum import Enum

from review import render
from review.post import GH, Target, newest, summaries, write

CHANGED = "The head or the body changed during the restore. Run it again."


class Unrestorable(Exception):
    """The summary cannot go back in the body."""


class Outcome(Enum):
    """What a restore did."""

    RESTORED = "Restored the summary."
    PRESENT = "The body holds the summary."
    PENDING = "A review of the head is on the way. That review writes the summary."


def restore(t: Target, gh: GH, *, key: str, writer: GH) -> Outcome:
    """Writes the summary that the bot wrote last back in the pull request body, and
    keeps the rest of the body. Writes nothing when the body holds it already, or while
    a review of the head is on the way: that review writes the summary.

    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    :param writer: Runs `gh` for the write of the body.
    :raises Unrestorable: No review of the head is on the way or it failed, the bot
        never wrote a summary, or the head or the body changed during the restore.
    """
    before = _body(t, gh)
    run = newest(t, gh, t.head, key)
    if run is None:
        if _requested(t, gh):
            return Outcome.PENDING
        raise Unrestorable(
            f"Sedosaur has no review of {t.head} on the way. Dispatch Review - "
            "Sedosaur to get one."
        )
    if run["status"] != "completed":
        return Outcome.PENDING
    if run["conclusion"] != "success":
        raise Unrestorable(
            f"Sedosaur's review of {t.head} failed. Dispatch Review - Sedosaur to get "
            "another."
        )
    block = next(iter(summaries(t, gh)), None)
    if block is None:
        raise Unrestorable(f"Sedosaur never wrote a summary in the body of #{t.pr}.")
    restored = render.splice(before, block)
    if restored == before:
        return Outcome.PRESENT
    # A review that posts during the reads writes a newer summary.
    if _body(t, gh) != before:
        raise Unrestorable(CHANGED)
    write(writer, "PATCH", f"repos/{t.repo}/pulls/{t.pr}", {"body": restored})
    return Outcome.RESTORED


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


def _requested(t: Target, gh: GH) -> bool:
    """Reports whether a relay run for the head or a review run for the pull request is
    queued or running. The relay waits two minutes before it dispatches the review, and
    the review creates its check run only once it starts."""
    relays = gh(
        "api",
        f"repos/{t.repo}/actions/workflows/relay.sedosaur.yaml/runs"
        f"?event=pull_request&head_sha={t.head}&per_page=100",
        "--jq",
        '.workflow_runs[] | select(.status != "completed") | .id',
    )
    reviews = gh(
        "api",
        f"repos/{t.repo}/actions/workflows/review.sedosaur.yaml/runs"
        "?event=workflow_dispatch&per_page=100",
        "--jq",
        '.workflow_runs[] | select(.status != "completed") | .display_title',
    )
    # A replay or a restore writes no summary, so only a plain or repeat run counts.
    titles = [line.split() for line in reviews.splitlines() if line]
    review = any(w[0] == str(t.pr) and w[1:] in ([], ["repeat"]) for w in titles)
    return bool(relays.strip()) or review
