#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Lets a member tell the bot to stop reporting a finding on a pull request, with a
reply of `@sedosaur dismiss` on the thread of the finding."""

import re
from typing import Any

from review import render
from review.findings import Dismissal
from review.post import GH, Target, Thread, newest, threads, write

COMMAND = re.compile(r"@sedosaur\s+dismiss\b", re.I)
MEMBERS = {"OWNER", "MEMBER", "COLLABORATOR"}
USAGE = (
    "To dismiss this finding, reply `@sedosaur dismiss` and a reason. To ask for a "
    "review, comment `@sedosaur review` on the pull request."
)
REFUSED = "Only a member can dismiss a finding, not an agent."
DISMISSED = "Dismissed. Later reviews of this pull request will not report it."
PASSED = (
    " Every finding of the review of the head is dismissed, so the head scores 5/5."
)
_REPLY = """mutation($t: ID!, $b: String!) {
  addPullRequestReviewThreadReply(input: {pullRequestReviewThreadId: $t, body: $b}) {
    comment { id }
  }
}"""
_RESOLVE = """mutation($t: ID!) {
  resolveReviewThread(input: {threadId: $t}) { thread { id } }
}"""


def dismissed(found: list[Thread]) -> list[Dismissal]:
    """Returns each finding that a member dismissed in one of the threads."""
    out: list[Dismissal] = []
    for th in found:
        reason = next((r for c in th.replies if (r := _reason(c)) is not None), None)
        if reason is not None:
            out.append(Dismissal(path=th.path, title=th.title, reason=reason))
    return out


def dismiss(t: Target, gh: GH, *, comment: int, key: str) -> None:
    """Answers a reply that mentions the bot on the thread of one of its findings. A
    member's `@sedosaur dismiss` resolves the thread, and when every finding of the
    newest review of the head is then dismissed, a new check run scores the head 5/5.
    Any other reply gets the usage.

    :param comment: The database ID of the reply.
    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    :raises ValueError: If the comment is not a reply to a finding of the bot.
    """
    found = threads(t, gh)
    hits = [(th, c) for th in found for c in th.replies if c["databaseId"] == comment]
    if not hits:
        raise ValueError(f"comment {comment} is not a reply to a finding of {t.bot}")
    [(thread, reply)] = hits
    if not COMMAND.search(reply["body"]):
        _reply(gh, thread, USAGE)
    elif _reason(reply) is None:
        _reply(gh, thread, REFUSED)
    else:
        _reply(gh, thread, DISMISSED + (PASSED if _pass(t, gh, found, key) else ""))
        gh("api", "graphql", "-f", f"query={_RESOLVE}", "-f", f"t={thread.id}")


def _reply(gh: GH, thread: Thread, body: str) -> None:
    gh(
        "api",
        "graphql",
        "-f",
        f"query={_REPLY}",
        "-f",
        f"t={thread.id}",
        "-f",
        f"b={body}",
    )


def _reason(comment: dict[str, Any]) -> str | None:
    """Returns what a member wrote beside the command, or None when the comment is not
    a dismissal by a member."""
    author = comment["author"]
    if (
        author is None
        or author["__typename"] != "User"
        or comment["authorAssociation"] not in MEMBERS
        or not COMMAND.search(comment["body"])
    ):
        return None
    return COMMAND.sub("", comment["body"]).strip()


def _pass(t: Target, gh: GH, found: list[Thread], key: str) -> bool:
    """Scores the head 5/5 in a new check run when a member dismissed every finding of
    its newest review. Returns whether it did."""
    run = newest(t, gh, t.head, key)
    if run is None or run["conclusion"] != "success":
        return False
    if run["output"]["title"] == render.scored(5):
        return False
    listed = render.listed(run["output"]["text"] or "")
    if not listed or not listed <= {(d.path, d.title) for d in dismissed(found)}:
        return False
    output = {
        "title": render.scored(5),
        "summary": "A member dismissed every finding of the review.\n\n"
        + run["output"]["summary"],
        "text": run["output"]["text"],
    }
    payload = {
        "name": run["name"],
        "head_sha": t.head,
        "external_id": key,
        "details_url": t.run_url,
        "status": "completed",
        "conclusion": "success",
        "output": output,
    }
    write(gh, "POST", f"repos/{t.repo}/check-runs", payload)
    return True
