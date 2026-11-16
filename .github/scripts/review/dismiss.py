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

from review.post import (
    GH,
    Target,
    Thread,
    command,
    issue_comment,
    member,
    named,
    outside_findings,
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
    `@sedosaur dismiss` dismisses the finding, and the bot resolves its thread or
    answers the comment. When every finding of the newest review of the head is then
    dismissed, a new check run scores the head 5/5. Any other reply gets the usage.

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
    c = issue_comment(t, gh, comment)
    if c is None:
        raise ValueError(f"comment {comment} is not on pull request {t.pr}")
    text = command(c["body"])
    if text is not None and not member(c):
        _comment(t, gh, REFUSED)
    elif text is None or named(text, outside_findings(t, gh)) is None:
        _comment(t, gh, OUTSIDE_USAGE)
    else:
        _comment(t, gh, DONE + (PASSED if rescore(t, gh, found, key) else ""))


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
