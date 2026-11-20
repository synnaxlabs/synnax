#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import re
import subprocess
import sys
from collections.abc import Callable
from dataclasses import dataclass, replace
from typing import Any

from review import render
from review.findings import Dismissal, Finding, Review, parse

GH = Callable[..., str]
"""Runs `gh` with the given arguments and optional stdin, and returns its stdout."""
Authority = Callable[[dict[str, Any]], bool]
"""Reports whether the author of a comment can dismiss a finding."""

THREADS = """query($o: String!, $r: String!, $p: Int!, $endCursor: String) {
  repository(owner: $o, name: $r) {
    pullRequest(number: $p) {
      reviewThreads(first: 100, after: $endCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          isResolved
          path
          comments(first: 100) {
            nodes { databaseId author { __typename login } body }
          }
        }
      }
    }
  }
}"""
EDITS = """query($o: String!, $r: String!, $p: Int!, $endCursor: String) {
  repository(owner: $o, name: $r) {
    pullRequest(number: $p) {
      userContentEdits(first: 100, after: $endCursor) {
        pageInfo { hasNextPage endCursor }
        nodes { editor { __typename login } diff editedAt }
      }
    }
  }
}"""
REPLY = """mutation($t: ID!, $b: String!) {
  addPullRequestReviewThreadReply(input: {pullRequestReviewThreadId: $t, body: $b}) {
    comment { id }
  }
}"""
BLOB = """query($o: String!, $r: String!, $e: String!) {
  repository(owner: $o, name: $r) { object(expression: $e) { oid } }
}"""
RESOLVE = """mutation($t: ID!) {
  resolveReviewThread(input: {threadId: $t}) { thread { id } }
}"""
THREAD_NODES = (
    ".data.repository.pullRequest.reviewThreads.nodes[]"
    " | {id, path, resolved: .isResolved, comments: .comments.nodes} | tojson"
)


COMMAND = re.compile(r"@sedosaur\s+dismiss\b", re.IGNORECASE)
"""The reply on the thread of a finding that dismisses it."""
WRITE = {"admin", "write"}
"""The repository permissions that can dismiss a finding. GitHub reports maintain as
write."""
AGENTS = frozenset({"synnax-claude-code", "synnax-codex"})
"""The logins of the agents whose rebuttals the bot judges."""
DISMISSED = "Every finding of the review is dismissed or withdrawn."
SETTLED = "The review of {head} finds this fixed: {reason}"
ANSWER_START = "<!-- sedosaur_answer "
ANSWER_END = " -->"
# The REST fields of a comment, in the shape that GraphQL serves.
_FIXED = re.compile(r"(?:\*agent reply\.\*\s*)?fixed (?:in|by)\b", re.IGNORECASE)
_COMMENT = (
    "{id, login: .user.login, author: {__typename: .user.type, login: .user.login}, "
    "body}"
)

_SUMMARY = re.compile(
    f"{re.escape(render.SUMMARY_START)}.*?{re.escape(render.SUMMARY_END)}", re.DOTALL
)


def gh(*args: str, stdin: str | None = None) -> str:
    return subprocess.run(
        ["gh", *args], input=stdin, stdout=subprocess.PIPE, text=True, check=True
    ).stdout


@dataclass(frozen=True, kw_only=True)
class Target:
    """The pull request and commits a review covers."""

    repo: str
    pr: int
    head: str
    base: str
    check_run: int | None
    """The check run that the review completes, or None for a replay, which posts
    nothing."""
    run_url: str
    bot: str
    """The slug of the GitHub App the review posts as."""


def post(rec: dict[str, Any], *, target: Target, gh: GH) -> None:
    """Posts the review of a record, without the findings that someone with write
    access dismissed or that the bot withdrew. A review with no finding left scores 5,
    and so does the head when the rest are dismissed or withdrawn while the review
    posts. Findings on lines of the diff become one review of inline threads, skipping
    any finding an unresolved thread of the bot already holds. Such a thread gets the
    finding's new heading when its severity changed, and keeps the rest of its comment.
    The findings outside the diff go in one comment the bot edits on each review. The
    check run takes its output before any other write. Then the summary in the pull
    request body is replaced and the check run completes. Last, while the target is
    still the head of the pull request, each open thread of the bot whose finding the
    review marks fixed, and does not report, is resolved with the reason as a reply. A
    failure there only warns.
    """
    t = target
    review = parse(rec["review"])
    found = threads(t, gh)
    can = authority(t, gh)
    gone = {(d.path, d.title) for d in dismissals(t, gh, found, can)}
    reported = {(f.path, render.title(render.heading(f))) for f in review.findings}
    review = _drop(review, gone)
    run = f"repos/{t.repo}/check-runs/{t.check_run}"
    # The summary links the check run, so its output lands first.
    write(gh, "PATCH", run, {"output": render.check(review)})
    blob = f"https://github.com/{t.repo}/blob/{t.head}"
    diff = _diff_lines(t, gh)
    inline = [f for f in review.findings if set(f.lines) <= diff.get(f.path, set())]
    posted = {(th.path, th.title): th for th in found if not th.resolved}
    fresh = [f for f in inline if (f.path, f.title) not in posted]
    for f in inline:
        th = posted.get((f.path, f.title))
        if th is not None and th.heading != render.heading(f):
            body = render.heading(f) + th.body[len(th.heading) :]
            comment = f"repos/{t.repo}/pulls/comments/{th.comment}"
            write(gh, "PATCH", comment, {"body": body})
    if fresh:
        comments = [_anchor(f) | {"body": render.comment(f, blob=blob)} for f in fresh]
        payload = {
            "commit_id": t.head,
            "event": "COMMENT",
            "body": "",
            "comments": comments,
        }
        write(gh, "POST", f"repos/{t.repo}/pulls/{t.pr}/reviews", payload)
    outside = [f for f in review.findings if f not in inline]
    _upsert_outside(t, gh, render.outside(outside, blob=blob), created=bool(outside))
    message = gh("api", f"repos/{t.repo}/commits/{t.head}", "--jq", ".commit.message")
    body = gh("api", f"repos/{t.repo}/pulls/{t.pr}", "--jq", '.body // ""')
    block = render.summary(
        review,
        count=render.reviews(body) + 1,
        commit=f"https://github.com/{t.repo}/commit/{t.head}",
        message=message,
        run_url=t.run_url,
        blob=blob,
    )
    body = render.splice(body.removesuffix("\n"), block)
    write(gh, "PATCH", f"repos/{t.repo}/pulls/{t.pr}", {"body": body})
    write(gh, "PATCH", run, {"status": "completed", "conclusion": "success"})
    try:
        fixed = {
            (f.path, f.title): f.reason
            for f in review.fixed
            if (f.path, f.title) not in reported | gone
        }
        _settle(t, gh, found, fixed)
    except subprocess.CalledProcessError as e:
        print(f"::warning::Could not resolve settled threads: {e}", file=sys.stderr)
    if review.findings:
        recheck(t, gh, can)


def _drop(review: Review, gone: set[tuple[str, str]]) -> Review:
    kept = [f for f in review.findings if (f.path, f.title) not in gone]
    if len(kept) == len(review.findings):
        return review
    if kept:
        return replace(review, findings=kept)
    return replace(review, score=5, verdict=DISMISSED, findings=kept)


def _commentable(patch: str) -> set[int]:
    """Returns the lines of the new file that a review comment can anchor to."""
    lines: set[int] = set()
    n = 0
    for row in patch.splitlines():
        if m := re.match(r"@@ -\d+(?:,\d+)? \+(\d+)", row):
            n = int(m[1])
        elif row.startswith(("+", " ")):
            lines.add(n)
            n += 1
    return lines


def write(gh: GH, method: str, path: str, payload: dict[str, Any]) -> None:
    gh("api", "-X", method, path, "--input", "-", stdin=json.dumps(payload))


def _diff_lines(t: Target, gh: GH) -> dict[str, set[int]]:
    out = gh(
        "api",
        f"repos/{t.repo}/compare/{t.base}...{t.head}",
        "--jq",
        ".files[] | {filename, patch} | tojson",
    )
    files = [json.loads(line) for line in out.splitlines() if line]
    return {f["filename"]: _commentable(f["patch"] or "") for f in files}


@dataclass(frozen=True, kw_only=True)
class Thread:
    """A review thread that a finding of the bot started."""

    id: str
    path: str
    title: str
    body: str
    """The finding's comment."""
    comment: int
    """The database ID of the finding's comment."""
    resolved: bool
    replies: list[dict[str, Any]]
    """The comments after the finding, oldest first, as GraphQL serves them."""

    @property
    def heading(self) -> str:
        """The first line of the finding's comment."""
        return self.body.split("\n", 1)[0]


def threads(t: Target, gh: GH) -> list[Thread]:
    """Returns each review thread that a finding of the bot started."""
    owner, name = t.repo.split("/")
    out = gh(
        "api",
        "graphql",
        "--paginate",
        "-f",
        f"query={THREADS}",
        "-f",
        f"o={owner}",
        "-f",
        f"r={name}",
        "-F",
        f"p={t.pr}",
        "--jq",
        THREAD_NODES,
    )
    found: list[Thread] = []
    for line in out.splitlines():
        node = json.loads(line)
        if not node["comments"]:
            continue
        first, *replies = node["comments"]
        if (first["author"] or {}).get("login") != t.bot:
            continue
        if title := render.title(first["body"]):
            found.append(
                Thread(
                    id=node["id"],
                    path=node["path"],
                    title=title,
                    body=first["body"],
                    comment=first["databaseId"],
                    resolved=node["resolved"],
                    replies=replies,
                )
            )
    return found


def _settle(
    t: Target, gh: GH, found: list[Thread], fixed: dict[tuple[str, str], str]
) -> None:
    done = [th for th in found if not th.resolved and (th.path, th.title) in fixed]
    if not done:
        return
    pr = f"repos/{t.repo}/pulls/{t.pr}"
    if gh("api", pr, "--jq", ".head.sha").strip() != t.head:
        return
    for th in done:
        resolve(gh, th)
        reason = fixed[(th.path, th.title)]
        reply(gh, th, SETTLED.format(head=t.head, reason=reason))


def reply(gh: GH, thread: Thread, body: str) -> None:
    gh(
        "api",
        "graphql",
        "-f",
        f"query={REPLY}",
        "-f",
        f"t={thread.id}",
        "-f",
        f"b={body}",
    )


def resolve(gh: GH, thread: Thread) -> None:
    gh("api", "graphql", "-f", f"query={RESOLVE}", "-f", f"t={thread.id}")


def dismissed(found: list[Thread], can: Authority) -> list[Dismissal]:
    """Returns each finding that someone with write access dismissed in one of the
    threads."""
    out: list[Dismissal] = []
    for th in found:
        reasons = (command(c["body"]) for c in th.replies if can(c))
        reason = next((r for r in reasons if r is not None), None)
        if reason is not None:
            out.append(Dismissal(path=th.path, title=th.title, reason=reason))
    return out


def withdrawn(t: Target, gh: GH, found: list[Thread]) -> list[Dismissal]:
    """Returns each finding that the bot withdrew on its thread in the threads after a
    rebuttal, while the finding's file at the head is the file the bot judged."""
    out: list[Dismissal] = []
    for th in found:
        records = [
            _record(c["body"])
            for c in th.replies
            if by_bot(t, c) and c["body"].startswith(ANSWER_START)
        ]
        records = [r for r in records if r.get("withdrawn")]
        if records and records[-1]["blob"] == blob(t, gh, t.head, th.path):
            reason = records[-1]["reason"]
            out.append(
                Dismissal(path=th.path, title=th.title, reason=reason, withdrawn=True)
            )
    return out


def dismissals(
    t: Target, gh: GH, found: list[Thread], can: Authority
) -> list[Dismissal]:
    """Returns each finding that someone with write access dismissed, on its thread in
    the threads or by its title in a record of the bot, and each finding that the bot
    withdrew and that still holds."""
    bound = [a for a in answers(t, comments(t, gh)) if "path" in a]
    return (
        dismissed(found, can)
        + [
            Dismissal(path=a["path"], title=a["title"], reason=a["reason"])
            for a in bound
        ]
        + withdrawn(t, gh, found)
    )


def answers(t: Target, found: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Returns the record in each of the bot's answers to a dismissal, from the comments
    on the pull request. A record of a dismissal holds the `path`, `title`, and `reason`
    of its finding."""
    return [
        _record(c["body"])
        for c in found
        if c["login"] == f"{t.bot}[bot]" and c["body"].startswith(ANSWER_START)
    ]


def _record(body: str) -> dict[str, Any]:
    data: dict[str, Any] = json.loads(body[len(ANSWER_START) :].split(ANSWER_END)[0])
    return data


def by_bot(t: Target, comment: dict[str, Any]) -> bool:
    """Reports whether the bot wrote a comment in the shape that GraphQL serves."""
    return (comment["author"] or {}).get("login") == t.bot


def pending(t: Target, th: Thread) -> list[dict[str, Any]]:
    """Returns the replies of a thread after the last reply of the bot."""
    last = max((i for i, r in enumerate(th.replies) if by_bot(t, r)), default=-1)
    return th.replies[last + 1 :]


def rebutted(t: Target, th: Thread, can: Authority) -> bool:
    """Reports whether an open thread waits for the bot to judge a rebuttal: a reply
    after the bot's last verdict, from someone with write access or an agent, that is
    not a note of a fix. A dismiss command of someone with write access among those
    replies takes precedence."""
    last = max((i for i, r in enumerate(th.replies) if ruling(t, r)), default=-1)
    replies = th.replies[last + 1 :]
    if th.resolved or any(command(r["body"]) is not None and can(r) for r in replies):
        return False
    return any(
        trusted(r, can) and not _FIXED.match(_unquoted(r["body"]).strip())
        for r in replies
    )


def ruling(t: Target, comment: dict[str, Any]) -> bool:
    """Reports whether a comment is the bot's verdict on a rebuttal, a hold or a
    withdrawal, from the record in it."""
    if not by_bot(t, comment) or not comment["body"].startswith(ANSWER_START):
        return False
    record = _record(comment["body"])
    return bool(record.get("held") or record.get("withdrawn"))


def trusted(comment: dict[str, Any], can: Authority) -> bool:
    """Reports whether someone with write access or an agent wrote a comment, so the
    bot judges it as a rebuttal."""
    return can(comment) or (comment["author"] or {}).get("login") in AGENTS


def blob(t: Target, gh: GH, commit: str, path: str) -> str | None:
    """Returns the ID of the blob of a file at a commit, or None when it has none."""
    owner, name = t.repo.split("/")
    out = gh(
        "api",
        "graphql",
        "-f",
        f"query={BLOB}",
        "-f",
        f"o={owner}",
        "-f",
        f"r={name}",
        "-f",
        f"e={commit}:{path}",
        "--jq",
        ".data.repository.object.oid",
    ).strip()
    return None if out in ("", "null") else out


def answer(record: dict[str, Any], text: str) -> str:
    """Renders a comment of the bot that holds a record `answers` reads."""
    # A ">" in a title would end the HTML comment early.
    data = json.dumps(record).replace(">", "\\u003e")
    return f"{ANSWER_START}{data}{ANSWER_END}\n\n{text}"


def command(body: str) -> str | None:
    """Returns the text of a reply beside the dismiss command, or None when the reply
    holds no command. Quoted lines do not count."""
    text = _unquoted(body)
    if not COMMAND.search(text):
        return None
    return COMMAND.sub("", text).strip().lstrip(",:;").strip()


def _unquoted(body: str) -> str:
    return "\n".join(r for r in body.splitlines() if not r.lstrip().startswith(">"))


def authority(t: Target, gh: GH) -> Authority:
    """Returns an `Authority` that accepts a person with write access to the repository,
    and never a bot. It asks GitHub once for each login."""
    known: dict[str, bool] = {}

    def can(comment: dict[str, Any]) -> bool:
        author = comment["author"]
        if author is None or author["__typename"] != "User":
            return False
        login = author["login"]
        if login not in known:
            path = f"repos/{t.repo}/collaborators/{login}/permission"
            known[login] = gh("api", path, "--jq", ".permission").strip() in WRITE
        return known[login]

    return can


def newest(t: Target, gh: GH, commit: str, key: str) -> dict[str, Any] | None:
    """Returns the newest check run of the bot's reviews of a commit for a key, or None.

    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    """
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


def current(run: dict[str, Any]) -> bool:
    """Reports whether a check run of the bot holds a review under the current rules.
    A review under older rules can score 5 with findings open."""
    return render.RULES in (run["output"]["text"] or "")


def reviewed(t: Target, gh: GH) -> list[str]:
    """Returns each commit that the bot reviewed, newest first, from the revisions of
    the pull request body."""
    commits: list[str] = []
    for block in summaries(t, gh):
        found = re.findall(r"/commit/([0-9a-f]{40})", block)
        if found and found[-1] not in commits:
            commits.append(found[-1])
    return commits


def summaries(t: Target, gh: GH) -> list[str]:
    """Returns each summary that the bot wrote in the pull request body, newest first,
    from the revisions of the body."""
    owner, name = t.repo.split("/")
    out = gh(
        "api",
        "graphql",
        "--paginate",
        "-f",
        f"query={EDITS}",
        "-f",
        f"o={owner}",
        "-f",
        f"r={name}",
        "-F",
        f"p={t.pr}",
        "--jq",
        ".data.repository.pullRequest.userContentEdits.nodes[] | tojson",
    )
    blocks: list[str] = []
    for line in out.splitlines():
        e = json.loads(line)
        block = _SUMMARY.search(e["diff"] or "")
        if (e["editor"] or {}).get("login") == t.bot and block is not None:
            blocks.append(block[0])
    return blocks


def _anchor(f: Finding) -> dict[str, Any]:
    anchor: dict[str, Any] = {"path": f.path, "line": f.line, "side": "RIGHT"}
    if f.start_line is not None and f.start_line < f.line:
        anchor |= {"start_line": f.start_line, "start_side": "RIGHT"}
    return anchor


def outside_findings(t: Target, gh: GH) -> set[tuple[str, str]]:
    """Returns the path and title of each finding that the bot's comment of findings
    outside the diff lists."""
    c = _outside(t, gh)
    return set() if c is None else render.outside_listed(c["body"])


def comments(t: Target, gh: GH) -> list[dict[str, Any]]:
    """Returns the comments on the pull request in the shape that GraphQL serves."""
    out = gh(
        "api",
        "--paginate",
        f"repos/{t.repo}/issues/{t.pr}/comments",
        "--jq",
        f".[] | {_COMMENT} | tojson",
    )
    return [json.loads(s) for s in out.splitlines() if s]


def _outside(t: Target, gh: GH) -> dict[str, Any] | None:
    for c in comments(t, gh):
        if c["login"] == f"{t.bot}[bot]" and c["body"].startswith(render.OUTSIDE):
            return c
    return None


def _upsert_outside(t: Target, gh: GH, body: str, *, created: bool) -> None:
    c = _outside(t, gh)
    if c is not None:
        write(gh, "PATCH", f"repos/{t.repo}/issues/comments/{c['id']}", {"body": body})
    elif created:
        write(gh, "POST", f"repos/{t.repo}/issues/{t.pr}/comments", {"body": body})


def recheck(t: Target, gh: GH, can: Authority) -> None:
    """Runs `rescore` once the check run of the target completes, for a dismissal that
    found it running. The head has its review, so a failure only warns."""
    try:
        run = f"repos/{t.repo}/check-runs/{t.check_run}"
        key = gh("api", run, "--jq", ".external_id").strip()
        rescore(t, gh, threads(t, gh), key, can)
    except subprocess.CalledProcessError as e:
        print(
            f"::warning::Could not read dismissals after the review: {e}",
            file=sys.stderr,
        )


def rescore(t: Target, gh: GH, found: list[Thread], key: str, can: Authority) -> bool:
    """Scores the head 5/5 in a new check run when every finding of its newest review is
    dismissed by someone with write access, in the threads or in the bot's records, or
    withdrawn by the bot. Returns whether it did.

    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    """
    run = newest(t, gh, t.head, key)
    if run is None or run["conclusion"] != "success" or not current(run):
        return False
    if run["output"]["title"] == render.scored(5):
        return False
    listed = render.listed(run["output"]["text"] or "")
    if not listed:
        return False
    gone = {(d.path, d.title) for d in dismissals(t, gh, found, can)}
    if not listed <= gone:
        return False
    output = {
        "title": render.scored(5),
        "summary": f"{DISMISSED}\n\n{run['output']['summary']}",
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
