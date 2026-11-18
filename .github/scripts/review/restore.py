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

from review import render
from review.post import GH, Target, newest, summaries


class Unreviewed(Exception):
    """The head has no finished review whose summary can go back in the body."""


def restore(t: Target, gh: GH, *, key: str) -> str | None:
    """Returns the pull request body with the summary that the bot wrote last, or None
    when the body holds it already. The rest of the body stays the same.

    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    :raises Unreviewed: The bot has no successful review of the head, or never wrote a
        summary.
    """
    run = newest(t, gh, t.head, key)
    if run is None or run["conclusion"] != "success":
        raise Unreviewed(
            f"Sedosaur has no finished review of {t.head}. Comment `@sedosaur review` "
            "to get one."
        )
    block = next(iter(summaries(t, gh)), None)
    if block is None:
        raise Unreviewed(f"Sedosaur never wrote a summary in the body of #{t.pr}.")
    body = gh("api", f"repos/{t.repo}/pulls/{t.pr}", "--jq", '.body // ""')
    body = body.removesuffix("\n")
    restored = render.splice(body, block)
    return None if restored == body else restored
