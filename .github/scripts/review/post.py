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
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

from review import render
from review.findings import Finding, Review

GH = Callable[..., str]
"""Runs `gh` with the given arguments and optional stdin, and returns its stdout."""

THREADS = """query($o: String!, $r: String!, $p: Int!, $endCursor: String) {
  repository(owner: $o, name: $r) {
    pullRequest(number: $p) {
      reviewThreads(first: 100, after: $endCursor) {
        pageInfo { hasNextPage endCursor }
        nodes { isResolved path comments(first: 1) { nodes { author { login } body } } }
      }
    }
  }
}"""
OPEN_THREADS = (
    ".data.repository.pullRequest.reviewThreads.nodes[] | select(.isResolved | not)"
    " | {path, first: .comments.nodes[0]} | tojson"
)


def gh(*args: str, stdin: str | None = None) -> str:
    return subprocess.run(
        ["gh", *args], input=stdin, capture_output=True, text=True, check=True
    ).stdout


@dataclass(frozen=True, kw_only=True)
class Target:
    """The pull request and commits a review covers."""

    repo: str
    pr: int
    head: str
    base: str
    check_run: int
    """The check run that the review completes."""
    run_url: str
    bot: str
    """The slug of the GitHub App the review posts as."""


def post(review: Review, *, target: Target, gh: GH) -> None:
    """Posts a review the way Greptile does. Findings on lines of the diff become one
    review of inline threads, skipping any finding an unresolved thread of the bot
    already holds. The rest go in one comment the bot edits on each review. The
    summary in the pull request body is replaced, and the check run completes.
    """
    t = target
    blob = f"https://github.com/{t.repo}/blob/{t.head}"
    diff = _diff_lines(t, gh)
    inline = [f for f in review.findings if set(f.lines) <= diff.get(f.path, set())]
    posted = open_findings(t, gh)
    fresh = [f for f in inline if (f.path, f.title) not in posted]
    if fresh:
        comments = [_anchor(f) | {"body": render.comment(f, blob=blob)} for f in fresh]
        payload = {
            "commit_id": t.head,
            "event": "COMMENT",
            "body": "",
            "comments": comments,
        }
        _write(gh, "POST", f"repos/{t.repo}/pulls/{t.pr}/reviews", payload)
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
    _write(gh, "PATCH", f"repos/{t.repo}/pulls/{t.pr}", {"body": body})
    done = {
        "status": "completed",
        "conclusion": "success",
        "output": render.check(review),
    }
    _write(gh, "PATCH", f"repos/{t.repo}/check-runs/{t.check_run}", done)


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


def _write(gh: GH, method: str, path: str, payload: dict[str, Any]) -> None:
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


def open_findings(t: Target, gh: GH) -> set[tuple[str, str]]:
    """Returns the path and title of each finding in an unresolved thread of the bot."""
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
        OPEN_THREADS,
    )
    keys: set[tuple[str, str]] = set()
    for line in out.splitlines():
        node = json.loads(line)
        first = node["first"]
        if first is None or (first["author"] or {}).get("login") != t.bot:
            continue
        if title := render.title(first["body"]):
            keys.add((node["path"], title))
    return keys


def _anchor(f: Finding) -> dict[str, Any]:
    anchor: dict[str, Any] = {"path": f.path, "line": f.line, "side": "RIGHT"}
    if f.start_line is not None and f.start_line < f.line:
        anchor |= {"start_line": f.start_line, "start_side": "RIGHT"}
    return anchor


def _upsert_outside(t: Target, gh: GH, body: str, *, created: bool) -> None:
    out = gh(
        "api",
        "--paginate",
        f"repos/{t.repo}/issues/{t.pr}/comments",
        "--jq",
        ".[] | {id, login: .user.login, body} | tojson",
    )
    for c in (json.loads(line) for line in out.splitlines() if line):
        if c["login"] == f"{t.bot}[bot]" and c["body"].startswith(render.OUTSIDE):
            path = f"repos/{t.repo}/issues/comments/{c['id']}"
            _write(gh, "PATCH", path, {"body": body})
            return
    if created:
        _write(gh, "POST", f"repos/{t.repo}/issues/{t.pr}/comments", {"body": body})
