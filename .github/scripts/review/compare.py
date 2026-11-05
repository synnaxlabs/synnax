#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Compares Sedosaur's reviews with Greptile's reviews of the same commits.

A finding is real when it has more 👍 than 👎, or when its thread was resolved after a
later commit changed its file. It is noise when it has more 👎 than 👍, or when its
thread was resolved with no change to its file. Otherwise it is open. A finding that
matches one of the other reviewer's takes that one's verdict while its own is open.
"""

import json
import re
import tempfile
from collections import defaultdict
from dataclasses import dataclass, replace
from datetime import datetime
from pathlib import Path
from typing import Any

from review import render
from review.post import GH

GREPTILE = "greptile-apps"
SEDOSAUR = "sedosaur"
ARTIFACT = "sedosaur-review"
NEAR = 5
"""Findings on one file match when their lines are at most this far apart."""

REAL, NOISE, OPEN = "real", "noise", "open"

PULL = """query($o: String!, $r: String!, $p: Int!) {
  repository(owner: $o, name: $r) {
    pullRequest(number: $p) {
      headRefOid
      userContentEdits(first: 100) { nodes { editor { __typename login } diff } }
      reviewThreads(first: 100) {
        nodes {
          isResolved path line originalLine
          comments(first: 1) {
            nodes {
              databaseId author { login } body originalCommit { oid }
              reactionGroups { content reactors { totalCount } }
            }
          }
        }
      }
    }
  }
}"""

_BLOCK = re.compile(r"<!-- greptile_comment -->(.*?)<!-- /greptile_comment -->", re.S)
_ITEM = re.compile(
    r'^\d+\. <img alt="(P\d)"[^>]*>.*?\*\*(.+?)\*\*(?:.*?#discussion_r(\d+))?', re.M
)


@dataclass(frozen=True, kw_only=True)
class Thread:
    """The first comment of a review thread, and how the thread ended."""

    id: int
    bot: str
    path: str
    line: int | None
    commit: str
    title: str | None
    resolved: bool
    up: int
    down: int


@dataclass(frozen=True, kw_only=True)
class Item:
    """A finding of one review, with where it is and whether it was real."""

    severity: str
    title: str
    path: str | None
    line: int | None
    verdict: str


@dataclass(frozen=True, kw_only=True)
class Pair:
    """Greptile's and Sedosaur's reviews of one commit."""

    pr: int
    head: str
    policy: str
    scores: tuple[int, int]
    greptile: list[Item]
    sedosaur: list[Item]
    matches: list[tuple[int, int]]
    cost_usd: float
    duration_ms: int


def greptile_reviews(
    edits: list[dict[str, Any]],
) -> dict[str, tuple[int, list[tuple[str, str, int | None]]]]:
    """Returns Greptile's score and findings for each commit it reviewed.

    :param edits: The revisions of a pull request body, newest first.
    :returns: For each commit, the score and each finding's severity, title, and the
        ID of the comment that starts its thread, if it has one.
    """
    out: dict[str, tuple[int, list[tuple[str, str, int | None]]]] = {}
    for e in edits:
        editor = e["editor"] or {}
        if editor.get("__typename") != "Bot" or editor.get("login") != GREPTILE:
            continue
        block = _BLOCK.search(e["diff"] or "")
        if block is None:
            continue
        commits = re.findall(r"/commit/([0-9a-f]{40})", block[1])
        score = re.search(r"greptile_confidence_score:(\d)", block[1])
        if not commits or score is None or commits[-1] in out:
            continue
        items = [
            (severity, title, int(d) if d else None)
            for severity, title, d in _ITEM.findall(block[1])
        ]
        out[commits[-1]] = (int(score[1]), items)
    return out


def threads(nodes: list[dict[str, Any]]) -> list[Thread]:
    """Returns the review threads that either reviewer started."""
    out: list[Thread] = []
    for n in nodes:
        [c] = n["comments"]["nodes"]
        login = (c["author"] or {}).get("login")
        if login not in (GREPTILE, SEDOSAUR):
            continue
        votes = {g["content"]: g["reactors"]["totalCount"] for g in c["reactionGroups"]}
        out.append(
            Thread(
                id=c["databaseId"],
                bot=login,
                path=n["path"],
                line=n["line"] or n["originalLine"],
                commit=c["originalCommit"]["oid"],
                title=render.title(c["body"]),
                resolved=n["isResolved"],
                up=votes.get("THUMBS_UP", 0),
                down=votes.get("THUMBS_DOWN", 0),
            )
        )
    return out


def verdict(t: Thread, touched: set[str]) -> str:
    """Decides whether a thread's finding was real.

    :param touched: The files that changed after the thread's commit.
    """
    if t.down > t.up:
        return NOISE
    if t.up > t.down:
        return REAL
    if not t.resolved:
        return OPEN
    return REAL if t.path in touched else NOISE


def match(a: list[Item], b: list[Item]) -> list[tuple[int, int]]:
    """Pairs the findings of two reviews that sit on one file within NEAR lines,
    closest first. Each finding joins at most one pair."""
    near = sorted(
        (abs(x.line - y.line), i, j)
        for i, x in enumerate(a)
        for j, y in enumerate(b)
        if x.path is not None
        and x.path == y.path
        and x.line is not None
        and y.line is not None
        and abs(x.line - y.line) <= NEAR
    )
    used_a: set[int] = set()
    used_b: set[int] = set()
    out: list[tuple[int, int]] = []
    for _, i, j in near:
        if i not in used_a and j not in used_b:
            used_a.add(i)
            used_b.add(j)
            out.append((i, j))
    return out


def records(gh: GH, repo: str, since: datetime) -> list[dict[str, Any]]:
    """Downloads the records of the Sedosaur reviews since a time, oldest first."""
    listed = gh(
        "api",
        "--paginate",
        f"repos/{repo}/actions/artifacts?name={ARTIFACT}&per_page=100",
        "--jq",
        ".artifacts[] | select(.expired | not)"
        ' | "\\(.created_at) \\(.workflow_run.id)"',
    )
    runs = sorted(tuple(line.split()) for line in listed.splitlines() if line)
    out: list[dict[str, Any]] = []
    with tempfile.TemporaryDirectory() as tmp:
        for created, run in runs:
            if datetime.fromisoformat(created) < since:
                continue
            into = Path(tmp) / run
            gh(
                "run",
                "download",
                run,
                "--repo",
                repo,
                "--name",
                ARTIFACT,
                "--dir",
                str(into),
            )
            out.append(json.loads((into / "review.json").read_text()))
    return out


def pairs(gh: GH, repo: str, recs: list[dict[str, Any]]) -> list[Pair]:
    """Pairs each Sedosaur review with Greptile's review of the same commit.

    :param recs: Review records, oldest first. The newest record of a commit counts.
    :returns: A pair for each commit that both reviewers reviewed.
    """
    latest = {(r["pr"], r["head"]): r for r in recs}
    out: list[Pair] = []
    for pr in sorted({pr for pr, _ in latest}):
        heads = {head: r for (n, head), r in latest.items() if n == pr}
        out += _pull(gh, repo, pr, heads)
    return out


def _pull(gh: GH, repo: str, pr: int, heads: dict[str, dict[str, Any]]) -> list[Pair]:
    owner, name = repo.split("/")
    data = json.loads(
        gh(
            "api",
            "graphql",
            "-f",
            f"query={PULL}",
            "-f",
            f"o={owner}",
            "-f",
            f"r={name}",
            "-F",
            f"p={pr}",
        )
    )["data"]["repository"]["pullRequest"]
    final = data["headRefOid"]
    changed: dict[str, set[str]] = {final: set()}

    def judge(t: Thread | None) -> str:
        if t is None:
            return OPEN
        if t.commit not in changed:
            files = gh(
                "api",
                f"repos/{repo}/compare/{t.commit}...{final}",
                "--jq",
                ".files[].filename",
            )
            changed[t.commit] = set(files.splitlines())
        return verdict(t, changed[t.commit])

    reviews = greptile_reviews(data["userContentEdits"]["nodes"])
    ts = threads(data["reviewThreads"]["nodes"])
    by_id = {t.id: t for t in ts}
    ours = {(t.path, t.title): t for t in ts if t.bot == SEDOSAUR}
    out: list[Pair] = []
    for head, rec in heads.items():
        if head not in reviews:
            continue
        score, found = reviews[head]
        g: list[Item] = []
        for severity, title, d in found:
            t = by_id.get(d) if d is not None else None
            g.append(
                Item(
                    severity=severity,
                    title=title,
                    path=t.path if t else None,
                    line=t.line if t else None,
                    verdict=judge(t),
                )
            )
        s = [
            Item(
                severity=f["severity"],
                title=f["title"],
                path=f["path"],
                line=f["line"],
                verdict=judge(ours.get((f["path"], f["title"]))),
            )
            for f in rec["review"]["findings"]
        ]
        matches = match(g, s)
        for i, j in matches:
            if g[i].verdict == OPEN:
                g[i] = replace(g[i], verdict=s[j].verdict)
            elif s[j].verdict == OPEN:
                s[j] = replace(s[j], verdict=g[i].verdict)
        out.append(
            Pair(
                pr=pr,
                head=head,
                policy=rec["policy"],
                scores=(score, rec["review"]["score"]),
                greptile=g,
                sedosaur=s,
                matches=matches,
                cost_usd=rec["usage"]["cost_usd"],
                duration_ms=rec["usage"]["duration_ms"],
            )
        )
    return out


def _share(n: int, d: int) -> str:
    return f"{n} of {d} ({round(100 * n / d)}%)" if d else "none yet"


def _count(n: int, noun: str) -> str:
    return f"{n} {noun}" if n == 1 else f"{n} {noun}s"


def _where(i: Item) -> str:
    return f"`{i.path}:{i.line}`" if i.path else "outside the diff"


def report(ps: list[Pair], *, since: datetime) -> str:
    """Renders the comparison of the paired reviews as Markdown."""
    lines = [f"## Sedosaur trial since {since:%Y-%m-%d %H:%M} UTC", ""]
    if not ps:
        return "\n".join([*lines, "No commit has a review from both yet."])
    g = [i for p in ps for i in p.greptile]
    s = [i for p in ps for i in p.sedosaur]
    g_matched = [i for p in ps for i, _ in p.matches if p.greptile[i].verdict == REAL]
    missed = [
        (p, x)
        for p in ps
        for i, x in enumerate(p.greptile)
        if x.verdict == REAL and i not in {m for m, _ in p.matches}
    ]
    only = [
        (p, x)
        for p in ps
        for j, x in enumerate(p.sedosaur)
        if x.verdict == REAL and j not in {m for _, m in p.matches}
    ]
    agree = sum(p.scores[0] == p.scores[1] for p in ps)
    higher = sum(p.scores[1] > p.scores[0] for p in ps)

    def counts(items: list[Item]) -> list[int]:
        return [sum(i.verdict == v for i in items) for v in (REAL, NOISE, OPEN)]

    (gr, gn, go), (sr, sn, so) = counts(g), counts(s)
    prs = _count(len({p.pr for p in ps}), "pull request")
    lines += [
        f"{_count(len(ps), 'commit')} on {prs} have a review from both.",
        "",
        "| | Greptile | Sedosaur |",
        "| --- | --- | --- |",
        f"| Findings | {len(g)} | {len(s)} |",
        f"| Real | {gr} | {sr} |",
        f"| Noise | {gn} | {sn} |",
        f"| Open | {go} | {so} |",
        f"| Precision | {_share(gr, gr + gn)} | {_share(sr, sr + sn)} |",
        "",
        f"- **Scores:** equal on {_share(agree, len(ps))}. Sedosaur was higher on "
        f"{higher} and lower on {len(ps) - agree - higher}.",
        f"- **Coverage:** Sedosaur found {_share(len(g_matched), gr)} real Greptile "
        "findings."
        if gr
        else "- **Coverage:** no real Greptile findings yet.",
        f"- **Only Sedosaur:** {_count(len(only), 'real finding')}.",
        f"- **Cost:** ${sum(p.cost_usd for p in ps):.2f} for "
        f"{_count(len(ps), 'review')}, "
        f"{sum(p.duration_ms for p in ps) / len(ps) / 1000:.0f} s each on average.",
        "",
        "### Real Greptile findings Sedosaur missed",
        "",
        *(
            [f"- {x.severity} **{x.title}** {_where(x)} (#{p.pr})" for p, x in missed]
            or ["None."]
        ),
        "",
        "### Real findings only Sedosaur found",
        "",
        *(
            [f"- {x.severity} **{x.title}** {_where(x)} (#{p.pr})" for p, x in only]
            or ["None."]
        ),
        "",
        "### By policy",
        "",
        "| Policy | Reviews | Equal scores | Coverage |",
        "| --- | --- | --- | --- |",
    ]
    by_policy: dict[str, list[Pair]] = defaultdict(list)
    for p in ps:
        by_policy[p.policy].append(p)
    for policy, group in by_policy.items():
        real = sum(x.verdict == REAL for p in group for x in p.greptile)
        found = sum(p.greptile[i].verdict == REAL for p in group for i, _ in p.matches)
        same = sum(p.scores[0] == p.scores[1] for p in group)
        lines.append(
            f"| `{policy[:10]}` | {len(group)} | {_share(same, len(group))} "
            f"| {_share(found, real)} |"
        )
    lines += [
        "",
        "### Commits",
        "",
        "| PR | Commit | Greptile | Sedosaur | Both | Only Greptile | Only Sedosaur |",
        "| --- | --- | --- | --- | --- | --- | --- |",
    ]
    for p in ps:
        both = len(p.matches)
        lines.append(
            f"| #{p.pr} | `{p.head[:10]}` | {p.scores[0]}/5 | {p.scores[1]}/5 | {both} "
            f"| {len(p.greptile) - both} | {len(p.sedosaur) - both} |"
        )
    return "\n".join(lines)
