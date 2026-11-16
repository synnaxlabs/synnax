#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Lets a member tell the bot to stop reporting a finding on a pull request, with a
reply of `@sedosaur dismiss` on the thread of the finding, or with a comment of
`@sedosaur dismiss` and its title on the pull request."""

import re

from review.post import (
    GH,
    Target,
    Thread,
    answer,
    answers,
    command,
    comments,
    issue_comment,
    member,
    outside_findings,
    rescore,
    threads,
    write,
)

USAGE = (
    "To dismiss this finding, reply `@sedosaur dismiss` and a reason. To ask for a "
    "review, comment `@sedosaur review` on the pull request."
)
UNNAMED = (
    "No one open finding has that title. To dismiss a finding, comment "
    "`@sedosaur dismiss`, its title, and a reason. When two findings share the title, "
    "add `path:<file>` after it."
)
REFUSED = "Only a member can dismiss a finding, not an agent."
DONE = "Dismissed. Later reviews of this pull request will not report it."
PASSED = (
    " Every finding of the review of the head is dismissed, so the head scores 5/5."
)
_QUOTES = "\"'\u201c\u201d\u2018\u2019"
_PATH = re.compile(r"(?:^|\s)path:(\S+)")
_REPLY = """mutation($t: ID!, $b: String!) {
  addPullRequestReviewThreadReply(input: {pullRequestReviewThreadId: $t, body: $b}) {
    comment { id }
  }
}"""
_RESOLVE = """mutation($t: ID!) {
  resolveReviewThread(input: {threadId: $t}) { thread { id } }
}"""


def dismiss(t: Target, gh: GH, *, comment: int, key: str) -> None:
    """Answers a reply that mentions the bot on the thread of one of its findings, or
    each dismissal comment on the pull request that the bot has not answered yet. A
    member's `@sedosaur dismiss` dismisses the finding and resolves its thread. A
    comment binds to the one open finding its title names, and the answer records that
    finding. When every finding of the newest review of the head is then dismissed, a
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
        _reply(gh, thread, DONE + (PASSED if rescore(t, gh, found, key) else ""))
        gh("api", "graphql", "-f", f"query={_RESOLVE}", "-f", f"t={thread.id}")


def _dismiss_outside(
    t: Target, gh: GH, found: list[Thread], *, comment: int, key: str
) -> None:
    if issue_comment(t, gh, comment) is None:
        raise ValueError(f"comment {comment} is not on pull request {t.pr}")
    posted = comments(t, gh)
    done = {a["comment"] for a in answers(t, posted)}
    still = {(th.path, th.title): th for th in found if not th.resolved}
    targets = outside_findings(t, gh) | still.keys()
    bound = False
    for c in posted:
        text = command(c["body"])
        if c["login"] == f"{t.bot}[bot]" or c["id"] in done or text is None:
            continue
        hit = _named(text, targets)
        if not member(c):
            _comment(t, gh, answer({"comment": c["id"]}, REFUSED))
        elif hit is None:
            _comment(t, gh, answer({"comment": c["id"]}, UNNAMED))
        else:
            (path, title), reason = hit
            record = {"comment": c["id"], "path": path, "title": title}
            _comment(t, gh, answer(record | {"reason": reason}, DONE))
            if (th := still.get((path, title))) is not None:
                gh("api", "graphql", "-f", f"query={_RESOLVE}", "-f", f"t={th.id}")
            bound = True
    if bound and rescore(t, gh, found, key):
        _comment(t, gh, PASSED.strip())


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
