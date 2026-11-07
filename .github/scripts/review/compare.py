#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Compares Sedosaur's reviews with Greptile's reviews of the same commits.

A finding is real when it has more 👍 than 👎, when a person replied that it is fixed,
or when its thread was resolved and a later commit of the pull request changed its file.
It is noise when it has more 👎 than 👍, or when its thread was resolved with no such
reply or change. Otherwise it is open. GitHub does not say when a thread was resolved,
so a commit made after the last reply to a thread does not count, and merge commits
never count. A finding that matches one of the other reviewer's takes that one's verdict
while its own is open.
"""

import json
import re
import subprocess
import tempfile
from collections import defaultdict
from collections.abc import Callable
from dataclasses import dataclass, replace
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from review import render
from review.post import EDITS, GH

Git = Callable[..., str]
"""Runs `git` in a clone of the repository with the given arguments, and returns its
stdout."""

GREPTILE = "greptile-apps"
SEDOSAUR = "sedosaur"
ARTIFACT = "sedosaur-review"
NEAR = 20
"""Findings on one file match when their lines are at most this far apart."""

REAL, NOISE, OPEN = "real", "noise", "open"

THREADS = """query($o: String!, $r: String!, $p: Int!, $endCursor: String) {
  repository(owner: $o, name: $r) {
    pullRequest(number: $p) {
      reviewThreads(first: 100, after: $endCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          isResolved path originalLine
          first: comments(first: 1) {
            nodes {
              databaseId author { login } body originalCommit { oid }
              reactionGroups { content reactors { totalCount } }
            }
          }
          last: comments(last: 100) { nodes { author { login } body createdAt } }
        }
      }
    }
  }
}"""

_FIXED = re.compile(r"^fixed\b", re.I | re.M)
"""A reply line that says a finding is fixed, such as "Fixed in 1a2b3c"."""
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
    """The line at the thread's commit."""
    commit: str | None
    """The commit the thread started on, or None when it no longer exists."""
    title: str | None
    resolved: bool
    up: int
    down: int
    last_reply: datetime | None
    """When a person last replied to the thread, if anyone did."""
    fixed: bool
    """A person replied that the finding is fixed, perhaps in another file or pull
    request."""


@dataclass(frozen=True, kw_only=True)
class Item:
    """A finding of one review, with where it is and whether it was real."""

    severity: str
    title: str
    path: str | None
    line: int | None
    verdict: str
    repeat: bool = False
    """Whether an earlier paired commit listed its thread. A repeat can match a finding
    of the other reviewer, but no count includes it."""


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
    bots = (GREPTILE, SEDOSAUR)
    out: list[Thread] = []
    for n in nodes:
        first = n["first"]["nodes"]
        if not first:
            continue
        c = first[0]
        login = (c["author"] or {}).get("login")
        if login not in bots:
            continue
        votes = {g["content"]: g["reactors"]["totalCount"] for g in c["reactionGroups"]}
        replies = [
            r
            for r in n["last"]["nodes"]
            if (r["author"] or {}).get("login") not in bots
        ]
        out.append(
            Thread(
                id=c["databaseId"],
                bot=login,
                path=n["path"],
                line=n["originalLine"],
                commit=(c["originalCommit"] or {}).get("oid"),
                title=render.title(c["body"]),
                resolved=n["isResolved"],
                up=votes.get("THUMBS_UP", 0),
                down=votes.get("THUMBS_DOWN", 0),
                last_reply=(
                    datetime.fromisoformat(max(r["createdAt"] for r in replies))
                    if replies
                    else None
                ),
                fixed=any(_FIXED.search(r["body"]) for r in replies),
            )
        )
    return out


def verdict(t: Thread, touched: set[str] | None) -> str:
    """Decides whether a thread's finding was real.

    :param touched: The files that the pull request changed after the thread's commit
        and before the last reply to it, or None when the commit is no longer in the
        history of the pull request.
    """
    if t.down > t.up:
        return NOISE
    if t.up > t.down or t.fixed:
        return REAL
    if not t.resolved or touched is None:
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
    """Downloads the records of the Sedosaur reviews since a time, oldest first. Only a
    run on `main` of the repository itself counts, so a fork cannot add a record."""
    listed = gh(
        "api",
        "--paginate",
        f"repos/{repo}/actions/artifacts?name={ARTIFACT}&per_page=100",
        "--jq",
        ".artifacts[] | tojson",
    )
    runs = sorted(
        (a["created_at"], str(w["id"]))
        for a in map(json.loads, listed.splitlines())
        if not a["expired"]
        and (w := a["workflow_run"])["head_branch"] == "main"
        and w["head_repository_id"] == w["repository_id"]
    )
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


def git(*args: str) -> str:
    return subprocess.run(
        ["git", *args], capture_output=True, text=True, check=True
    ).stdout


def pairs(gh: GH, git: Git, repo: str, recs: list[dict[str, Any]]) -> list[Pair]:
    """Pairs each Sedosaur review with Greptile's review of the same commit.

    :param git: Runs in a clone with full history. The pairing fetches the head of each
        pull request into it as `refs/review/<number>`.
    :param recs: Review records, oldest first. The newest record of a commit counts.
    :returns: A pair for each commit that both reviewers reviewed.
    """
    latest = {(r["pr"], r["head"]): r for r in recs}
    prs = sorted({pr for pr, _ in latest})
    if not prs:
        return []
    git(
        *("fetch", "--no-tags", "origin"),
        *(f"+refs/pull/{pr}/head:refs/review/{pr}" for pr in prs),
    )
    out: list[Pair] = []
    for pr in prs:
        heads = {head: r for (n, head), r in latest.items() if n == pr}
        out += _pull(gh, git, repo, pr, heads)
    return out


def _nodes(gh: GH, repo: str, pr: int, query: str, field: str) -> list[dict[str, Any]]:
    owner, name = repo.split("/")
    out = gh(
        "api",
        "graphql",
        "--paginate",
        "-f",
        f"query={query}",
        "-f",
        f"o={owner}",
        "-f",
        f"r={name}",
        "-F",
        f"p={pr}",
        "--jq",
        f".data.repository.pullRequest.{field}.nodes[] | tojson",
    )
    return [json.loads(line) for line in out.splitlines() if line]


def _later(git: Git, ref: str, commit: str) -> list[tuple[datetime, set[str]]]:
    """Returns when each commit after one on the first-parent line of a ref was made,
    and the files it changed. A merge changes no files."""
    out = git(
        *("log", "--first-parent", "--diff-merges=off", "--name-only"),
        *("--format=%x00%cI", ref, f"^{commit}"),
    )
    later: list[tuple[datetime, set[str]]] = []
    for chunk in out.split("\0")[1:]:
        date, *files = chunk.splitlines()
        later.append((datetime.fromisoformat(date), {f for f in files if f}))
    return later


def _pull(
    gh: GH, git: Git, repo: str, pr: int, heads: dict[str, dict[str, Any]]
) -> list[Pair]:
    ref = f"refs/review/{pr}"
    line = set(git("rev-list", "--first-parent", ref).split())
    later: dict[str, list[tuple[datetime, set[str]]]] = {}

    def judge(t: Thread | None) -> str:
        if t is None:
            return OPEN
        if t.commit not in line:
            return verdict(t, None)
        if t.commit not in later:
            later[t.commit] = _later(git, ref, t.commit)
        cutoff = t.last_reply
        touched = {
            f
            for at, files in later[t.commit]
            if cutoff is None or at <= cutoff
            for f in files
        }
        return verdict(t, touched)

    reviews = greptile_reviews(_nodes(gh, repo, pr, EDITS, "userContentEdits"))
    ts = threads(_nodes(gh, repo, pr, THREADS, "reviewThreads"))
    by_id = {t.id: t for t in ts}
    # Sedosaur opens a new thread for a finding only once its last one is resolved, so
    # each review's finding belongs to the newest thread for it as of that review.
    mine = sorted((t for t in ts if t.bot == SEDOSAUR), key=lambda t: t.id)
    ours = {(t.path, t.title): t for t in mine if t.commit not in heads}
    # Both reviewers list a finding again on each commit while its thread is open, so
    # each thread counts once, on the first paired commit that lists it.
    counted: set[int] = set()

    def repeat(t: Thread | None) -> bool:
        if t is None or t.id not in counted:
            if t is not None:
                counted.add(t.id)
            return False
        return True

    out: list[Pair] = []
    for head, rec in heads.items():
        ours |= {(t.path, t.title): t for t in mine if t.commit == head}
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
                    repeat=repeat(t),
                )
            )
        s: list[Item] = []
        for f in rec["review"]["findings"]:
            t = ours.get((f["path"], f["title"]))
            s.append(
                Item(
                    severity=f["severity"],
                    title=f["title"],
                    path=f["path"],
                    line=f["line"],
                    verdict=judge(t),
                    repeat=repeat(t),
                )
            )
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


def _verdicts(items: list[Item]) -> tuple[int, int, int]:
    return (
        sum(i.verdict == REAL for i in items),
        sum(i.verdict == NOISE for i in items),
        sum(i.verdict == OPEN for i in items),
    )


@dataclass(frozen=True, kw_only=True)
class _Tally:
    reviews: int
    greptile: tuple[int, int, int]
    """Real, noise, and open findings."""
    sedosaur: tuple[int, int, int]
    """Real, noise, and open findings."""
    found: int
    """Real Greptile findings that Sedosaur also found."""
    missed: list[tuple[Pair, Item]]
    only: list[tuple[Pair, Item]]
    """Real Sedosaur findings that Greptile did not find."""
    equal: int
    higher: int
    cost_usd: float
    duration_ms: int

    @classmethod
    def of(cls, ps: list[Pair]) -> "_Tally":
        return cls(
            reviews=len(ps),
            greptile=_verdicts([i for p in ps for i in p.greptile if not i.repeat]),
            sedosaur=_verdicts([i for p in ps for i in p.sedosaur if not i.repeat]),
            found=sum(
                p.greptile[i].verdict == REAL and not p.greptile[i].repeat
                for p in ps
                for i, _ in p.matches
            ),
            missed=[
                (p, x)
                for p in ps
                for i, x in enumerate(p.greptile)
                if x.verdict == REAL
                and not x.repeat
                and i not in {m for m, _ in p.matches}
            ],
            only=[
                (p, x)
                for p in ps
                for j, x in enumerate(p.sedosaur)
                if x.verdict == REAL
                and not x.repeat
                and j not in {m for _, m in p.matches}
            ],
            equal=sum(p.scores[0] == p.scores[1] for p in ps),
            higher=sum(p.scores[1] > p.scores[0] for p in ps),
            cost_usd=sum(p.cost_usd for p in ps),
            duration_ms=sum(p.duration_ms for p in ps),
        )

    @property
    def precision(self) -> str:
        real, noise, _ = self.sedosaur
        return _share(real, real + noise)

    @property
    def coverage(self) -> str:
        return _share(self.found, self.greptile[0])


def _limits(recs: list[dict[str, Any]]) -> list[str]:
    # Records from before reviews kept their limits, and reviews that Claude Code
    # reported no limits for, have none.
    newest = next((r for r in reversed(recs) if r["usage"].get("limits")), None)
    if newest is None:
        return ["No review has recorded the usage limits yet."]
    at = datetime.fromisoformat(newest["at"])
    return [
        f"As of the newest review, at {at:%Y-%m-%d %H:%M} UTC:",
        "",
        "| Window | Used | Status | Resets |",
        "| --- | --- | --- | --- |",
        *(
            f"| `{name}` | {round(100 * w['utilization'])}% | {w['status']} "
            f"| {datetime.fromtimestamp(w['resets_at'], UTC):%Y-%m-%d %H:%M} UTC |"
            for name, w in sorted(newest["usage"]["limits"].items())
        ),
    ]


def report(ps: list[Pair], recs: list[dict[str, Any]], *, since: datetime) -> str:
    """Renders the usage limits of the Claude subscription and the comparison of the
    paired reviews as Markdown.

    :param recs: The review records the pairs came from, oldest first.
    """
    lines = [
        f"## Sedosaur trial since {since:%Y-%m-%d %H:%M} UTC",
        "",
        "### Usage limits",
        "",
        *_limits(recs),
        "",
        "### Findings",
        "",
    ]
    if not ps:
        return "\n".join([*lines, "No commit has a review from both yet."])
    t = _Tally.of(ps)
    (gr, gn, go), (sr, sn, so) = t.greptile, t.sedosaur
    prs = _count(len({p.pr for p in ps}), "pull request")
    lines += [
        f"{_count(len(ps), 'commit')} on {prs} have a review from both.",
        "",
        "| | Greptile | Sedosaur |",
        "| --- | --- | --- |",
        f"| Findings | {gr + gn + go} | {sr + sn + so} |",
        f"| Real | {gr} | {sr} |",
        f"| Noise | {gn} | {sn} |",
        f"| Open | {go} | {so} |",
        f"| Precision | {_share(gr, gr + gn)} | {t.precision} |",
        "",
        f"- **Scores:** equal on {_share(t.equal, len(ps))}. Sedosaur was higher on "
        f"{t.higher} and lower on {len(ps) - t.equal - t.higher}.",
        f"- **Coverage:** Sedosaur found {t.coverage} real Greptile findings."
        if gr
        else "- **Coverage:** no real Greptile findings yet.",
        f"- **Only Sedosaur:** {_count(len(t.only), 'real finding')}.",
        f"- **Cost:** ${t.cost_usd:.2f} for {_count(len(ps), 'review')}, "
        f"{t.duration_ms / len(ps) / 1000:.0f} s each on average.",
        "",
        "### Real Greptile findings Sedosaur missed",
        "",
        *(
            [f"- {x.severity} **{x.title}** {_where(x)} (#{p.pr})" for p, x in t.missed]
            or ["None."]
        ),
        "",
        "### Real findings only Sedosaur found",
        "",
        *(
            [f"- {x.severity} **{x.title}** {_where(x)} (#{p.pr})" for p, x in t.only]
            or ["None."]
        ),
        "",
        "### By policy",
        "",
        "| Policy | Reviews | Precision | Coverage | Only Sedosaur | Equal scores "
        "| Cost per review |",
        "| --- | --- | --- | --- | --- | --- | --- |",
    ]
    by_policy: dict[str, list[Pair]] = defaultdict(list)
    for p in ps:
        by_policy[p.policy].append(p)
    for policy, group in by_policy.items():
        g = _Tally.of(group)
        lines.append(
            f"| `{policy[:10]}` | {g.reviews} | {g.precision} | {g.coverage} "
            f"| {len(g.only)} | {_share(g.equal, g.reviews)} "
            f"| ${g.cost_usd / g.reviews:.2f} |"
        )
    lines += [
        "",
        "### Commits",
        "",
        "| PR | Commit | Greptile | Sedosaur | Both | Only Greptile | Only Sedosaur |",
        "| --- | --- | --- | --- | --- | --- | --- |",
    ]
    for p in ps:
        matched_g = {i for i, _ in p.matches}
        matched_s = {j for _, j in p.matches}
        both = sum(
            not (p.greptile[i].repeat and p.sedosaur[j].repeat) for i, j in p.matches
        )
        only_g = sum(
            not x.repeat for i, x in enumerate(p.greptile) if i not in matched_g
        )
        only_s = sum(
            not x.repeat for j, x in enumerate(p.sedosaur) if j not in matched_s
        )
        lines.append(
            f"| #{p.pr} | `{p.head[:10]}` | {p.scores[0]}/5 | {p.scores[1]}/5 | {both} "
            f"| {only_g} | {only_s} |"
        )
    return "\n".join(lines)
