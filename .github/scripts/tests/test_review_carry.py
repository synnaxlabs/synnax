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

from review import Target, carry

REPO = "synnaxlabs/synnax"
OLD = "a" * 40
HEAD = "b" * 40
KEY = "7:main"
TARGET = Target(
    repo=REPO,
    pr=7,
    head=HEAD,
    base="base",
    check_run=55,
    run_url="https://github.com/synnaxlabs/synnax/actions/runs/2",
    bot="sedosaur",
)
PATCH = "@@ -1,3 +1,4 @@ func f() {\n one\n-two\n+two\n+three\n four"


def lines(items: list[dict[str, Any]]) -> str:
    return "".join(json.dumps(i) + "\n" for i in items)


def summary_edit(commit: str) -> dict[str, Any]:
    block = f"[c](https://github.com/{REPO}/commit/{commit})"
    body = f"Fixes A.\n<!-- sedosaur_comment -->\n{block}\n<!-- /sedosaur_comment -->"
    return {"editor": {"__typename": "Bot", "login": "sedosaur"}, "diff": body}


def check_run(**kw: Any) -> dict[str, Any]:
    run: dict[str, Any] = {
        "app": {"slug": "sedosaur"},
        "external_id": KEY,
        "started_at": "2026-11-08T05:00:00Z",
        "conclusion": "success",
        "output": {
            "title": "Confidence Score: 4/5",
            "summary": "Fix one thing.\n\nFindings: 1 P1.",
            "text": None,
        },
    }
    return run | kw


def file(**kw: Any) -> dict[str, Any]:
    base = {"filename": "a.go", "status": "modified", "sha": "blob", "patch": PATCH}
    return base | kw


class FakeGH:
    """Serves pull request 7, whose bot reviewed OLD, and records each write."""

    def __init__(self) -> None:
        self.edits = [summary_edit(OLD)]
        self.runs = {OLD: [check_run()]}
        self.files = {OLD: [file()], HEAD: [file()]}
        self.writes: list[tuple[str, str, dict[str, Any]]] = []

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        if args[:2] == ("api", "-X"):
            assert stdin is not None
            self.writes.append((args[2], args[3], json.loads(stdin)))
            return ""
        path = args[2] if args[1] == "--paginate" else args[1]
        if path == "graphql":
            return lines(self.edits)
        for commit, runs in self.runs.items():
            if path.startswith(f"repos/{REPO}/commits/{commit}/check-runs?"):
                return lines(runs)
        for commit, files in self.files.items():
            if path == f"repos/{REPO}/compare/base...{commit}":
                return lines(files)
        raise AssertionError(f"unexpected gh call: {args}")


@pytest.fixture
def gh() -> FakeGH:
    return FakeGH()


class TestCarry:
    """Tests giving a head the newest review when its diff did not change."""

    def test_should_give_the_head_the_review_of_the_same_diff(self, gh: FakeGH) -> None:
        assert carry(TARGET, gh, key=KEY)
        [done] = gh.writes
        assert done[:2] == ("PATCH", f"repos/{REPO}/check-runs/55")
        assert done[2]["status"] == "completed"
        assert done[2]["conclusion"] == "success"
        output = done[2]["output"]
        assert output["title"] == "Confidence Score: 4/5"
        assert output["summary"] == (
            f"The diff is the same as at [aaaaaaa](https://github.com/{REPO}/commit/"
            f"{OLD}), so this is its review.\n\nFix one thing.\n\nFindings: 1 P1."
        )
        assert "text" not in output

    def test_should_keep_the_text_of_the_review(self, gh: FakeGH) -> None:
        gh.runs[OLD] = [check_run(output=check_run()["output"] | {"text": "Listed."})]
        assert carry(TARGET, gh, key=KEY)
        assert gh.writes[0][2]["output"]["text"] == "Listed."

    def test_should_carry_a_diff_whose_lines_moved(self, gh: FakeGH) -> None:
        moved = PATCH.replace("@@ -1,3 +1,4 @@", "@@ -11,3 +12,4 @@")
        gh.files[HEAD] = [file(patch=moved, sha="other")]
        assert carry(TARGET, gh, key=KEY)

    def test_should_take_the_newest_run_of_the_commit(self, gh: FakeGH) -> None:
        late = check_run(started_at="2026-11-08T06:00:00Z", conclusion="failure")
        gh.runs[OLD] = [check_run(), late]
        assert not carry(TARGET, gh, key=KEY)
        assert gh.writes == []

    @pytest.mark.parametrize(
        "changed",
        [
            [file(patch=PATCH + "\n+five")],
            [file(), file(filename="b.go")],
            [file(status="renamed", previous_filename="old.go")],
            [file(patch=None, sha="png")],
        ],
    )
    def test_should_not_carry_a_changed_diff(
        self, gh: FakeGH, changed: list[dict[str, Any]]
    ) -> None:
        gh.files[HEAD] = changed
        assert not carry(TARGET, gh, key=KEY)
        assert gh.writes == []

    def test_should_compare_a_binary_file_by_its_blob(self, gh: FakeGH) -> None:
        gh.files = {OLD: [file(patch=None, sha="png")], HEAD: [file(patch=None)]}
        assert not carry(TARGET, gh, key=KEY)
        gh.files[HEAD] = [file(patch=None, sha="png")]
        assert carry(TARGET, gh, key=KEY)

    def test_should_not_carry_a_comparison_it_cannot_list(self, gh: FakeGH) -> None:
        many = [file(filename=f"{i}.go") for i in range(300)]
        gh.files = {OLD: many, HEAD: many}
        assert not carry(TARGET, gh, key=KEY)

    @pytest.mark.parametrize(
        "runs",
        [
            [],
            [check_run(conclusion="failure")],
            [check_run(external_id="7:release/1.0")],
            [check_run(app={"slug": "greptile"})],
        ],
    )
    def test_should_not_carry_without_a_review_that_succeeded(
        self, gh: FakeGH, runs: list[dict[str, Any]]
    ) -> None:
        gh.runs[OLD] = runs
        assert not carry(TARGET, gh, key=KEY)
        assert gh.writes == []

    def test_should_not_carry_when_the_bot_reviewed_nothing(self, gh: FakeGH) -> None:
        gh.edits = []
        assert not carry(TARGET, gh, key=KEY)

    def test_should_not_carry_the_review_of_the_head_itself(self, gh: FakeGH) -> None:
        gh.edits = [summary_edit(HEAD)]
        assert not carry(TARGET, gh, key=KEY)
