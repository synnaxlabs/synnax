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
from collections import defaultdict
from collections.abc import Callable, Iterator
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, replace
from datetime import UTC, datetime
from itertools import islice
from typing import Any

from review import render
from review.post import EDITS, GH

Git = Callable[..., str]
"""Runs `git` in a clone of the repository with the given arguments, and returns its
stdout."""

GREPTILE = "greptile-apps"
SEDOSAUR = "sedosaur"
SEDOSAUR_APP = 5632498
"""The ID of the Sedosaur GitHub App."""
BATCH = 50
"""How many commits one query for records reads."""
NEAR = 20
"""Findings on one file match when their lines are at most this far apart."""
WORKERS = 8
"""How many pull requests the comparison reads from GitHub at once."""
LIMIT_PRS = 100
"""The most pull requests that the search for the usage limits reads."""

REAL, NOISE, OPEN = "real", "noise", "open"

PULLS = """query($o: String!, $r: String!, $endCursor: String) {
  repository(owner: $o, name: $r) {
    pullRequests(
      first: 100, after: $endCursor, orderBy: {field: UPDATED_AT, direction: DESC}
    ) {
      pageInfo { hasNextPage endCursor }
      nodes {
        number updatedAt
        userContentEdits(first: 100) {
          pageInfo { hasNextPage }
          nodes { editor { login } }
        }
      }
    }
  }
}"""

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

RECORDS = """query($o: String!, $r: String!, $app: Int!) {
  repository(owner: $o, name: $r) { %s }
}
fragment Runs on Commit {
  checkSuites(first: 1, filterBy: {appId: $app}) {
    nodes {
      app { slug }
      checkRuns(last: 10, filterBy: {checkName: "Sedosaur Review"}) {
        nodes { detailsUrl startedAt text }
      }
    }
  }
}"""
"""Reads Sedosaur's check runs of each commit that an alias in place of `%s` names."""

_FIXED = re.compile(r"^fixed\b", re.I | re.M)
"""A reply line that says a finding is fixed, such as "Fixed in 1a2b3c"."""
_BLOCK = re.compile(r"<!-- greptile_comment -->(.*?)<!-- /greptile_comment -->", re.S)
_SUMMARY = re.compile(
    f"{re.escape(render.SUMMARY_START)}(.*?){re.escape(render.SUMMARY_END)}", re.S
)
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


Reviews = dict[str, tuple[int, list[tuple[str, str, int | None]]]]
"""Greptile's score and findings for each commit it reviewed."""


def greptile_reviews(edits: list[dict[str, Any]]) -> Reviews:
    """Returns Greptile's score and findings for each commit it reviewed.

    :param edits: The revisions of a pull request body, newest first.
    :returns: For each commit, the score and each finding's severity, title, and the
        ID of the comment that starts its thread, if it has one.
    """
    out: Reviews = {}
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


def prs(
    gh: GH, repo: str, since: datetime, bots: set[str]
) -> Iterator[tuple[int, datetime]]:
    """Yields each pull request updated since a time whose body every given bot edited,
    and when it was updated, newest first. A pull request with too many revisions to
    tell counts."""
    owner, name = repo.split("/")
    cursor: list[str] = []
    while True:
        page = json.loads(
            gh(
                *("api", "graphql", "-f", f"query={PULLS}"),
                *("-f", f"o={owner}", "-f", f"r={name}", *cursor),
            )
        )["data"]["repository"]["pullRequests"]
        for n in page["nodes"]:
            updated = datetime.fromisoformat(n["updatedAt"])
            if updated < since:
                return
            edits = n["userContentEdits"]
            editors = {(e["editor"] or {}).get("login") for e in edits["nodes"]}
            if edits["pageInfo"]["hasNextPage"] or bots <= editors:
                yield n["number"], updated
        if not page["pageInfo"]["hasNextPage"]:
            return
        cursor = ["-f", f"endCursor={page['pageInfo']['endCursor']}"]


def sedosaur_reviews(
    edits: list[dict[str, Any]], repo: str, since: datetime
) -> dict[str, int]:
    """Returns the run of Sedosaur's newest review since a time of each commit that its
    summary in the pull request body links. Only the Sedosaur App edits as the bot, so
    a fork cannot add a review.

    :param edits: The revisions of a pull request body, newest first.
    :returns: The run of each commit, oldest commit first.
    """
    out: dict[str, int] = {}
    for e in edits:
        editor = e["editor"] or {}
        if (
            editor.get("__typename") != "Bot"
            or editor.get("login") != SEDOSAUR
            or datetime.fromisoformat(e["editedAt"]) < since
        ):
            continue
        block = _SUMMARY.search(e["diff"] or "")
        if block is None:
            continue
        commits = re.findall(r"/commit/([0-9a-f]{40})", block[1])
        # The footer links the run last, after any run the summary cites.
        runs = re.findall(
            rf"github\.com/{re.escape(repo)}/actions/runs/(\d+)", block[1]
        )
        if commits and runs and commits[-1] not in out:
            out[commits[-1]] = int(runs[-1])
    return dict(reversed(out.items()))


def records(
    gh: GH, repo: str, reviews: list[tuple[str, int]]
) -> dict[tuple[str, int], dict[str, Any]]:
    """Returns the record of each Sedosaur review run of a commit that kept one, from
    the newest of the run's check runs that holds a record. A run from before check runs
    held records has none. Only the Sedosaur App writes its check runs, so a fork cannot
    add a record. It reads BATCH commits in each query.

    :param reviews: Each commit and the run that reviewed it.
    """
    owner, name = repo.split("/")
    commits = list(dict.fromkeys(c for c, _ in reviews))
    runs: dict[str, list[dict[str, Any]]] = {}
    for i in range(0, len(commits), BATCH):
        batch = commits[i : i + BATCH]
        objects = " ".join(
            f'c{j}: object(oid: "{c}") {{ ...Runs }}' for j, c in enumerate(batch)
        )
        out = gh(
            *("api", "graphql", "-f", f"query={RECORDS % objects}"),
            *("-f", f"o={owner}", "-f", f"r={name}", "-F", f"app={SEDOSAUR_APP}"),
        )
        found = json.loads(out)["data"]["repository"]
        for j, c in enumerate(batch):
            runs[c] = [
                r
                for s in found[f"c{j}"]["checkSuites"]["nodes"]
                if s["app"]["slug"] == SEDOSAUR
                for r in s["checkRuns"]["nodes"]
            ]
    recs: dict[tuple[str, int], dict[str, Any]] = {}
    for commit, run in reviews:
        url = f"https://github.com/{repo}/actions/runs/{run}"
        ours = sorted(
            (r for r in runs[commit] if r["detailsUrl"] == url),
            key=lambda r: r["startedAt"],
            reverse=True,
        )
        kept = (render.recorded(r["text"] or "") for r in ours)
        if (rec := next((k for k in kept if k is not None), None)) is not None:
            recs[(commit, run)] = rec
    return recs


def limits(gh: GH, repo: str, since: datetime) -> dict[str, Any] | None:
    """Returns the newest record since a time that holds the usage limits, or None when
    none does. It reads the pull requests that Sedosaur reviewed, newest first, until
    one was updated before that record, as a review edits the body after its record, or
    until it read LIMIT_PRS of them."""
    newest: dict[str, Any] | None = None
    for pr, updated in islice(prs(gh, repo, since, {SEDOSAUR}), LIMIT_PRS):
        if newest is not None and updated < datetime.fromisoformat(newest["at"]):
            break
        edits = _nodes(gh, repo, pr, EDITS, "userContentEdits")
        reviews = list(reversed(sedosaur_reviews(edits, repo, since).items()))
        recs = records(gh, repo, reviews)
        # Records from before reviews kept their limits, and reviews that Claude Code
        # reported no limits for, have none.
        found = [
            r
            for k in reviews
            if (r := recs.get(k)) is not None and r["usage"].get("limits")
        ]
        if found:
            newest = max(
                [r for r in (newest, found[0]) if r is not None],
                key=lambda r: datetime.fromisoformat(r["at"]),
            )
    return newest


def git(*args: str) -> str:
    return subprocess.run(
        ["git", *args], capture_output=True, text=True, check=True
    ).stdout


@dataclass(frozen=True, kw_only=True)
class _Read:
    """What the comparison reads from GitHub about one pull request."""

    pr: int
    heads: list[str]
    """The commits Sedosaur reviewed, oldest first."""
    reviews: Reviews
    recs: dict[str, dict[str, Any]]
    """The record of each commit both reviewed in the window."""
    threads: list[Thread]


def pairs(gh: GH, git: Git, repo: str, since: datetime) -> list[Pair]:
    """Pairs each Sedosaur review since a time with Greptile's review of the same
    commit. It reads WORKERS pull requests from GitHub at a time.

    :param git: Runs in a clone with full history. The pairing fetches the head of each
        paired pull request into it as `refs/review/<number>`.
    :returns: A pair for each commit that both reviewers reviewed.
    """
    both = [pr for pr, _ in prs(gh, repo, since, {GREPTILE, SEDOSAUR})]
    with ThreadPoolExecutor(WORKERS) as pool:
        edits = list(
            pool.map(lambda pr: _nodes(gh, repo, pr, EDITS, "userContentEdits"), both)
        )
    runs = [sedosaur_reviews(e, repo, since) for e in edits]
    reviews = [greptile_reviews(e) for e in edits]
    recs = records(
        gh,
        repo,
        [
            (c, r)
            for rs, rv in zip(runs, reviews, strict=True)
            for c, r in rs.items()
            if c in rv
        ],
    )
    read: list[_Read] = []
    for pr, rs, rv in zip(both, runs, reviews, strict=True):
        kept = {
            c: rec
            for c, r in rs.items()
            if (rec := recs.get((c, r))) is not None
            and datetime.fromisoformat(rec["at"]) >= since
        }
        if kept:
            read.append(_Read(pr=pr, heads=list(rs), reviews=rv, recs=kept, threads=[]))
    with ThreadPoolExecutor(WORKERS) as pool:
        nodes = pool.map(
            lambda r: _nodes(gh, repo, r.pr, THREADS, "reviewThreads"), read
        )
        read = [
            replace(r, threads=threads(n)) for r, n in zip(read, nodes, strict=True)
        ]
    if not read:
        return []
    read.sort(key=lambda r: r.pr)
    git(
        *("fetch", "--no-tags", "origin"),
        *(f"+refs/pull/{r.pr}/head:refs/review/{r.pr}" for r in read),
    )
    return [p for r in read for p in _pull(git, r)]


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


def _pull(git: Git, r: _Read) -> list[Pair]:
    pr, heads, reviews, recs = r.pr, r.heads, r.reviews, r.recs
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

    by_id = {t.id: t for t in r.threads}
    # Sedosaur opens a new thread for a finding only once its last one is resolved, so
    # each review's finding belongs to the newest thread for it as of that review.
    mine = sorted((t for t in r.threads if t.bot == SEDOSAUR), key=lambda t: t.id)
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
    for head in heads:
        ours |= {(t.path, t.title): t for t in mine if t.commit == head}
        if head not in recs:
            continue
        rec = recs[head]
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


def _limits(newest: dict[str, Any] | None) -> list[str]:
    if newest is None:
        return ["No recent review recorded the usage limits."]
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


def report(ps: list[Pair], newest: dict[str, Any] | None, *, since: datetime) -> str:
    """Renders the usage limits of the Claude subscription and the comparison of the
    paired reviews as Markdown.

    :param newest: The newest record that holds the usage limits, if any.
    """
    lines = [
        f"## Sedosaur trial since {since:%Y-%m-%d %H:%M} UTC",
        "",
        "### Usage limits",
        "",
        *_limits(newest),
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
