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
from review.findings import Dismissal, Finding, Review

GH = Callable[..., str]
"""Runs `gh` with the given arguments and optional stdin, and returns its stdout."""

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
            nodes { databaseId author { __typename login } authorAssociation body }
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
        nodes { editor { __typename login } diff }
      }
    }
  }
}"""
THREAD_NODES = (
    ".data.repository.pullRequest.reviewThreads.nodes[]"
    " | {id, path, resolved: .isResolved, comments: .comments.nodes} | tojson"
)


COMMAND = re.compile(r"@sedosaur\s+dismiss\b", re.I)
"""The reply on the thread of a finding that dismisses it."""
MEMBERS = {"OWNER", "MEMBER", "COLLABORATOR"}
DISMISSED = "A member dismissed every finding of the review that blocks merge."

_SUMMARY = re.compile(
    f"{re.escape(render.SUMMARY_START)}.*?{re.escape(render.SUMMARY_END)}", re.S
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


def post(review: Review, *, target: Target, gh: GH) -> None:
    """Posts a review, without the findings that a member dismissed. A review with no
    blocking finding left scores 5, and so does the head when the rest are dismissed
    while the review posts. Findings on lines of the diff become one review of inline
    threads, skipping any finding an unresolved thread of the bot already holds. Such a
    thread gets the finding's new heading when its severity or follow-up mark changed,
    and keeps the rest of its comment. The findings outside the diff go in one comment
    the bot edits on each review. The summary in the pull request body is replaced, and
    the check run completes.
    """
    t = target
    found = threads(t, gh)
    review = _drop(review, {(d.path, d.title) for d in dismissed(found)})
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
    done = {
        "status": "completed",
        "conclusion": "success",
        "output": render.check(review),
    }
    write(gh, "PATCH", f"repos/{t.repo}/check-runs/{t.check_run}", done)
    if any(f.blocks for f in review.findings):
        recheck(t, gh)


def _drop(review: Review, gone: set[tuple[str, str]]) -> Review:
    kept = [f for f in review.findings if (f.path, f.title) not in gone]
    if len(kept) == len(review.findings):
        return review
    if any(f.blocks for f in kept) or not any(f.blocks for f in review.findings):
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


def dismissed(found: list[Thread]) -> list[Dismissal]:
    """Returns each finding that a member dismissed in one of the threads."""
    out: list[Dismissal] = []
    for th in found:
        reasons = (command(c["body"]) for c in th.replies if member(c))
        reason = next((r for r in reasons if r is not None), None)
        if reason is not None:
            out.append(Dismissal(path=th.path, title=th.title, reason=reason))
    return out


def command(body: str) -> str | None:
    """Returns the text of a reply beside the dismiss command, or None when the reply
    holds no command. Quoted lines do not count."""
    text = "\n".join(r for r in body.splitlines() if not r.lstrip().startswith(">"))
    if not COMMAND.search(text):
        return None
    return COMMAND.sub("", text).strip().lstrip(",:;").strip()


def member(comment: dict[str, Any]) -> bool:
    """Reports whether a person who is a member of the repository wrote a comment."""
    author = comment["author"]
    return (
        author is not None
        and author["__typename"] == "User"
        and comment["authorAssociation"] in MEMBERS
    )


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


def reviewed(t: Target, gh: GH) -> list[str]:
    """Returns each commit that the bot reviewed, newest first, from the revisions of
    the pull request body."""
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
    commits: list[str] = []
    for line in out.splitlines():
        e = json.loads(line)
        block = _SUMMARY.search(e["diff"] or "")
        if (e["editor"] or {}).get("login") != t.bot or block is None:
            continue
        found = re.findall(r"/commit/([0-9a-f]{40})", block[0])
        if found and found[-1] not in commits:
            commits.append(found[-1])
    return commits


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


def _outside(t: Target, gh: GH) -> dict[str, Any] | None:
    out = gh(
        "api",
        "--paginate",
        f"repos/{t.repo}/issues/{t.pr}/comments",
        "--jq",
        ".[] | {id, login: .user.login, body} | tojson",
    )
    comments: list[dict[str, Any]] = [json.loads(s) for s in out.splitlines() if s]
    for c in comments:
        if c["login"] == f"{t.bot}[bot]" and c["body"].startswith(render.OUTSIDE):
            return c
    return None


def _upsert_outside(t: Target, gh: GH, body: str, *, created: bool) -> None:
    c = _outside(t, gh)
    if c is not None:
        write(gh, "PATCH", f"repos/{t.repo}/issues/comments/{c['id']}", {"body": body})
    elif created:
        write(gh, "POST", f"repos/{t.repo}/issues/{t.pr}/comments", {"body": body})


def recheck(t: Target, gh: GH) -> None:
    """Runs `rescore` once the check run of the target completes, for a dismissal that
    found it running. The head has its review, so a failure only warns."""
    try:
        run = f"repos/{t.repo}/check-runs/{t.check_run}"
        key = gh("api", run, "--jq", ".external_id").strip()
        rescore(t, gh, threads(t, gh), key)
    except subprocess.CalledProcessError as e:
        print(
            f"::warning::Could not read dismissals after the review: {e}",
            file=sys.stderr,
        )


def rescore(t: Target, gh: GH, found: list[Thread], key: str) -> bool:
    """Scores the head 5/5 in a new check run when a member dismissed every blocking
    finding of its newest review in the threads. Returns whether it did.

    :param key: The external ID of the review's check run, `<pr>:<base ref>`.
    """
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
