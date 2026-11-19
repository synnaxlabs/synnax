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

from review import Target, Unrestorable, render, restore
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
    back the body of the review of OLD. Reads come through the instance, and writes
    through `writer`."""

    def __init__(self) -> None:
        self.bodies = [f"{TEXT}\n\n{block(OLD, 3)}\n"]
        """The body at each read, the last one for every later read."""
        self.head = HEAD
        self.edits = [
            edit("someone", self.bodies[0]),
            edit("sedosaur", f"{TEXT}\n\n{block(HEAD, 5)}\n"),
            edit("sedosaur", self.bodies[0]),
        ]
        self.runs = [
            {
                "app": {"slug": "sedosaur"},
                "external_id": KEY,
                "started_at": "2026-11-18T05:00:00Z",
                "conclusion": "success",
            }
        ]
        self.writes: list[tuple[str, str, dict[str, Any]]] = []

    @property
    def body(self) -> str:
        return self.bodies[-1]

    @body.setter
    def body(self, body: str) -> None:
        self.bodies = [body]

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        assert args[:2] != ("api", "-X"), "a read token must not write"
        path = args[2] if args[1] == "--paginate" else args[1]
        if path == "graphql":
            return lines(self.edits)
        if path.startswith(f"repos/{REPO}/commits/{HEAD}/check-runs?"):
            return lines(self.runs)
        if path == f"repos/{REPO}/pulls/7":
            body = self.bodies.pop(0) if len(self.bodies) > 1 else self.body
            return json.dumps({"head": self.head, "body": body}) + "\n"
        raise AssertionError(f"unexpected gh call: {args}")

    def writer(self, *args: str, stdin: str | None = None) -> str:
        assert args[:2] == ("api", "-X")
        assert stdin is not None
        self.writes.append((args[2], args[3], json.loads(stdin)))
        return ""

    def written(self) -> str | None:
        """Returns the body that the restore wrote, or None when it wrote nothing."""
        if not self.writes:
            return None
        [(method, path, payload)] = self.writes
        assert (method, path) == ("PATCH", f"repos/{REPO}/pulls/7")
        body: str = payload["body"]
        return body


@pytest.fixture
def gh() -> FakeGH:
    return FakeGH()


def run(gh: FakeGH) -> bool:
    return restore(TARGET, gh, key=KEY, writer=gh.writer)


class TestRestore:
    """Tests putting the bot's last summary back in the pull request body."""

    def test_should_put_back_the_summary_the_bot_wrote_last(self, gh: FakeGH) -> None:
        assert run(gh)
        assert gh.written() == f"{TEXT}\n\n{block(HEAD, 5)}\n"

    def test_should_leave_the_rest_of_the_body_the_same(self, gh: FakeGH) -> None:
        gh.body = f"{TEXT}\n\n{block(OLD, 3)}\n\nA later note."
        run(gh)
        assert gh.written() == f"{TEXT}\n\n{block(HEAD, 5)}\n\nA later note."

    def test_should_append_the_summary_when_the_body_lost_it(self, gh: FakeGH) -> None:
        gh.body = TEXT
        run(gh)
        assert gh.written() == f"{TEXT}\n\n{block(HEAD, 5)}\n"

    def test_should_write_nothing_when_the_body_holds_it(self, gh: FakeGH) -> None:
        gh.body = f"{TEXT}\n\n{block(HEAD, 5)}"
        assert not run(gh)
        assert gh.written() is None

    def test_should_fail_when_the_head_has_no_review(self, gh: FakeGH) -> None:
        gh.runs = []
        with pytest.raises(Unrestorable, match=f"no finished review of {HEAD}"):
            run(gh)
        assert gh.written() is None

    def test_should_fail_when_the_review_of_the_head_failed(self, gh: FakeGH) -> None:
        gh.runs[0]["conclusion"] = "failure"
        with pytest.raises(Unrestorable, match="no finished review"):
            run(gh)

    def test_should_fail_when_the_bot_wrote_no_summary(self, gh: FakeGH) -> None:
        gh.edits = [edit("someone", gh.body)]
        with pytest.raises(Unrestorable, match="never wrote a summary"):
            run(gh)

    def test_should_not_write_over_a_body_written_during_it(self, gh: FakeGH) -> None:
        gh.bodies.append(f"{TEXT}\n\n{block('c' * 40, 4)}\n")
        with pytest.raises(Unrestorable, match="changed during the restore"):
            run(gh)
        assert gh.written() is None

    def test_should_fail_when_the_head_moved(self, gh: FakeGH) -> None:
        gh.head = "c" * 40
        with pytest.raises(Unrestorable, match="changed during the restore"):
            run(gh)
        assert gh.written() is None


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
            "WRITE_TOKEN": "app",
        }.items():
            monkeypatch.setenv(k, v)
        monkeypatch.delenv("CHECK_RUN", raising=False)
        monkeypatch.setattr("review.__main__.gh", gh)
        monkeypatch.setattr("review.__main__._writer", lambda _: gh.writer)

    @pytest.mark.usefixtures("env")
    def test_should_write_the_body(self, gh: FakeGH) -> None:
        main(["restore"])
        assert gh.written() == f"{TEXT}\n\n{block(HEAD, 5)}\n"

    @pytest.mark.usefixtures("env")
    def test_should_exit_with_an_error_without_a_review(self, gh: FakeGH) -> None:
        gh.runs = []
        with pytest.raises(SystemExit, match="::error::Sedosaur has no finished"):
            main(["restore"])
