#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Lets someone with write access to the repository tell the bot to stop reporting a
finding on a pull request, with a reply of `@sedosaur dismiss` on the thread of the
finding, or with a comment of `@sedosaur dismiss` and its title on the pull request."""

import re

from review.post import (
    GH,
    Authority,
    Target,
    Thread,
    answer,
    answers,
    authority,
    by_bot,
    command,
    comments,
    issue_comment,
    outside_findings,
    pending,
    rebutted,
    reply,
    rescore,
    resolve,
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
REFUSED = "Only someone with write access to the repository can dismiss a finding."
DONE = "Dismissed. Later reviews of this pull request will not report it."
PASSED = (
    " Every finding of the review of the head is dismissed or withdrawn, so the head "
    "scores 5/5."
)
_QUOTES = "\"'\u201c\u201d\u2018\u2019"
_PATH = re.compile(r"(?:^|\s)path:(\S+)")


def dismiss(t: Target, gh: GH, *, comment: int, key: str) -> bool:
    """Answers a reply on the thread of one of the bot's findings, or a comment on the
    pull request, and each reply and comment on the pull request that has no answer
    yet, so a run also does the work of a run that never finished. A writer's
    `@sedosaur dismiss` reply dismisses the finding of its thread. A rebuttal waits for
    `judge`. An open thread with other replies gets the usage once. A writer's comment
    binds to the one open finding its title names, and the answer records that finding.
    Each dismissal resolves the finding's thread. When every finding of the newest
    review of the head is then dismissed, a new check run scores the head 5/5. Returns
    whether a rebuttal waits for `judge`.

    :param comment: The database ID of the reply or the comment.
    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    :raises ValueError: If the comment is neither a reply to a finding of the bot nor a
        comment on the pull request.
    """
    found = threads(t, gh)
    hit = any(c["databaseId"] == comment for th in found for c in th.replies)
    if not hit and issue_comment(t, gh, comment) is None:
        raise ValueError(f"comment {comment} is not on pull request {t.pr}")
    can = authority(t, gh)
    _sweep_threads(t, gh, found, can)
    _sweep_comments(t, gh, found, can)
    if rescore(t, gh, found, key, can):
        _comment(t, gh, PASSED.strip())
    return any(rebutted(t, th, can) for th in threads(t, gh))


def _sweep_threads(t: Target, gh: GH, found: list[Thread], can: Authority) -> None:
    """Answers each dismissal reply that no reply of the bot follows, and gives the
    usage to each open thread that has other replies only and no rebuttal."""
    for th in found:
        if not th.resolved and _needs_usage(t, th, can):
            reply(gh, th, USAGE)
        for i, r in enumerate(th.replies):
            later = th.replies[i + 1 :]
            if (
                by_bot(t, r)
                or command(r["body"]) is None
                or any(by_bot(t, c) for c in later)
            ):
                continue
            if not can(r):
                reply(gh, th, REFUSED)
                continue
            reply(gh, th, DONE)
            resolve(gh, th)


def _needs_usage(t: Target, th: Thread, can: Authority) -> bool:
    replies = pending(t, th)
    return (
        bool(replies)
        and all(command(r["body"]) is None for r in replies)
        and not rebutted(t, th, can)
        and not any(by_bot(t, r) and r["body"] == USAGE for r in th.replies)
    )


def _sweep_comments(t: Target, gh: GH, found: list[Thread], can: Authority) -> None:
    """Answers each dismissal comment on the pull request that has no answer yet."""
    posted = comments(t, gh)
    done = {a["comment"] for a in answers(t, posted)}
    still = {(th.path, th.title): th for th in found if not th.resolved}
    targets = outside_findings(t, gh) | still.keys()
    for c in posted:
        text = command(c["body"])
        if c["login"] == f"{t.bot}[bot]" or c["id"] in done or text is None:
            continue
        hit = _named(text, targets)
        if not can(c):
            _comment(t, gh, answer({"comment": c["id"]}, REFUSED))
        elif hit is None:
            _comment(t, gh, answer({"comment": c["id"]}, UNNAMED))
        else:
            (path, title), reason = hit
            record = {"comment": c["id"], "path": path, "title": title}
            _comment(t, gh, answer(record | {"reason": reason}, DONE))
            if (th := still.get((path, title))) is not None:
                resolve(gh, th)


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
