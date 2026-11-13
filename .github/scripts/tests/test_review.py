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
from pathlib import Path
from typing import Any

import pytest

from review import Finding, Review, Target, gh as call_gh, parse, post, render, reviewed

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
        self.later: list[dict[str, Any]] | None = None
        """The threads after the first read, if they changed."""
        self.runs: list[dict[str, Any]] = []
        self.edits: list[dict[str, Any]] = []
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
            query = next(a for a in args if a.startswith("query="))
            if "userContentEdits" in query:
                return lines(self.edits)
            threads = self.threads
            if self.later is not None:
                self.threads = self.later
            return lines(threads)
        if path == f"repos/{REPO}/check-runs/55":
            return "7:main\n"
        if path.startswith(f"repos/{REPO}/commits/head/check-runs?"):
            return lines(self.runs)
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

    @pytest.mark.parametrize(
        ("severity", "deferrable", "blocks"),
        [("P1", True, False), ("P2", False, True), ("P0", True, True)],
    )
    def test_should_read_whether_a_finding_blocks_merge(
        self, severity: str, deferrable: bool, blocks: bool
    ) -> None:
        f = {"severity": severity, "title": "T", "body": "B", "path": "x.go", "line": 9}
        data = {"score": 4, "verdict": "V", "summary": [], "findings": []}
        data["findings"] = [f | {"deferrable": deferrable}]
        assert parse(data).findings[0].blocks is blocks

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
        gh.threads = [thread("sedosaur")]
        post(review(finding()), target=TARGET, gh=gh)
        assert gh.written("POST", "pulls/7/reviews") == []

    def test_should_mark_an_open_thread_whose_finding_became_a_follow_up(
        self, gh: FakeGH
    ) -> None:
        gh.threads = [posted(finding())]
        post(review(finding(deferrable=True)), target=TARGET, gh=gh)
        [edit] = gh.written("PATCH", "pulls/comments/9")
        assert "**Samples return old data** (follow-up)\n" in edit["body"]
        assert gh.written("POST", "pulls/7/reviews") == []

    def test_should_leave_an_open_thread_whose_heading_is_the_same(
        self, gh: FakeGH
    ) -> None:
        gh.threads = [posted(finding())]
        post(review(finding()), target=TARGET, gh=gh)
        assert gh.written("PATCH", "pulls/comments/9") == []

    def test_should_post_a_finding_of_a_resolved_thread(self, gh: FakeGH) -> None:
        gh.threads = [thread("sedosaur", resolved=True)]
        post(review(finding()), target=TARGET, gh=gh)
        assert len(gh.written("POST", "pulls/7/reviews")) == 1

    def test_should_post_a_finding_another_author_raised(self, gh: FakeGH) -> None:
        gh.threads = [thread("greptile-apps")]
        post(review(finding()), target=TARGET, gh=gh)
        assert len(gh.written("POST", "pulls/7/reviews")) == 1


class TestDismissedFindings:
    """Tests leaving out the findings that a member dismissed."""

    def test_should_leave_out_a_dismissed_finding(self, gh: FakeGH) -> None:
        gh.threads = [thread("sedosaur", "@sedosaur dismiss", resolved=True)]
        post(
            review(finding(), finding(severity="P2", title="Nit")), target=TARGET, gh=gh
        )
        [posted] = gh.written("POST", "pulls/7/reviews")
        assert [c["body"].count("**Nit**") for c in posted["comments"]] == [1]
        [done] = gh.written("PATCH", "check-runs/55")
        assert done["output"]["title"] == "Confidence Score: 4/5"
        assert render.listed(done["output"]["text"]) == {("a.go", "Nit")}

    def test_should_score_5_when_every_finding_is_dismissed(self, gh: FakeGH) -> None:
        gh.threads = [thread("sedosaur", "@sedosaur dismiss", resolved=True)]
        post(review(finding()), target=TARGET, gh=gh)
        assert gh.written("POST", "pulls/7/reviews") == []
        [done] = gh.written("PATCH", "check-runs/55")
        assert done["output"]["title"] == "Confidence Score: 5/5"
        assert done["output"]["summary"] == (
            "A member dismissed every finding of the review that blocks merge.\n\n"
            "Findings: none."
        )

    def test_should_score_5_when_only_follow_ups_are_left(self, gh: FakeGH) -> None:
        gh.threads = [thread("sedosaur", "@sedosaur dismiss", resolved=True)]
        later = finding(severity="P2", title="Later", line=3, deferrable=True)
        post(review(finding(), later), target=TARGET, gh=gh)
        [posted] = gh.written("POST", "pulls/7/reviews")
        assert [c["body"].count("**Later**") for c in posted["comments"]] == [1]
        [done] = gh.written("PATCH", "check-runs/55")
        assert done["output"]["title"] == "Confidence Score: 5/5"
        assert done["output"]["summary"].endswith("Findings: 1 follow-up.")

    def test_should_score_the_head_5_after_a_dismissal_while_posting(
        self, gh: FakeGH
    ) -> None:
        gh.threads = [thread("sedosaur")]
        gh.later = [thread("sedosaur", "@sedosaur dismiss", resolved=True)]
        gh.runs = [
            {
                "name": "Sedosaur Review",
                "app": {"slug": "sedosaur"},
                "external_id": "7:main",
                "started_at": "2026-11-08T05:00:00Z",
                "conclusion": "success",
                "output": render.check(review(finding())),
            }
        ]
        post(review(finding()), target=TARGET, gh=gh)
        [check] = gh.written("POST", "check-runs")
        assert check["output"]["title"] == "Confidence Score: 5/5"
        assert check["external_id"] == "7:main"

    def test_should_keep_the_review_when_the_late_read_fails(
        self, gh: FakeGH, capsys: pytest.CaptureFixture[str]
    ) -> None:
        def fail(*args: str, stdin: str | None = None) -> str:
            if args[1] == f"repos/{REPO}/check-runs/55":
                raise subprocess.CalledProcessError(1, ["gh", *args])
            return gh(*args, stdin=stdin)

        post(review(finding()), target=TARGET, gh=fail)
        [done] = gh.written("PATCH", "check-runs/55")
        assert done["output"]["title"] == "Confidence Score: 4/5"
        assert "::warning::Could not read dismissals" in capsys.readouterr().err

    def test_should_keep_a_finding_whose_dismissal_is_quoted(self, gh: FakeGH) -> None:
        reply = "> reply `@sedosaur dismiss` and a reason\n\nNo, I disagree."
        gh.threads = [thread("sedosaur", reply)]
        post(review(finding()), target=TARGET, gh=gh)
        [done] = gh.written("PATCH", "check-runs/55")
        assert done["output"]["title"] == "Confidence Score: 4/5"


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

    def test_should_encode_a_path_in_a_link(self, gh: FakeGH) -> None:
        gh.files.append({"filename": "lv/Synnax Client.lvclass", "patch": PATCH})
        spaced = finding(path="lv/Synnax Client.lvclass", rule="docs/My Rules.md")
        post(review(spaced, finding(path="old file.go", line=9)), target=TARGET, gh=gh)
        blob = f"https://github.com/{REPO}/blob/head"
        [edit] = gh.written("PATCH", "pulls/7")
        link = f"[`lv/Synnax Client.lvclass:2`]({blob}/lv/Synnax%20Client.lvclass#L2)"
        assert link in edit["body"]
        [posted] = gh.written("POST", "pulls/7/reviews")
        rule = f"[`docs/My Rules.md`]({blob}/docs/My%20Rules.md)"
        assert rule in posted["comments"][0]["body"]
        [comment] = gh.written("POST", "issues/7/comments")
        assert f"({blob}/old%20file.go#L9)" in comment["body"]

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
        assert render.listed(done["output"]["text"]) == {
            ("a.go", "Samples return old data"),
            ("a.go", "Nit"),
        }

    def test_should_mark_a_follow_up_and_leave_it_out_of_the_listing(
        self, gh: FakeGH
    ) -> None:
        later = finding(severity="P2", title="Later", line=3, deferrable=True)
        post(review(finding(), later), target=TARGET, gh=gh)
        [posted] = gh.written("POST", "pulls/7/reviews")
        assert "**Later** (follow-up)" in posted["comments"][1]["body"]
        [done] = gh.written("PATCH", "check-runs/55")
        assert done["output"]["summary"].endswith("Findings: 1 P1, 1 follow-up.")
        assert render.listed(done["output"]["text"]) == {
            ("a.go", "Samples return old data")
        }


class TestListing:
    """Tests the hidden list of findings in the text of a check run."""

    def test_should_read_back_what_it_wrote(self) -> None:
        text = "Above.\n" + render.listing([finding(title="A --> B")]) + "\nBelow."
        assert text.count("-->") == 1
        assert render.listed(text) == {("a.go", "A --> B")}

    def test_should_read_an_empty_list(self) -> None:
        assert render.listed(render.listing([])) == set()

    def test_should_read_none_from_a_text_without_a_list(self) -> None:
        assert render.listed("Findings: 1 P1.") is None


def thread(login: str, *replies: str, resolved: bool = False) -> dict[str, Any]:
    def comment(body: str, kind: str, author: str) -> dict[str, Any]:
        who = {"__typename": kind, "login": author}
        return {
            "databaseId": 9,
            "author": who,
            "authorAssociation": "MEMBER",
            "body": body,
        }

    body = "<picture></picture> **Samples return old data**\n\nOld text."
    comments = [comment(body, "Bot", login)]
    comments += [comment(r, "User", "member") for r in replies]
    return {"id": "T", "path": "a.go", "resolved": resolved, "comments": comments}


def posted(f: Finding) -> dict[str, Any]:
    """An open thread of the bot that holds a finding as the poster wrote it."""
    th = thread("sedosaur")
    th["comments"][0]["body"] = render.comment(f, blob="https://github.com/b")
    return th


def summary_edit(commit: str, login: str = "sedosaur") -> dict[str, Any]:
    block = f"[c](https://github.com/{REPO}/commit/{commit})"
    body = f"Fixes A.\n<!-- sedosaur_comment -->\n{block}\n<!-- /sedosaur_comment -->"
    return {"editor": {"__typename": "Bot", "login": login}, "diff": body}


class TestReviewed:
    """Tests reading the commits that the bot reviewed."""

    def test_should_list_them_newest_first_once_each(self, gh: FakeGH) -> None:
        a, b = "a" * 40, "b" * 40
        gh.edits = [summary_edit(b), summary_edit(a), summary_edit(b)]
        assert reviewed(TARGET, gh) == [b, a]

    def test_should_skip_an_edit_by_another_author(self, gh: FakeGH) -> None:
        gh.edits = [summary_edit("a" * 40, login="pat"), {"editor": None, "diff": None}]
        assert reviewed(TARGET, gh) == []


class TestGH:
    """Tests the helper that runs the `gh` executable."""

    def test_should_show_the_error_of_a_failed_call(
        self,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
        capfd: pytest.CaptureFixture[str],
    ) -> None:
        fake = tmp_path / "gh"
        fake.write_text(
            "#!/bin/sh\necho 'HTTP 422: Unprocessable Entity' >&2\nexit 1\n"
        )
        fake.chmod(0o755)
        monkeypatch.setenv("PATH", f"{tmp_path}{os.pathsep}{os.environ['PATH']}")
        with pytest.raises(subprocess.CalledProcessError):
            call_gh("api", "-X", "POST", f"repos/{REPO}/pulls/7/reviews")
        assert "HTTP 422: Unprocessable Entity" in capfd.readouterr().err
