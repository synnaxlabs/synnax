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
import subprocess
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import pytest

from review import compare

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
) -> dict[str, Any]:
    badge = '<img alt="P1" src="p1.svg">'
    last = [{"author": {"login": login}, "createdAt": "2026-11-05T09:00:00Z"}]
    if replied is not None:
        last += [
            {"author": {"login": "pat"}, "createdAt": replied},
            {"author": {"login": "greptile-apps"}, "createdAt": "2026-11-05T23:00:00Z"},
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
        self.artifacts: list[tuple[str, str]] = []
        self.records: dict[str, dict[str, Any]] = {}

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        if args[:3] == ("api", "graphql", "--paginate"):
            query = args[args.index("-f") + 1]
            assert "$endCursor" in query
            nodes = self.edits if "userContentEdits" in query else self.threads
            return "".join(f"{json.dumps(n)}\n" for n in nodes)
        if args[:3] == (
            "api",
            "--paginate",
            f"repos/{REPO}/actions/artifacts?name=sedosaur-review&per_page=100",
        ):
            return "".join(f"{created} {run}\n" for created, run in self.artifacts)
        if args[:2] == ("run", "download"):
            into = Path(args[args.index("--dir") + 1])
            into.mkdir(parents=True)
            (into / "review.json").write_text(json.dumps(self.records[args[2]]))
            return ""
        raise AssertionError(f"unexpected gh call: {args}")


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

    def test_should_not_pair_other_files_or_far_lines(self) -> None:
        a = [item("a.go", 10), item("a.go", 30), item(None, None)]
        b = [item("b.go", 10), item("a.go", 40), item(None, None)]
        assert compare.match(a, b) == []


class TestRecords:
    """Tests downloading the records of Sedosaur's reviews."""

    def test_should_read_the_records_since_a_time_oldest_first(self) -> None:
        gh = FakeGH()
        gh.artifacts = [
            ("2026-11-06T10:00:00Z", "3"),
            ("2026-11-04T10:00:00Z", "1"),
            ("2026-11-05T12:00:00Z", "2"),
        ]
        gh.records = {run: record(run * 40) for run in ("1", "2", "3")}
        assert [r["head"] for r in compare.records(gh, REPO, SINCE)] == [
            "2" * 40,
            "3" * 40,
        ]


class TestPairs:
    """Tests pairing Sedosaur's records with Greptile's reviews."""

    def test_should_pair_a_commit_both_reviewed(self, repo: Repo) -> None:
        repo.commit({"a.go": "y\n"})
        gh = FakeGH(repo.head)
        [p] = compare.pairs(gh, repo.git, REPO, [record(repo.head), record("d" * 40)])
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

    def test_should_count_the_newest_record_of_a_commit(self, repo: Repo) -> None:
        gh = FakeGH(repo.head)
        recs = [record(repo.head, score=2), record(repo.head, score=4)]
        [p] = compare.pairs(gh, repo.git, REPO, recs)
        assert p.scores == (4, 4)

    def test_should_take_a_resolve_without_a_fix_as_noise(self, repo: Repo) -> None:
        repo.commit({"c.go": "2\n"})
        [p] = compare.pairs(FakeGH(repo.head), repo.git, REPO, [record(repo.head)])
        assert p.greptile[0].verdict == compare.NOISE
        assert p.sedosaur[0].verdict == compare.NOISE

    def test_should_not_count_a_merge_from_main_as_a_fix(self, repo: Repo) -> None:
        repo.merge_main({"a.go": "".join(f"{n}\n" for n in range(1, 8)) + "z\n"})
        [p] = compare.pairs(FakeGH(repo.head), repo.git, REPO, [record(repo.head)])
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
        [p] = compare.pairs(gh, repo.git, REPO, [record(repo.head)])
        assert p.greptile[0].verdict == compare.NOISE

    def test_should_keep_open_a_thread_on_a_commit_outside_the_history(
        self, repo: Repo
    ) -> None:
        gh = FakeGH(repo.head)
        gh.threads[0] = thread(101, "greptile-apps", "Leaks a handle", 10, "e" * 40)
        gh.threads[0]["isResolved"] = True
        [p] = compare.pairs(gh, repo.git, REPO, [record(repo.head)])
        assert p.greptile[0].verdict == compare.OPEN

    def test_should_judge_each_review_by_its_own_thread(self, repo: Repo) -> None:
        later = repo.commit({"c.go": "2\n"})
        gh = FakeGH(repo.head)
        gh.edits = [edit(greptile_block(3, later)), edit(greptile_block(4, repo.head))]
        gh.threads[1:] = [
            thread(201, "sedosaur", "Leaks a handle too", 12, repo.head, up=1),
            thread(202, "sedosaur", "Leaks a handle too", 12, later, down=1),
        ]
        first, second = compare.pairs(
            gh, repo.git, REPO, [record(repo.head), record(later)]
        )
        assert first.sedosaur[0].verdict == compare.REAL
        assert second.sedosaur[0].verdict == compare.NOISE


class TestReport:
    """Tests the Markdown report of the comparison."""

    def test_should_report_coverage_and_precision(self, repo: Repo) -> None:
        repo.commit({"a.go": "y\n"})
        ps = compare.pairs(FakeGH(repo.head), repo.git, REPO, [record(repo.head)])
        text = compare.report(ps, since=SINCE)
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
        text = compare.report(compare.pairs(gh, repo.git, REPO, [r]), since=SINCE)
        missed = text.split("### Real Greptile findings Sedosaur missed\n\n")[1]
        assert missed.startswith("- P1 **Leaks a handle** `a.go:10` (#7)")

    def test_should_say_when_no_finding_is_real_yet(self, repo: Repo) -> None:
        gh = FakeGH(repo.head)
        gh.threads[0]["isResolved"] = False
        text = compare.report(
            compare.pairs(gh, repo.git, REPO, [record(repo.head)]), since=SINCE
        )
        assert "- **Coverage:** no real Greptile findings yet." in text

    def test_should_say_when_nothing_is_paired(self) -> None:
        assert compare.report([], since=SINCE).endswith(
            "No commit has a review from both yet."
        )
