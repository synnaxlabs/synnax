#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import re
import subprocess
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import pytest

from review import compare, render

REPO = "o/r"
HEAD = "a" * 40
SINCE = datetime(2026, 11, 5, tzinfo=UTC)
MORNING = "2026-11-05T10:00:00Z"
NOON = "2026-11-05T12:00:00Z"


def run(cwd: Path, *args: str, date: str = MORNING) -> str:
    env = {**os.environ, "GIT_AUTHOR_DATE": date, "GIT_COMMITTER_DATE": date}
    return subprocess.run(
        ["git", "-c", "user.name=t", "-c", "user.email=t@t", *args],
        cwd=cwd,
        env=env,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()


class Repo:
    """A remote that holds pull request 7, and a clone the comparison runs git in.

    The pull request changes the first line of `a.go` in its head commit.
    """

    def __init__(self, root: Path) -> None:
        self.origin = root / "origin"
        self.origin.mkdir()
        run(self.origin, "init", "-q", "-b", "main")
        (self.origin / "a.go").write_text("".join(f"{n}\n" for n in range(1, 9)))
        (self.origin / "c.go").write_text("1\n")
        run(self.origin, "add", ".")
        run(self.origin, "commit", "-qm", "base")
        run(self.origin, "branch", "pr")
        run(root, "clone", "-q", str(self.origin), "clone")
        self.clone = root / "clone"
        run(self.origin, "checkout", "-q", "pr")
        self.head = self.commit(
            {"a.go": "x\n" + "".join(f"{n}\n" for n in range(2, 9))}
        )

    def commit(self, files: dict[str, str], date: str = MORNING) -> str:
        """Commits files to the branch the remote has checked out."""
        for name, text in files.items():
            (self.origin / name).write_text(text)
        run(self.origin, "add", ".")
        run(self.origin, "commit", "-qm", "c", date=date)
        run(self.origin, "update-ref", "refs/pull/7/head", "refs/heads/pr")
        return run(self.origin, "rev-parse", "HEAD")

    def merge_main(self, files: dict[str, str]) -> None:
        """Commits files to main, then merges main into the pull request."""
        run(self.origin, "checkout", "-q", "main")
        self.commit(files)
        run(self.origin, "checkout", "-q", "pr")
        run(self.origin, "merge", "-q", "--no-edit", "main")
        run(self.origin, "update-ref", "refs/pull/7/head", "refs/heads/pr")

    def git(self, *args: str) -> str:
        return run(self.clone, *args)


@pytest.fixture
def repo(tmp_path: Path) -> Repo:
    return Repo(tmp_path)


def greptile_block(score: int, commit: str = HEAD) -> str:
    """A pull request body the way Greptile leaves it after it reviews a commit."""
    return (
        "Fixes A.\n\n<!-- greptile_comment -->\n"
        f"<h2>Confidence Score: {score}/5</h2>\n\n<h2>Findings</h2>\n\n"
        '1. <img alt="P1" src="p1.svg" align="top">&nbsp;**Leaks a handle** '
        '<a href="https://github.com/o/r/pull/7#discussion_r101">▶</a>\n'
        '2. <img alt="P2" src="p2.svg" align="top"> <img alt="security" src="s.svg">'
        "&nbsp;**Reads a secret** "
        '<a href="https://github.com/o/r/pull/7#issuecomment-5">▶</a>\n\n'
        f"<!-- greptile_confidence_score:{score} -->\n"
        f'<sub>Last reviewed commit: ["Fix"](https://github.com/o/r/commit/{commit})'
        "</sub>\n<!-- /greptile_comment -->"
    )


def edit(body: str, login: str = "greptile-apps", kind: str = "Bot") -> dict[str, Any]:
    return {"editor": {"__typename": kind, "login": login}, "diff": body}


def thread(
    id: int,
    login: str,
    title: str,
    line: int,
    commit: str,
    *,
    resolved: bool = False,
    up: int = 0,
    down: int = 0,
    replied: str | None = None,
    reply: str = "Thanks.",
) -> dict[str, Any]:
    badge = '<img alt="P1" src="p1.svg">'
    opened = "2026-11-05T09:00:00Z"
    last = [{"author": {"login": login}, "body": "", "createdAt": opened}]
    if replied is not None:
        last += [
            {"author": {"login": "pat"}, "body": reply, "createdAt": replied},
            {
                "author": {"login": "greptile-apps"},
                "body": "Fixed.",
                "createdAt": "2026-11-05T23:00:00Z",
            },
        ]
    return {
        "isResolved": resolved,
        "path": "a.go",
        "originalLine": line,
        "first": {
            "nodes": [
                {
                    "databaseId": id,
                    "author": {"login": login},
                    "body": f"{badge} **{title}**\n\nWhy it matters.",
                    "originalCommit": {"oid": commit},
                    "reactionGroups": [
                        {"content": "THUMBS_UP", "reactors": {"totalCount": up}},
                        {"content": "THUMBS_DOWN", "reactors": {"totalCount": down}},
                    ],
                }
            ]
        },
        "last": {"nodes": last},
    }


def record(head: str, score: int = 5) -> dict[str, Any]:
    return {
        "pr": 7,
        "head": head,
        "base": "b" * 40,
        "at": "2026-11-06T10:00:00+00:00",
        "policy": "c" * 40,
        "usage": {"cost_usd": 0.5, "duration_ms": 60_000},
        "review": {
            "score": score,
            "findings": [
                {
                    "severity": "P1",
                    "title": "Leaks a handle too",
                    "path": "a.go",
                    "line": 12,
                },
                {"severity": "P2", "title": "Odd name", "path": "b.go", "line": 3},
            ],
        },
    }


def limited(utilization: float, at: str, head: str = HEAD) -> dict[str, Any]:
    """A record of a review that kept the usage limits of the subscription."""
    r = record(head)
    r["at"] = at
    week = {
        "status": "allowed_warning",
        "utilization": utilization,
        "resets_at": 1_794_247_200,
    }
    r["usage"]["limits"] = {"seven_day": week}
    return r


def summary(
    head: str, run: int, *, repo: str = REPO, edited: str = NOON, cites: str = ""
) -> dict[str, Any]:
    """A revision of the pull request body by Sedosaur after it reviews a commit."""
    e = edit(
        f"Fixes A.\n\n<!-- sedosaur_comment -->\n- {cites}\n"
        f"<sub>Reviews (1) · Last reviewed commit: [Fix](https://github.com/o/r/commit/"
        f"{head}) · [Run](https://github.com/{repo}/actions/runs/{run})</sub>\n"
        "<!-- /sedosaur_comment -->",
        "sedosaur",
    )
    return e | {"editedAt": edited}


def check_run(
    run: int,
    rec: dict[str, Any] | None,
    *,
    slug: str = "sedosaur",
    started: str = NOON,
    status: str = "completed",
) -> dict[str, Any]:
    """A check run of a Sedosaur review run, with its record if it kept one."""
    text = render.RULES if rec is None else f"{render.RULES}\n\n{render.record(rec)}"
    return {
        "app": {"slug": slug},
        "details_url": f"https://github.com/{REPO}/actions/runs/{run}",
        "started_at": started,
        "status": status,
        "output": {"text": text},
    }


def pull(
    number: int, updated: str, edits: list[dict[str, Any]], *, more: bool = False
) -> dict[str, Any]:
    return {
        "number": number,
        "updatedAt": updated,
        "userContentEdits": {
            "pageInfo": {"hasNextPage": more},
            "nodes": [{"editor": e["editor"]} for e in edits],
        },
    }


class FakeGH:
    """Serves pull request 7 and its records through the gh calls the comparison makes.

    Greptile reviewed the head and resolved its thread. Sedosaur's thread is open.
    """

    def __init__(self, head: str = HEAD) -> None:
        self.edits = [edit(greptile_block(4, head))]
        self.threads = [
            thread(101, "greptile-apps", "Leaks a handle", 10, head, resolved=True),
            thread(201, "sedosaur", "Leaks a handle too", 12, head),
        ]
        self.pulls: list[dict[str, Any]] = []
        self.pages: list[list[dict[str, Any]]] = []
        self.cursors: list[str | None] = []
        self.bodies: dict[int, list[dict[str, Any]]] = {}
        """The revisions of the body of each pull request but 7, whose are edits."""
        self.checks: dict[str, list[dict[str, Any]]] = {}
        """The check runs of each commit."""
        self.reads: list[str] = []
        """The commits whose check runs the comparison read."""

    def review(
        self, run: int, rec: dict[str, Any], *, pr: int = 7, kept: bool = True
    ) -> None:
        """Adds a Sedosaur review run of a pull request, newer than every other, and
        its check run, which holds the record when the run kept it."""
        self.body(pr).insert(0, summary(rec["head"], run))
        held = rec if kept else None
        self.checks.setdefault(rec["head"], []).append(check_run(run, held))

    def body(self, pr: int) -> list[dict[str, Any]]:
        return self.edits if pr == 7 else self.bodies.setdefault(pr, [])

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        if args[:3] == ("api", "graphql", "--paginate"):
            query = args[args.index("-f") + 1]
            assert "$endCursor" in query
            pr = int(next(a[2:] for a in args if a.startswith("p=")))
            nodes = self.body(pr) if "userContentEdits" in query else self.threads
            return "".join(f"{json.dumps(n)}\n" for n in nodes)
        if args[:2] == ("api", "graphql"):
            cursor = next((a[10:] for a in args if a.startswith("endCursor=")), None)
            self.cursors.append(cursor)
            pages = self.pages or [self.pulls or [pull(7, NOON, self.edits)]]
            i = int(cursor or 0)
            page = {
                "pageInfo": {
                    "hasNextPage": i + 1 < len(pages),
                    "endCursor": str(i + 1),
                },
                "nodes": pages[i],
            }
            return json.dumps({"data": {"repository": {"pullRequests": page}}})
        if m := re.fullmatch(
            rf"repos/{REPO}/commits/(\w+)/check-runs"
            r"\?check_name=Sedosaur%20Review&filter=all&per_page=100",
            args[1],
        ):
            self.reads.append(m[1])
            return "".join(f"{json.dumps(r)}\n" for r in self.checks.get(m[1], []))
        raise AssertionError(f"unexpected gh call: {args}")


def pairs(gh: FakeGH, repo: Repo, recs: list[dict[str, Any]]) -> list[compare.Pair]:
    """Pairs records that runs 1, 2, and so on kept, oldest first."""
    for run, rec in enumerate(recs, 1):
        gh.review(run, rec)
    return compare.pairs(gh, repo.git, REPO, SINCE, cache(repo))


def cache(repo: Repo) -> Path:
    path = repo.clone.parent / "records"
    path.mkdir(exist_ok=True)
    return path


def t(**kw: Any) -> compare.Thread:
    fields: dict[str, Any] = {
        "id": 1,
        "bot": "sedosaur",
        "path": "a.go",
        "line": 1,
        "commit": HEAD,
        "title": "T",
        "resolved": False,
        "up": 0,
        "down": 0,
        "last_reply": None,
        "fixed": False,
    }
    return compare.Thread(**(fields | kw))


def item(path: str | None, line: int | None) -> compare.Item:
    return compare.Item(
        severity="P2", title="T", path=path, line=line, verdict=compare.OPEN
    )


class TestGreptileReviews:
    """Tests reading Greptile's reviews from the revisions of a pull request body."""

    def test_should_read_the_score_and_findings_of_a_commit(self) -> None:
        assert compare.greptile_reviews([edit(greptile_block(4))]) == {
            HEAD: (4, [("P1", "Leaks a handle", 101), ("P2", "Reads a secret", None)])
        }

    def test_should_keep_the_newest_review_of_a_commit(self) -> None:
        edits = [edit(greptile_block(5)), edit(greptile_block(3))]
        assert compare.greptile_reviews(edits)[HEAD][0] == 5

    def test_should_ignore_a_revision_by_a_person(self) -> None:
        edits = [edit(greptile_block(5), kind="User"), edit(greptile_block(5), "pat")]
        assert compare.greptile_reviews(edits) == {}

    def test_should_ignore_a_deleted_revision(self) -> None:
        assert compare.greptile_reviews([{"editor": None, "diff": None}]) == {}


class TestThreads:
    """Tests reading the review threads of the reviewers."""

    def test_should_skip_a_thread_with_no_comments(self) -> None:
        node = thread(101, "greptile-apps", "Leaks a handle", 10, HEAD)
        node["first"]["nodes"] = []
        assert compare.threads([node]) == []

    def test_should_keep_a_thread_whose_commit_is_gone(self) -> None:
        node = thread(101, "greptile-apps", "Leaks a handle", 10, HEAD)
        node["first"]["nodes"][0]["originalCommit"] = None
        assert compare.threads([node])[0].commit is None

    def test_should_read_a_fix_from_a_reply_of_a_person(self) -> None:
        reply = (
            "*Emiliano + Claude: generated reply.*\n\nFixed in 1a2b3c, in a_test.go."
        )
        node = thread(
            101,
            "sedosaur",
            "Leaks a handle",
            10,
            HEAD,
            replied="2026-11-05T11:00:00Z",
            reply=reply,
        )
        assert compare.threads([node])[0].fixed

    def test_should_not_read_a_fix_from_a_reply_of_a_reviewer(self) -> None:
        node = thread(
            101, "sedosaur", "Leaks a handle", 10, HEAD, replied="2026-11-05T11:00:00Z"
        )
        assert not compare.threads([node])[0].fixed


class TestVerdict:
    """Tests deciding whether a finding was real."""

    def test_should_take_more_thumbs_down_as_noise(self) -> None:
        assert (
            compare.verdict(t(up=1, down=2, resolved=True), {"a.go"}) == compare.NOISE
        )

    def test_should_take_more_thumbs_up_as_real(self) -> None:
        assert compare.verdict(t(up=1), set()) == compare.REAL

    def test_should_keep_an_unresolved_thread_open(self) -> None:
        assert compare.verdict(t(), {"a.go"}) == compare.OPEN

    def test_should_take_a_fix_as_real(self) -> None:
        assert compare.verdict(t(resolved=True), {"a.go"}) == compare.REAL

    def test_should_take_a_reply_that_says_fixed_as_real(self) -> None:
        assert compare.verdict(t(fixed=True), None) == compare.REAL

    def test_should_take_a_resolve_without_a_fix_as_noise(self) -> None:
        assert compare.verdict(t(resolved=True), {"b.go"}) == compare.NOISE

    def test_should_keep_a_thread_open_when_its_commit_left_the_history(self) -> None:
        assert compare.verdict(t(resolved=True), None) == compare.OPEN


class TestMatch:
    """Tests matching the findings of two reviews."""

    def test_should_pair_the_closest_findings_once(self) -> None:
        a = [item("a.go", 10), item("a.go", 20)]
        b = [item("a.go", 19), item("a.go", 11)]
        assert sorted(compare.match(a, b)) == [(0, 1), (1, 0)]

    def test_should_pair_findings_up_to_20_lines_apart(self) -> None:
        assert compare.match([item("a.go", 10)], [item("a.go", 30)]) == [(0, 0)]

    def test_should_not_pair_other_files_or_far_lines(self) -> None:
        a = [item("a.go", 10), item("a.go", 30), item(None, None)]
        b = [item("b.go", 10), item("a.go", 51), item(None, None)]
        assert compare.match(a, b) == []


class TestPrs:
    """Tests finding the pull requests that bots edited."""

    def test_should_list_pull_requests_both_edited_until_one_is_too_old(
        self,
    ) -> None:
        gh = FakeGH()
        both = [edit("", "greptile-apps"), edit("", "sedosaur")]
        gh.pulls = [
            pull(9, NOON, both),
            pull(8, MORNING, both[:1]),
            pull(7, MORNING, [], more=True),
            pull(6, "2026-11-04T10:00:00Z", both),
        ]
        found = compare.prs(gh, REPO, SINCE, {"greptile-apps", "sedosaur"})
        noon, morning = (datetime.fromisoformat(t) for t in (NOON, MORNING))
        assert list(found) == [(9, noon), (7, morning)]

    def test_should_list_pull_requests_one_bot_edited(self) -> None:
        gh = FakeGH()
        gh.pulls = [pull(9, NOON, [edit("", "greptile-apps")]), pull(8, NOON, [])]
        found = compare.prs(gh, REPO, SINCE, {"greptile-apps"})
        assert [pr for pr, _ in found] == [9]

    def test_should_read_the_next_page_until_one_is_too_old(self) -> None:
        gh = FakeGH()
        both = [edit("", "greptile-apps"), edit("", "sedosaur")]
        gh.pages = [
            [pull(9, NOON, both)],
            [pull(8, NOON, both), pull(7, "2026-11-04T10:00:00Z", both)],
            [pull(6, NOON, both)],
        ]
        found = compare.prs(gh, REPO, SINCE, {"greptile-apps", "sedosaur"})
        assert [pr for pr, _ in found] == [9, 8]
        assert gh.cursors == [None, "1"]


class TestSedosaurReviews:
    """Tests reading Sedosaur's reviews from the revisions of a pull request body."""

    def test_should_read_the_newest_run_of_each_commit_oldest_first(self) -> None:
        a, b = "a" * 40, "b" * 40
        edits = [summary(a, 3), summary(b, 2), summary(a, 1)]
        runs = compare.sedosaur_reviews(edits, REPO, SINCE)
        assert list(runs.items()) == [(b, 2), (a, 3)]

    def test_should_take_the_run_of_the_footer(self) -> None:
        cited = f"Fixes [a run](https://github.com/{REPO}/actions/runs/9)."
        edits = [summary(HEAD, 2, cites=cited)]
        assert compare.sedosaur_reviews(edits, REPO, SINCE) == {HEAD: 2}

    def test_should_skip_a_review_from_before_a_time(self) -> None:
        edits = [summary(HEAD, 2, edited="2026-11-04T10:00:00Z")]
        assert compare.sedosaur_reviews(edits, REPO, SINCE) == {}

    def test_should_ignore_a_revision_by_a_person_or_another_repository(
        self,
    ) -> None:
        person = summary(HEAD, 1)
        person["editor"]["__typename"] = "User"
        edits = [person, summary(HEAD, 2, repo="fork/r")]
        assert compare.sedosaur_reviews(edits, REPO, SINCE) == {}


class TestRecord:
    """Tests reading the record of a Sedosaur review from its check run."""

    def test_should_read_the_record_of_the_run(self, tmp_path: Path) -> None:
        gh = FakeGH()
        gh.checks[HEAD] = [
            check_run(1, record(HEAD, score=2)),
            check_run(2, record(HEAD)),
            check_run(2, record(HEAD, score=3), slug="other"),
        ]
        assert compare.record(gh, REPO, HEAD, 2, tmp_path) == record(HEAD)

    def test_should_take_the_newest_check_run_that_holds_a_record(
        self, tmp_path: Path
    ) -> None:
        gh = FakeGH()
        gh.checks[HEAD] = [
            check_run(1, None, started="2026-11-05T14:00:00Z"),
            check_run(1, record(HEAD), started="2026-11-05T13:00:00Z"),
            check_run(1, record(HEAD, score=2)),
        ]
        assert compare.record(gh, REPO, HEAD, 1, tmp_path) == record(HEAD)

    def test_should_return_none_for_a_check_run_with_no_record(
        self, tmp_path: Path
    ) -> None:
        gh = FakeGH()
        gh.checks[HEAD] = [check_run(1, None)]
        assert compare.record(gh, REPO, HEAD, 1, tmp_path) is None

    def test_should_return_none_for_a_record_of_another_app(
        self, tmp_path: Path
    ) -> None:
        gh = FakeGH()
        gh.checks[HEAD] = [check_run(1, record(HEAD), slug="other")]
        assert compare.record(gh, REPO, HEAD, 1, tmp_path) is None

    def test_should_read_a_record_once(self, tmp_path: Path) -> None:
        gh = FakeGH()
        gh.checks[HEAD] = [check_run(1, record(HEAD))]
        assert compare.record(gh, REPO, HEAD, 1, tmp_path) == record(HEAD)
        assert compare.record(gh, REPO, HEAD, 1, tmp_path) == record(HEAD)
        assert gh.reads == [HEAD]

    def test_should_read_once_that_a_run_kept_no_record(self, tmp_path: Path) -> None:
        gh = FakeGH()
        gh.checks[HEAD] = [check_run(1, None)]
        assert compare.record(gh, REPO, HEAD, 1, tmp_path) is None
        assert compare.record(gh, REPO, HEAD, 1, tmp_path) is None
        assert gh.reads == [HEAD]

    def test_should_read_again_while_a_check_run_is_in_progress(
        self, tmp_path: Path
    ) -> None:
        gh = FakeGH()
        gh.checks[HEAD] = [check_run(1, None, status="in_progress")]
        assert compare.record(gh, REPO, HEAD, 1, tmp_path) is None
        gh.checks[HEAD] = [check_run(1, record(HEAD))]
        assert compare.record(gh, REPO, HEAD, 1, tmp_path) == record(HEAD)
        assert gh.reads == [HEAD, HEAD]


class TestLimits:
    """Tests finding the newest record of the usage limits."""

    def test_should_take_the_newest_record_with_limits(self, tmp_path: Path) -> None:
        gh = FakeGH()
        a, b, c, d = "a" * 40, "b" * 40, "c" * 40, "d" * 40
        empty = limited(0.5, "2026-11-05T11:30:00+00:00", c)
        empty["usage"]["limits"] = {}
        gh.review(1, limited(0.81, "2026-11-05T11:00:00+00:00", b), pr=9)
        gh.review(2, empty, pr=9)
        gh.review(3, limited(0.2, "2026-11-05T09:00:00+00:00", a), pr=8)
        gh.review(4, limited(0.9, "2026-11-05T08:00:00+00:00", d), pr=6)
        bot = [edit("", "sedosaur")]
        gh.pulls = [
            pull(9, NOON, bot),
            pull(8, NOON, bot),
            pull(7, NOON, []),
            pull(6, "2026-11-05T10:00:00Z", bot),
        ]
        newest = compare.limits(gh, REPO, SINCE, tmp_path)
        assert newest is not None
        assert newest["usage"]["limits"]["seven_day"]["utilization"] == 0.81
        assert gh.reads == [c, b, a]

    def test_should_skip_a_record_of_another_app_or_from_before_a_time(
        self, tmp_path: Path
    ) -> None:
        gh = FakeGH()
        gh.review(1, limited(0.81, NOON), pr=9)
        gh.checks[HEAD][0]["app"]["slug"] = "other"
        gh.review(2, limited(0.81, NOON, "a" * 40), pr=8)
        gh.bodies[8][0]["editedAt"] = "2026-11-04T10:00:00Z"
        bot = [edit("", "sedosaur")]
        gh.pulls = [pull(9, NOON, bot), pull(8, NOON, bot)]
        assert compare.limits(gh, REPO, SINCE, tmp_path) is None

    def test_should_stop_after_reading_the_most_pull_requests(
        self, monkeypatch: pytest.MonkeyPatch, tmp_path: Path
    ) -> None:
        monkeypatch.setattr(compare, "LIMIT_PRS", 1)
        gh = FakeGH()
        gh.review(1, limited(0.5, NOON, "a" * 40), pr=9)
        gh.review(2, limited(0.81, NOON), pr=8)
        gh.checks["a" * 40][0]["output"]["text"] = render.RULES
        bot = [edit("", "sedosaur")]
        gh.pulls = [pull(9, NOON, bot), pull(8, NOON, bot)]
        assert compare.limits(gh, REPO, SINCE, tmp_path) is None
        assert gh.reads == ["a" * 40]


class TestPairs:
    """Tests pairing Sedosaur's records with Greptile's reviews."""

    def test_should_pair_a_commit_both_reviewed(self, repo: Repo) -> None:
        repo.commit({"a.go": "y\n"})
        gh = FakeGH(repo.head)
        [p] = pairs(gh, repo, [record(repo.head), record("d" * 40)])
        assert (p.pr, p.head, p.scores) == (7, repo.head, (4, 5))
        assert [(i.title, i.path, i.line, i.verdict) for i in p.greptile] == [
            ("Leaks a handle", "a.go", 10, compare.REAL),
            ("Reads a secret", None, None, compare.OPEN),
        ]
        assert [(i.title, i.verdict) for i in p.sedosaur] == [
            ("Leaks a handle too", compare.REAL),
            ("Odd name", compare.OPEN),
        ]
        assert p.matches == [(0, 0)]

    def test_should_read_only_the_records_of_paired_commits(self, repo: Repo) -> None:
        gh = FakeGH(repo.head)
        pairs(gh, repo, [record(repo.head), record("d" * 40)])
        assert gh.reads == [repo.head]

    def test_should_count_the_newest_record_of_a_commit(self, repo: Repo) -> None:
        gh = FakeGH(repo.head)
        recs = [record(repo.head, score=2), record(repo.head, score=4)]
        [p] = pairs(gh, repo, recs)
        assert p.scores == (4, 4)

    def test_should_skip_a_commit_whose_run_kept_no_record(self, repo: Repo) -> None:
        gh = FakeGH(repo.head)
        gh.review(1, record(repo.head), kept=False)
        assert compare.pairs(gh, repo.git, REPO, SINCE, cache(repo)) == []

    def test_should_skip_a_record_from_before_the_window(self, repo: Repo) -> None:
        old = record(repo.head)
        old["at"] = "2026-11-04T10:00:00+00:00"
        assert pairs(FakeGH(repo.head), repo, [old]) == []

    def test_should_skip_a_pull_request_one_reviewer_missed(self, repo: Repo) -> None:
        gh = FakeGH(repo.head)
        gh.pulls = [pull(7, NOON, [edit("", "sedosaur")])]
        assert pairs(gh, repo, [record(repo.head)]) == []
        assert gh.reads == []

    def test_should_take_a_resolve_without_a_fix_as_noise(self, repo: Repo) -> None:
        repo.commit({"c.go": "2\n"})
        [p] = pairs(FakeGH(repo.head), repo, [record(repo.head)])
        assert p.greptile[0].verdict == compare.NOISE
        assert p.sedosaur[0].verdict == compare.NOISE

    def test_should_not_count_a_merge_from_main_as_a_fix(self, repo: Repo) -> None:
        repo.merge_main({"a.go": "".join(f"{n}\n" for n in range(1, 8)) + "z\n"})
        [p] = pairs(FakeGH(repo.head), repo, [record(repo.head)])
        assert p.greptile[0].verdict == compare.NOISE

    def test_should_not_count_a_commit_after_the_last_reply(self, repo: Repo) -> None:
        repo.commit({"a.go": "y\n"}, date=NOON)
        gh = FakeGH(repo.head)
        gh.threads[0] = thread(
            101,
            "greptile-apps",
            "Leaks a handle",
            10,
            repo.head,
            resolved=True,
            replied="2026-11-05T11:00:00Z",
        )
        [p] = pairs(gh, repo, [record(repo.head)])
        assert p.greptile[0].verdict == compare.NOISE

    def test_should_keep_open_a_thread_on_a_commit_outside_the_history(
        self, repo: Repo
    ) -> None:
        gone = "e" * 40
        gh = FakeGH(gone)
        gh.threads[0]["isResolved"] = True
        [p] = pairs(gh, repo, [record(gone)])
        assert p.greptile[0].verdict == compare.OPEN

    def test_should_mark_a_thread_listed_again_as_a_repeat(self, repo: Repo) -> None:
        first = repo.head
        later = repo.commit({"c.go": "2\n"})
        gh = FakeGH(first)
        gh.edits = [edit(greptile_block(4, later)), edit(greptile_block(4, first))]
        one, two = pairs(gh, repo, [record(first), record(later)])
        assert [i.repeat for i in one.greptile + one.sedosaur] == [False] * 4
        assert [i.repeat for i in two.greptile] == [True, False]
        assert [i.repeat for i in two.sedosaur] == [True, False]
        text = compare.report([one, two], None, since=SINCE)
        assert "| Findings | 3 | 3 |" in text

    def test_should_match_a_new_finding_with_a_repeat(self, repo: Repo) -> None:
        first = repo.head
        later = repo.commit({"c.go": "2\n"})
        gh = FakeGH(first)
        gh.threads[0] = thread(101, "greptile-apps", "Leaks a handle", 10, later)
        only_secret = greptile_block(5, first).replace("**Leaks a handle**", "")
        gh.edits = [edit(greptile_block(4, later)), edit(only_secret)]
        _, two = pairs(gh, repo, [record(first), record(later)])
        assert (two.greptile[0].title, two.greptile[0].repeat) == (
            "Leaks a handle",
            False,
        )
        assert two.sedosaur[0].repeat
        assert (0, 0) in two.matches

    def test_should_count_a_finding_on_the_first_paired_commit_that_lists_it(
        self, repo: Repo
    ) -> None:
        later = repo.commit({"c.go": "2\n"})
        gh = FakeGH("e" * 40)
        gh.edits = [edit(greptile_block(4, later))]
        [p] = pairs(gh, repo, [record(later)])
        assert [i.title for i in p.greptile] == ["Leaks a handle", "Reads a secret"]
        assert [i.title for i in p.sedosaur] == ["Leaks a handle too", "Odd name"]

    def test_should_judge_each_review_by_its_own_thread(self, repo: Repo) -> None:
        later = repo.commit({"c.go": "2\n"})
        gh = FakeGH(repo.head)
        gh.edits = [edit(greptile_block(3, later)), edit(greptile_block(4, repo.head))]
        gh.threads[1:] = [
            thread(201, "sedosaur", "Leaks a handle too", 12, repo.head, up=1),
            thread(202, "sedosaur", "Leaks a handle too", 12, later, down=1),
        ]
        first, second = pairs(gh, repo, [record(repo.head), record(later)])
        assert first.sedosaur[0].verdict == compare.REAL
        assert second.sedosaur[0].verdict == compare.NOISE


class TestReport:
    """Tests the Markdown report of the comparison."""

    def test_should_report_coverage_and_precision(self, repo: Repo) -> None:
        repo.commit({"a.go": "y\n"})
        ps = pairs(FakeGH(repo.head), repo, [record(repo.head)])
        text = compare.report(ps, None, since=SINCE)
        short = repo.head[:10]
        assert "1 commit on 1 pull request have a review from both." in text
        assert "| Precision | 1 of 1 (100%) | 1 of 1 (100%) |" in text
        assert "equal on 0 of 1 (0%). Sedosaur was higher on 1 and lower on 0." in text
        assert "Sedosaur found 1 of 1 (100%) real Greptile findings." in text
        assert (
            "| `cccccccccc` | 1 | 1 of 1 (100%) | 1 of 1 (100%) | 0 | 0 of 1 (0%) "
            "| $0.50 |" in text
        )
        assert f"| #7 | `{short}` | 4/5 | 5/5 | 1 | 1 | 1 |" in text

    def test_should_list_a_real_greptile_finding_sedosaur_missed(
        self, repo: Repo
    ) -> None:
        repo.commit({"a.go": "y\n"})
        gh = FakeGH(repo.head)
        gh.threads = gh.threads[:1]
        r = record(repo.head)
        r["review"]["findings"] = []
        text = compare.report(pairs(gh, repo, [r]), None, since=SINCE)
        missed = text.split("### Real Greptile findings Sedosaur missed\n\n")[1]
        assert missed.startswith("- P1 **Leaks a handle** `a.go:10` (#7)")

    def test_should_say_when_no_finding_is_real_yet(self, repo: Repo) -> None:
        gh = FakeGH(repo.head)
        gh.threads[0]["isResolved"] = False
        text = compare.report(pairs(gh, repo, [record(repo.head)]), None, since=SINCE)
        assert "- **Coverage:** no real Greptile findings yet." in text

    def test_should_show_the_limits_of_the_newest_review(self) -> None:
        text = compare.report([], limited(0.81, NOON), since=SINCE)
        assert "As of the newest review, at 2026-11-05 12:00 UTC:" in text
        assert "| `seven_day` | 81% | allowed_warning | 2026-11-09 18:00 UTC |" in text

    def test_should_say_when_no_review_recorded_the_limits(self) -> None:
        text = compare.report([], None, since=SINCE)
        assert "No recent review recorded the usage limits." in text

    def test_should_say_when_nothing_is_paired(self) -> None:
        assert compare.report([], None, since=SINCE).endswith(
            "No commit has a review from both yet."
        )
