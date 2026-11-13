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

from review import Dismissal, Target, Thread, dismiss, dismissed, render
from review.__main__ import main

REPO = "synnaxlabs/synnax"
HEAD = "b" * 40
KEY = "7:main"
TARGET = Target(
    repo=REPO,
    pr=7,
    head=HEAD,
    base="base",
    check_run=None,
    run_url="https://github.com/synnaxlabs/synnax/actions/runs/3",
    bot="sedosaur",
)
FINDING = "<picture></picture> **Samples return old data**\n\nOld text."


def lines(items: list[dict[str, Any]]) -> str:
    return "".join(json.dumps(i) + "\n" for i in items)


def comment(
    id: int, body: str, *, kind: str = "User", association: str = "MEMBER"
) -> dict[str, Any]:
    return {
        "databaseId": id,
        "author": {"__typename": kind, "login": "someone"},
        "authorAssociation": association,
        "body": body,
    }


def thread(
    id: str, *replies: dict[str, Any], path: str = "a.go", title: str = "Samples"
) -> dict[str, Any]:
    first = comment(1, f"<picture></picture> **{title}**\n\nText.", kind="Bot")
    first["author"]["login"] = "sedosaur"
    return {"id": id, "path": path, "resolved": False, "comments": [first, *replies]}


def check_run(findings: list[list[str]] | None, **kw: Any) -> dict[str, Any]:
    text = None
    if findings is not None:
        text = render.LISTING_START + json.dumps(findings) + render.LISTING_END
    run: dict[str, Any] = {
        "name": "Sedosaur Review",
        "app": {"slug": "sedosaur"},
        "external_id": KEY,
        "started_at": "2026-11-08T05:00:00Z",
        "conclusion": "success",
        "output": {
            "title": "Confidence Score: 4/5",
            "summary": "Fix one thing.\n\nFindings: 1 P2.",
            "text": text,
        },
    }
    return run | kw


class FakeGH:
    """Serves the threads and the check runs of pull request 7, and records each write
    and each mutation."""

    def __init__(self) -> None:
        self.threads: list[dict[str, Any]] = []
        self.runs: list[dict[str, Any]] = []
        self.writes: list[tuple[str, str, dict[str, Any]]] = []
        self.replies: list[tuple[str, str]] = []
        self.resolved: list[str] = []

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        if args[:2] == ("api", "-X"):
            assert stdin is not None
            self.writes.append((args[2], args[3], json.loads(stdin)))
            return ""
        values = dict(a.split("=", 1) for a in args if "=" in a)
        if "addPullRequestReviewThreadReply" in values.get("query", ""):
            self.replies.append((values["t"], values["b"]))
            return "{}"
        if "resolveReviewThread" in values.get("query", ""):
            self.resolved.append(values["t"])
            return "{}"
        path = args[2] if args[1] == "--paginate" else args[1]
        if path == "graphql":
            return lines(self.threads)
        if path.startswith(f"repos/{REPO}/commits/{HEAD}/check-runs?"):
            return lines(self.runs)
        raise AssertionError(f"unexpected gh call: {args}")

    def checks(self) -> list[dict[str, Any]]:
        return [
            p
            for m, at, p in self.writes
            if (m, at) == ("POST", f"repos/{REPO}/check-runs")
        ]


@pytest.fixture
def gh() -> FakeGH:
    return FakeGH()


class TestDismissed:
    """Tests which replies dismiss a finding."""

    def test_should_take_a_member_dismissal_with_its_reason(self) -> None:
        th = Thread(
            id="T",
            path="a.go",
            title="Samples",
            heading="<picture></picture> **Samples**",
            comment=1,
            resolved=True,
            replies=[comment(2, "Meant. @Sedosaur  Dismiss")],
        )
        assert dismissed([th]) == [
            Dismissal(path="a.go", title="Samples", reason="Meant.")
        ]

    @pytest.mark.parametrize(
        "reply",
        [
            comment(2, "@sedosaur dismiss", kind="Bot"),
            comment(2, "@sedosaur dismiss", association="CONTRIBUTOR"),
            comment(2, "@sedosaur dismissal is wrong"),
            comment(2, "> reply `@sedosaur dismiss` and a reason\n\nNo."),
            comment(2, "Please dismiss this."),
            {**comment(2, "@sedosaur dismiss"), "author": None},
        ],
    )
    def test_should_ignore_any_other_reply(self, reply: dict[str, Any]) -> None:
        th = Thread(
            id="T",
            path="a.go",
            title="Samples",
            heading="<picture></picture> **Samples**",
            comment=1,
            resolved=False,
            replies=[reply],
        )
        assert dismissed([th]) == []


class TestDismiss:
    """Tests answering a reply that mentions the bot on a thread of its finding."""

    def test_should_resolve_the_thread_of_a_dismissed_finding(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss, it is meant"))]
        dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.replies == [
            ("T1", "Dismissed. Later reviews of this pull request will not report it.")
        ]
        assert gh.resolved == ["T1"]
        assert gh.checks() == []

    def test_should_score_the_head_5_when_no_finding_is_left(self, gh: FakeGH) -> None:
        gh.threads = [
            thread("T1", comment(2, "@sedosaur dismiss")),
            thread("T2", comment(3, "@sedosaur dismiss"), path="b.go", title="Nit"),
        ]
        gh.runs = [check_run([["a.go", "Samples"], ["b.go", "Nit"]])]
        dismiss(TARGET, gh, comment=3, key=KEY)
        [check] = gh.checks()
        assert check == {
            "name": "Sedosaur Review",
            "head_sha": HEAD,
            "external_id": KEY,
            "details_url": TARGET.run_url,
            "status": "completed",
            "conclusion": "success",
            "output": {
                "title": "Confidence Score: 5/5",
                "summary": "A member dismissed every finding of the review that blocks "
                "merge.\n\nFix one thing.\n\nFindings: 1 P2.",
                "text": gh.runs[0]["output"]["text"],
            },
        }
        [(_, body)] = gh.replies
        assert body.endswith(
            "will not report it. Every blocking finding of the review of the head is "
            "dismissed, so the head scores 5/5."
        )

    def test_should_take_the_newest_review(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss"))]
        gh.runs = [
            check_run([["a.go", "Samples"]]),
            check_run([["b.go", "Nit"]], started_at="2026-11-08T06:00:00Z"),
        ]
        dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.checks() == []

    @pytest.mark.parametrize(
        "run",
        [
            check_run([["a.go", "Samples"], ["b.go", "Nit"]]),
            check_run(None),
            check_run([]),
            check_run([["a.go", "Samples"]], conclusion="failure"),
            check_run([["a.go", "Samples"]], external_id="7:rc"),
            check_run([["a.go", "Samples"]], app={"slug": "other"}),
            check_run(
                [["a.go", "Samples"]],
                output={"title": "Confidence Score: 5/5", "summary": "S", "text": None},
            ),
        ],
    )
    def test_should_keep_the_score_otherwise(
        self, gh: FakeGH, run: dict[str, Any]
    ) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss"))]
        gh.runs = [run]
        dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.checks() == []
        assert gh.resolved == ["T1"]

    def test_should_refuse_an_agent(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss", kind="Bot"))]
        gh.runs = [check_run([["a.go", "Samples"]])]
        dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.replies == [
            ("T1", "Only a member can dismiss a finding, not an agent.")
        ]
        assert gh.resolved == []
        assert gh.checks() == []

    def test_should_answer_another_mention_with_the_usage(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur why?"))]
        dismiss(TARGET, gh, comment=2, key=KEY)
        [(_, body)] = gh.replies
        assert body.startswith("To dismiss this finding, reply `@sedosaur dismiss`")
        assert gh.resolved == []

    def test_should_fail_on_a_comment_off_the_threads_of_the_bot(
        self, gh: FakeGH
    ) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss"))]
        with pytest.raises(ValueError, match="comment 9 is not a reply to a finding"):
            dismiss(TARGET, gh, comment=9, key=KEY)
        assert gh.replies == []


class TestCommand:
    """Tests the `dismiss` command of the review CLI."""

    def test_should_answer_the_comment(
        self, gh: FakeGH, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setattr("review.__main__.gh", gh)
        env = {
            "GITHUB_REPOSITORY": REPO,
            "PR": "7",
            "HEAD_SHA": HEAD,
            "BASE_SHA": "base",
            "RUN_URL": TARGET.run_url,
            "BOT": "sedosaur",
            "COMMENT": "2",
            "KEY": KEY,
        }
        for k, v in env.items():
            monkeypatch.setenv(k, v)
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss"))]
        main(["dismiss"])
        assert gh.resolved == ["T1"]
