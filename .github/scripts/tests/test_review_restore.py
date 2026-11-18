#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from pathlib import Path
from typing import Any

import pytest

from review import Target, Unreviewed, render, restore
from review.__main__ import main

REPO = "synnaxlabs/synnax"
OLD = "a" * 40
HEAD = "b" * 40
KEY = "7:main"
TARGET = Target(
    repo=REPO,
    pr=7,
    head=HEAD,
    base="base",
    check_run=None,
    run_url="https://github.com/synnaxlabs/synnax/actions/runs/2",
    bot="sedosaur",
)
TEXT = "## Description\n\nFixes the thing."


def block(commit: str, score: int) -> str:
    return (
        f"{render.SUMMARY_START}\nConfidence Score: {score}/5\n"
        f"[c](https://github.com/{REPO}/commit/{commit})\n{render.SUMMARY_END}"
    )


def edit(login: str, body: str) -> dict[str, Any]:
    return {"editor": {"__typename": "Bot", "login": login}, "diff": body}


def lines(items: list[dict[str, Any]]) -> str:
    return "".join(json.dumps(i) + "\n" for i in items)


class FakeGH:
    """Serves pull request 7, whose bot reviewed OLD and then HEAD, after a person put
    back the body of the review of OLD."""

    def __init__(self) -> None:
        self.body = f"{TEXT}\n\n{block(OLD, 3)}\n"
        self.edits = [
            edit("someone", self.body),
            edit("sedosaur", f"{TEXT}\n\n{block(HEAD, 5)}\n"),
            edit("sedosaur", self.body),
        ]
        self.runs = [
            {
                "app": {"slug": "sedosaur"},
                "external_id": KEY,
                "started_at": "2026-11-18T05:00:00Z",
                "conclusion": "success",
            }
        ]

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        assert args[:2] != ("api", "-X"), "restore must not write"
        path = args[2] if args[1] == "--paginate" else args[1]
        if path == "graphql":
            return lines(self.edits)
        if path.startswith(f"repos/{REPO}/commits/{HEAD}/check-runs?"):
            return lines(self.runs)
        if path == f"repos/{REPO}/pulls/7":
            return self.body + "\n"
        raise AssertionError(f"unexpected gh call: {args}")


@pytest.fixture
def gh() -> FakeGH:
    return FakeGH()


class TestRestore:
    """Tests putting the bot's last summary back in the pull request body."""

    def test_should_put_back_the_summary_the_bot_wrote_last(self, gh: FakeGH) -> None:
        body = restore(TARGET, gh, key=KEY)
        assert body is not None
        assert block(HEAD, 5) in body
        assert block(OLD, 3) not in body

    def test_should_leave_the_rest_of_the_body_the_same(self, gh: FakeGH) -> None:
        gh.body = f"{TEXT}\n\n{block(OLD, 3)}\n\nA later note."
        body = restore(TARGET, gh, key=KEY)
        assert body == f"{TEXT}\n\n{block(HEAD, 5)}\n\nA later note."

    def test_should_append_the_summary_when_the_body_lost_it(self, gh: FakeGH) -> None:
        gh.body = TEXT
        assert restore(TARGET, gh, key=KEY) == f"{TEXT}\n\n{block(HEAD, 5)}\n"

    def test_should_return_none_when_the_body_holds_it(self, gh: FakeGH) -> None:
        gh.body = f"{TEXT}\n\n{block(HEAD, 5)}"
        assert restore(TARGET, gh, key=KEY) is None

    def test_should_fail_when_the_head_has_no_review(self, gh: FakeGH) -> None:
        gh.runs = []
        with pytest.raises(Unreviewed, match=f"no finished review of {HEAD}"):
            restore(TARGET, gh, key=KEY)

    def test_should_fail_when_the_review_of_the_head_failed(self, gh: FakeGH) -> None:
        gh.runs[0]["conclusion"] = "failure"
        with pytest.raises(Unreviewed, match="no finished review"):
            restore(TARGET, gh, key=KEY)

    def test_should_fail_when_the_bot_wrote_no_summary(self, gh: FakeGH) -> None:
        gh.edits = [edit("someone", gh.body)]
        with pytest.raises(Unreviewed, match="never wrote a summary"):
            restore(TARGET, gh, key=KEY)


class TestMain:
    """Tests the restore command."""

    @pytest.fixture
    def env(self, monkeypatch: pytest.MonkeyPatch, gh: FakeGH) -> None:
        for k, v in {
            "GITHUB_REPOSITORY": REPO,
            "PR": "7",
            "HEAD_SHA": HEAD,
            "BASE_SHA": "base",
            "RUN_URL": TARGET.run_url,
            "BOT": "sedosaur",
            "KEY": KEY,
        }.items():
            monkeypatch.setenv(k, v)
        monkeypatch.delenv("CHECK_RUN", raising=False)
        monkeypatch.setattr("review.__main__.gh", gh)

    @pytest.mark.usefixtures("env")
    def test_should_write_the_payload_of_the_update(self, tmp_path: Path) -> None:
        out = tmp_path / "body.json"
        main(["restore", str(out)])
        assert block(HEAD, 5) in json.loads(out.read_text())["body"]

    @pytest.mark.usefixtures("env")
    def test_should_write_nothing_when_the_body_holds_it(
        self, gh: FakeGH, tmp_path: Path
    ) -> None:
        gh.body = block(HEAD, 5)
        out = tmp_path / "body.json"
        main(["restore", str(out)])
        assert not out.exists()

    @pytest.mark.usefixtures("env")
    def test_should_exit_with_an_error_without_a_review(
        self, gh: FakeGH, tmp_path: Path
    ) -> None:
        gh.runs = []
        with pytest.raises(SystemExit, match="::error::Sedosaur has no finished"):
            main(["restore", str(tmp_path / "body.json")])
