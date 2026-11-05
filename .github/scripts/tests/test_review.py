#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from typing import Any

import pytest

from review import Finding, Review, Target, parse, post

REPO = "synnaxlabs/synnax"
TARGET = Target(
    repo=REPO,
    pr=7,
    head="head",
    base="base",
    check_run=55,
    run_url="https://github.com/synnaxlabs/synnax/actions/runs/1",
    bot="sedosaur",
)
# Lines 1-4 and 21-22 of the new file are in the diff.
PATCH = (
    "@@ -1,3 +1,4 @@\n one\n-two\n+two\n+three\n four\n@@ -20,2 +21,2 @@\n ctx\n+new"
)
OUTSIDE = "<!-- sedosaur_outside_diff -->"
GREPTILE = "<!-- greptile_comment -->\nGreptile's summary\n<!-- /greptile_comment -->"
MESSAGE = "[ci] Bound MQTT samples and give each discovery request a deadline"


def finding(path: str = "a.go", line: int = 2, **kw: Any) -> Finding:
    fields: dict[str, Any] = {
        "severity": "P1",
        "title": "Samples return old data",
        "body": "It reads the oldest record.",
    }
    return Finding(**(fields | kw), path=path, line=line)


def review(*findings: Finding, score: int = 4) -> Review:
    return Review(
        score=score,
        verdict="One finding should be fixed before merge.",
        summary=["Bounds the samples."],
        findings=list(findings),
    )


def lines(items: list[dict[str, Any]]) -> str:
    return "".join(json.dumps(i) + "\n" for i in items)


class FakeGH:
    """Serves pull request 7 through the gh calls the poster makes, and records each
    write."""

    def __init__(self) -> None:
        self.files: list[dict[str, Any]] = [
            {"filename": "a.go", "patch": PATCH},
            {"filename": "image.png", "patch": None},
        ]
        self.threads: list[dict[str, Any]] = []
        self.comments: list[dict[str, Any]] = []
        self.body = "Fixes the thing."
        self.writes: list[tuple[str, str, dict[str, Any]]] = []

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        if args[:2] == ("api", "-X"):
            assert stdin is not None
            self.writes.append((args[2], args[3], json.loads(stdin)))
            return ""
        path = args[2] if args[1] == "--paginate" else args[1]
        if path.startswith(f"repos/{REPO}/compare/base...head"):
            return lines(self.files)
        if path == "graphql":
            return lines(self.threads)
        if path == f"repos/{REPO}/issues/7/comments":
            return lines(self.comments)
        if path == f"repos/{REPO}/commits/head":
            return MESSAGE + "\n"
        if path == f"repos/{REPO}/pulls/7":
            return self.body + "\n"
        raise AssertionError(f"unexpected gh call: {args}")

    def written(self, method: str, path: str) -> list[dict[str, Any]]:
        return [
            p for m, at, p in self.writes if (m, at) == (method, f"repos/{REPO}/{path}")
        ]


@pytest.fixture
def gh() -> FakeGH:
    return FakeGH()


class TestParse:
    """Tests building a review from the JSON the reviewer writes."""

    def test_should_build_a_review_with_its_findings(self) -> None:
        data = {
            "score": 3,
            "verdict": "Unsafe.",
            "summary": ["Adds a cache."],
            "findings": [
                {
                    "severity": "P0",
                    "title": "Leak",
                    "body": "The cache never evicts.",
                    "path": "x.go",
                    "line": 9,
                    "start_line": 7,
                    "rule": "AGENTS.md",
                }
            ],
        }
        got = parse(data)
        assert got.score == 3
        assert got.findings[0].lines == range(7, 10)
        assert got.findings[0].rule == "AGENTS.md"

    @pytest.mark.parametrize(
        ("change", "error"),
        [
            ({"score": 6}, "score 6 is not between 1 and 5"),
            ({"severity": "P3"}, "severity 'P3' is not one of"),
            ({"start_line": 10}, "lines 10-9 of x.go are invalid"),
            ({"start_line": 0}, "lines 0-9 of x.go are invalid"),
            ({"line": 0}, "lines None-0 of x.go are invalid"),
        ],
    )
    def test_should_reject_values_out_of_range(
        self, change: dict[str, Any], error: str
    ) -> None:
        f = {"severity": "P1", "title": "T", "body": "B", "path": "x.go", "line": 9}
        top = {"score": change.pop("score", 4)}
        data = top | {"verdict": "V", "summary": [], "findings": [f | change]}
        with pytest.raises(ValueError, match=error):
            parse(data)

    def test_should_reject_an_unknown_field(self) -> None:
        f = {"severity": "P1", "title": "T", "body": "B", "path": "x.go", "line": 9}
        data = {"score": 4, "verdict": "V", "summary": [], "findings": [f | {"x": 1}]}
        with pytest.raises(TypeError, match="unexpected keyword argument 'x'"):
            parse(data)


class TestInlineFindings:
    """Tests posting findings on lines of the diff as review threads."""

    def test_should_post_them_as_one_review_of_the_head(self, gh: FakeGH) -> None:
        fix = finding(line=3, start_line=2, suggestion="two\nthree", rule="AGENTS.md")
        post(review(fix, finding(line=22, title="Other")), target=TARGET, gh=gh)
        [posted] = gh.written("POST", "pulls/7/reviews")
        assert posted["commit_id"] == "head"
        assert posted["event"] == "COMMENT"
        first, second = posted["comments"]
        assert {k: v for k, v in first.items() if k != "body"} == {
            "path": "a.go",
            "line": 3,
            "side": "RIGHT",
            "start_line": 2,
            "start_side": "RIGHT",
        }
        assert second["line"] == 22
        assert "start_line" not in second
        assert first["body"].startswith('<img alt="P1" src="')
        assert "**Samples return old data**" in first["body"]
        assert "```suggestion\ntwo\nthree\n```" in first["body"]
        rule = f"[`AGENTS.md`](https://github.com/{REPO}/blob/head/AGENTS.md)"
        assert rule in first["body"]

    def test_should_fence_a_suggestion_past_its_own_backticks(self, gh: FakeGH) -> None:
        post(review(finding(suggestion="x := `a````")), target=TARGET, gh=gh)
        [posted] = gh.written("POST", "pulls/7/reviews")
        assert "`````suggestion\nx := `a````\n`````" in posted["comments"][0]["body"]

    def test_should_skip_a_finding_an_open_thread_of_the_bot_holds(
        self, gh: FakeGH
    ) -> None:
        body = "<picture></picture> **Samples return old data**\n\nOld text."
        gh.threads = [
            {"path": "a.go", "first": {"author": {"login": "sedosaur"}, "body": body}}
        ]
        post(review(finding()), target=TARGET, gh=gh)
        assert gh.written("POST", "pulls/7/reviews") == []

    def test_should_post_a_finding_another_author_raised(self, gh: FakeGH) -> None:
        body = "<picture></picture> **Samples return old data**"
        author = {"login": "greptile-apps"}
        gh.threads = [{"path": "a.go", "first": {"author": author, "body": body}}]
        post(review(finding()), target=TARGET, gh=gh)
        assert len(gh.written("POST", "pulls/7/reviews")) == 1


class TestOutsideFindings:
    """Tests listing findings off the diff in the bot's one outside-diff comment."""

    def test_should_post_them_in_a_new_comment(self, gh: FakeGH) -> None:
        post(
            review(finding(line=10), finding(path="b.go", line=1, title="Gone")),
            target=TARGET,
            gh=gh,
        )
        assert gh.written("POST", "pulls/7/reviews") == []
        [comment] = gh.written("POST", "issues/7/comments")
        body = comment["body"]
        assert body.startswith(OUTSIDE)
        assert "These findings could not be posted inline." in body
        assert f"[`a.go:10`](https://github.com/{REPO}/blob/head/a.go#L10)" in body
        assert "**Gone**" in body

    def test_should_edit_the_bot_comment_when_none_remain(self, gh: FakeGH) -> None:
        gh.comments = [
            {"id": 8, "login": "greptile-apps[bot]", "body": OUTSIDE},
            {"id": 9, "login": "sedosaur[bot]", "body": f"{OUTSIDE}\nold"},
        ]
        post(review(finding()), target=TARGET, gh=gh)
        [edit] = gh.written("PATCH", "issues/comments/9")
        assert "No findings outside the diff remain." in edit["body"]
        assert gh.written("POST", "issues/7/comments") == []

    def test_should_post_no_comment_when_none_is_needed(self, gh: FakeGH) -> None:
        post(review(finding()), target=TARGET, gh=gh)
        assert [w for w in gh.writes if "/comments" in w[1]] == []


class TestSummary:
    """Tests the summary that heads the pull request body."""

    def test_should_append_it_and_count_the_first_review(self, gh: FakeGH) -> None:
        post(review(finding()), target=TARGET, gh=gh)
        [edit] = gh.written("PATCH", "pulls/7")
        body = edit["body"]
        assert body.startswith("Fixes the thing.\n\n<!-- sedosaur_comment -->")
        assert 'alt="4 of 5"' in body
        assert "</picture> Confidence Score: 4/5</h2>" in body
        assert "<!-- sedosaur_confidence_score:4 -->" in body
        label = "\\[ci\\] Bound MQTT samples and give each..."
        commit = f"https://github.com/{REPO}/commit/head"
        assert f"Reviews (1) · Last reviewed commit: [{label}]({commit})" in body
        assert body.endswith("<!-- /sedosaur_comment -->\n")

    def test_should_list_each_finding_with_a_link(self, gh: FakeGH) -> None:
        post(
            review(finding(), finding(path="b.go", line=9, title="Gone")),
            target=TARGET,
            gh=gh,
        )
        [edit] = gh.written("PATCH", "pulls/7")
        body = edit["body"]
        blob = f"https://github.com/{REPO}/blob/head"
        assert '1. <img alt="P1"' in body
        assert f"**Samples return old data** [`a.go:2`]({blob}/a.go#L2)" in body
        assert '2. <img alt="P1"' in body
        assert f"**Gone** [`b.go:9`]({blob}/b.go#L9)" in body
        assert body.index("<h3>Findings</h3>") < body.index("<h3>Summary</h3>")

    def test_should_list_no_findings_when_there_are_none(self, gh: FakeGH) -> None:
        post(review(score=5), target=TARGET, gh=gh)
        [edit] = gh.written("PATCH", "pulls/7")
        assert "<h3>Findings</h3>" not in edit["body"]

    def test_should_replace_it_in_place_and_count_each_review(self, gh: FakeGH) -> None:
        old = "\n".join(
            [
                "<!-- sedosaur_comment -->",
                "<sub>Reviews (2)</sub>",
                "<!-- /sedosaur_comment -->",
            ]
        )
        gh.body = f"Fixes the thing.\n\n{old}\n\n{GREPTILE}"
        post(review(score=5), target=TARGET, gh=gh)
        [edit] = gh.written("PATCH", "pulls/7")
        body = edit["body"]
        assert body.count("<!-- sedosaur_comment -->") == 1
        assert "Reviews (3)" in body
        assert body.endswith(f"<!-- /sedosaur_comment -->\n\n{GREPTILE}")

    def test_should_complete_the_check_run_with_the_score(self, gh: FakeGH) -> None:
        post(
            review(finding(), finding(severity="P2", title="Nit")), target=TARGET, gh=gh
        )
        [done] = gh.written("PATCH", "check-runs/55")
        assert done["status"] == "completed"
        assert done["conclusion"] == "success"
        assert done["output"]["title"] == "Confidence Score: 4/5"
        assert done["output"]["summary"].endswith("Findings: 1 P1, 1 P2.")
