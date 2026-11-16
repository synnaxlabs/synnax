#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Lets a member tell the bot to stop reporting a finding on a pull request, with a
reply of `@sedosaur dismiss` on the thread of the finding, or for a finding outside the
diff, a comment of `@sedosaur dismiss` and its title."""

import json
import re
from typing import Any

from review.findings import Dismissal
from review.post import (
    GH,
    Target,
    Thread,
    command,
    dismissals,
    member,
    outside_findings,
    record,
    rescore,
    threads,
    write,
)

USAGE = (
    "To dismiss this finding, reply `@sedosaur dismiss` and a reason. To ask for a "
    "review, comment `@sedosaur review` on the pull request."
)
OUTSIDE_USAGE = (
    "No one finding outside the diff has that title. To dismiss one, comment "
    "`@sedosaur dismiss`, its title, and a reason. When two findings share the title, "
    "add `path:<file>` after it."
)
_QUOTES = "\"'\u201c\u201d\u2018\u2019"
_PATH = re.compile(r"(?:^|\s)path:(\S+)")
REFUSED = "Only a member can dismiss a finding, not an agent."
DONE = "Dismissed. Later reviews of this pull request will not report it."
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


def dismiss(t: Target, gh: GH, *, comment: int, key: str) -> None:
    """Answers a reply that mentions the bot on the thread of one of its findings, or a
    comment on the pull request that dismisses a finding outside the diff. A member's
    `@sedosaur dismiss` resolves the thread, or records the dismissal in a comment of
    the bot. When every finding of the newest review of the head is then dismissed, a
    new check run scores the head 5/5. Any other reply gets the usage.

    :param comment: The database ID of the reply or the comment.
    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    :raises ValueError: If the comment is neither a reply to a finding of the bot nor a
        comment on the pull request.
    """
    found = threads(t, gh)
    hits = [(th, c) for th in found for c in th.replies if c["databaseId"] == comment]
    if not hits:
        _dismiss_outside(t, gh, found, comment=comment, key=key)
        return
    [(thread, reply)] = hits
    if command(reply["body"]) is None:
        _reply(gh, thread, USAGE)
    elif not member(reply):
        _reply(gh, thread, REFUSED)
    else:
        passed = rescore(t, gh, dismissals(t, gh, found), key)
        _reply(gh, thread, DONE + (PASSED if passed else ""))
        gh("api", "graphql", "-f", f"query={_RESOLVE}", "-f", f"t={thread.id}")


def _dismiss_outside(
    t: Target, gh: GH, found: list[Thread], *, comment: int, key: str
) -> None:
    c = json.loads(gh("api", f"repos/{t.repo}/issues/comments/{comment}"))
    if not c["issue_url"].endswith(f"/issues/{t.pr}"):
        raise ValueError(f"comment {comment} is not on pull request {t.pr}")
    # The shape that `member` reads from GraphQL.
    reply: dict[str, Any] = {
        "author": {"__typename": c["user"]["type"], "login": c["user"]["login"]},
        "authorAssociation": c["author_association"],
        "body": c["body"],
    }
    text = command(reply["body"])
    if text is not None and not member(reply):
        _comment(t, gh, REFUSED)
        return
    named = None if text is None else _named(text, outside_findings(t, gh))
    if named is None:
        _comment(t, gh, OUTSIDE_USAGE)
        return
    (path, title), reason = named
    d = Dismissal(path=path, title=title, reason=reason)
    passed = rescore(t, gh, [*dismissals(t, gh, found), d], key)
    _comment(t, gh, f"{record(d)}\n\n{DONE}" + (PASSED if passed else ""))


def _named(
    text: str, findings: set[tuple[str, str]]
) -> tuple[tuple[str, str], str] | None:
    """Returns the one finding whose title starts the text, and the reason after it, or
    None. Case, backticks, and quotes around the title do not count. `path:<file>` picks
    one of the findings that share a title."""
    text = text.replace("`", "").strip().lstrip(_QUOTES)
    plain = {f: f[1].replace("`", "").strip().strip(_QUOTES).strip() for f in findings}
    hits = [f for f, p in plain.items() if text.casefold().startswith(p.casefold())]
    size = max((len(plain[f]) for f in hits), default=0)
    hits = [f for f in hits if len(plain[f]) == size]
    rest = text[size:].lstrip(_QUOTES)
    if m := _PATH.search(rest):
        hits = [f for f in hits if f[0] == m[1]]
        rest = rest[: m.start()] + rest[m.end() :]
    if len(hits) != 1:
        return None
    return hits[0], rest.strip().lstrip(",:;.").strip()


def _comment(t: Target, gh: GH, body: str) -> None:
    write(gh, "POST", f"repos/{t.repo}/issues/{t.pr}/comments", {"body": body})


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
