#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from datetime import UTC, datetime
from typing import Any

from review.compare import (
    NOISE,
    OPEN,
    REAL,
    Item,
    Thread,
    greptile_reviews,
    match,
    pairs,
    report,
    verdict,
)

REPO = "o/r"
HEAD = "a" * 40
FINAL = "f" * 40
SINCE = datetime(2026, 11, 5, tzinfo=UTC)


def greptile_block(score: int, commit: str = HEAD) -> str:
    """A pull request body the way Greptile leaves it after it reviews a commit."""
    return (
        "Fixes A.\n\n<!-- greptile_comment -->\n"
        f"<h2>Confidence Score: {score}/5</h2>\n\n<h2>Findings</h2>\n\n"
        '1. <img alt="P1" src="p1.svg" align="top">&nbsp;**Leaks a handle** '
        '<a href="https://github.com/o/r/pull/7#discussion_r101">▶</a>\n'
        '2. <img alt="P2" src="p2.svg" align="top"> <img alt="security" src="s.svg">'
        '&nbsp;**Reads a secret** <a href="https://github.com/o/r/pull/7#issuecomment-5">'
        "▶</a>\n\n"
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
    *,
    resolved: bool = False,
    up: int = 0,
    down: int = 0,
) -> dict[str, Any]:
    badge = '<img alt="P1" src="p1.svg">'
    return {
        "isResolved": resolved,
        "path": "a.go",
        "line": line,
        "originalLine": line,
        "comments": {
            "nodes": [
                {
                    "databaseId": id,
                    "author": {"login": login},
                    "body": f"{badge} **{title}**\n\nWhy it matters.",
                    "originalCommit": {"oid": HEAD},
                    "reactionGroups": [
                        {"content": "THUMBS_UP", "reactors": {"totalCount": up}},
                        {"content": "THUMBS_DOWN", "reactors": {"totalCount": down}},
                    ],
                }
            ]
        },
    }


def record(head: str = HEAD, score: int = 5) -> dict[str, Any]:
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
    """Serves pull request 7 through the gh calls the comparison makes."""

    def __init__(self) -> None:
        self.edits = [edit(greptile_block(4))]
        self.threads = [
            thread(101, "greptile-apps", "Leaks a handle", 10, resolved=True),
            thread(201, "sedosaur", "Leaks a handle too", 12),
        ]
        self.changed = ["a.go"]
        self.calls: list[tuple[str, ...]] = []

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        self.calls.append(args)
        if args[:2] == ("api", "graphql"):
            pull = {
                "headRefOid": FINAL,
                "userContentEdits": {"nodes": self.edits},
                "reviewThreads": {"nodes": self.threads},
            }
            return json.dumps({"data": {"repository": {"pullRequest": pull}}})
        if args[1] == f"repos/{REPO}/compare/{HEAD}...{FINAL}":
            return "".join(f"{f}\n" for f in self.changed)
        raise AssertionError(f"unexpected gh call: {args}")


def t(**kw: Any) -> Thread:
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
    }
    return Thread(**(fields | kw))


def item(path: str | None, line: int | None) -> Item:
    return Item(severity="P2", title="T", path=path, line=line, verdict=OPEN)


class TestGreptileReviews:
    """Tests reading Greptile's reviews from the revisions of a pull request body."""

    def test_should_read_the_score_and_findings_of_a_commit(self) -> None:
        assert greptile_reviews([edit(greptile_block(4))]) == {
            HEAD: (4, [("P1", "Leaks a handle", 101), ("P2", "Reads a secret", None)])
        }

    def test_should_keep_the_newest_review_of_a_commit(self) -> None:
        edits = [edit(greptile_block(5)), edit(greptile_block(3))]
        assert greptile_reviews(edits)[HEAD][0] == 5

    def test_should_ignore_a_revision_by_a_person(self) -> None:
        edits = [edit(greptile_block(5), kind="User"), edit(greptile_block(5), "pat")]
        assert greptile_reviews(edits) == {}

    def test_should_ignore_a_deleted_revision(self) -> None:
        assert greptile_reviews([{"editor": None, "diff": None}]) == {}


class TestVerdict:
    """Tests deciding whether a finding was real."""

    def test_should_take_more_thumbs_down_as_noise(self) -> None:
        assert verdict(t(up=1, down=2, resolved=True), {"a.go"}) == NOISE

    def test_should_take_more_thumbs_up_as_real(self) -> None:
        assert verdict(t(up=1), set()) == REAL

    def test_should_keep_an_unresolved_thread_open(self) -> None:
        assert verdict(t(), {"a.go"}) == OPEN

    def test_should_take_a_fix_as_real(self) -> None:
        assert verdict(t(resolved=True), {"a.go"}) == REAL

    def test_should_take_a_resolve_without_a_fix_as_noise(self) -> None:
        assert verdict(t(resolved=True), {"b.go"}) == NOISE


class TestMatch:
    """Tests matching the findings of two reviews."""

    def test_should_pair_the_closest_findings_once(self) -> None:
        a = [item("a.go", 10), item("a.go", 20)]
        b = [item("a.go", 19), item("a.go", 11)]
        assert sorted(match(a, b)) == [(0, 1), (1, 0)]

    def test_should_not_pair_other_files_or_far_lines(self) -> None:
        a = [item("a.go", 10), item("a.go", 30), item(None, None)]
        b = [item("b.go", 10), item("a.go", 40), item(None, None)]
        assert match(a, b) == []


class TestPairs:
    """Tests pairing Sedosaur's records with Greptile's reviews."""

    def test_should_pair_a_commit_both_reviewed(self) -> None:
        [p] = pairs(FakeGH(), REPO, [record(), record(head="d" * 40)])
        assert (p.pr, p.head, p.scores) == (7, HEAD, (4, 5))
        assert [(i.title, i.path, i.line, i.verdict) for i in p.greptile] == [
            ("Leaks a handle", "a.go", 10, REAL),
            ("Reads a secret", None, None, OPEN),
        ]
        assert [(i.title, i.verdict) for i in p.sedosaur] == [
            ("Leaks a handle too", REAL),
            ("Odd name", OPEN),
        ]
        assert p.matches == [(0, 0)]

    def test_should_count_the_newest_record_of_a_commit(self) -> None:
        [p] = pairs(FakeGH(), REPO, [record(score=2), record(score=4)])
        assert p.scores == (4, 4)

    def test_should_take_a_resolve_without_a_fix_as_noise(self) -> None:
        gh = FakeGH()
        gh.changed = ["c.go"]
        [p] = pairs(gh, REPO, [record()])
        assert p.greptile[0].verdict == NOISE
        assert p.sedosaur[0].verdict == NOISE


class TestReport:
    """Tests the Markdown report of the comparison."""

    def test_should_report_coverage_and_precision(self) -> None:
        text = report(pairs(FakeGH(), REPO, [record()]), since=SINCE)
        assert "1 commit on 1 pull request have a review from both." in text
        assert "| Precision | 1 of 1 (100%) | 1 of 1 (100%) |" in text
        assert "equal on 0 of 1 (0%). Sedosaur was higher on 1 and lower on 0." in text
        assert "Sedosaur found 1 of 1 (100%) real Greptile findings." in text
        assert "| `cccccccccc` | 1 | 0 of 1 (0%) | 1 of 1 (100%) |" in text
        assert "| #7 | `aaaaaaaaaa` | 4/5 | 5/5 | 1 | 1 | 1 |" in text

    def test_should_list_a_real_greptile_finding_sedosaur_missed(self) -> None:
        gh = FakeGH()
        gh.threads = gh.threads[:1]
        r = record()
        r["review"]["findings"] = []
        text = report(pairs(gh, REPO, [r]), since=SINCE)
        missed = text.split("### Real Greptile findings Sedosaur missed\n\n")[1]
        assert missed.startswith("- P1 **Leaks a handle** `a.go:10` (#7)")

    def test_should_say_when_no_finding_is_real_yet(self) -> None:
        gh = FakeGH()
        gh.threads[0]["isResolved"] = False
        text = report(pairs(gh, REPO, [record()]), since=SINCE)
        assert "- **Coverage:** no real Greptile findings yet." in text

    def test_should_say_when_nothing_is_paired(self) -> None:
        assert report([], since=SINCE).endswith("No commit has a review from both yet.")
