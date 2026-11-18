#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import re
from typing import Any

import pytest

from review import (
    Dismissal,
    Target,
    Thread,
    authority,
    dismiss,
    dismissals,
    dismissed,
    render,
)
from review.__main__ import main
from review.dismiss import REFUSED

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
    id: int, body: str, *, kind: str = "User", login: str = "someone"
) -> dict[str, Any]:
    return {
        "databaseId": id,
        "author": {"__typename": kind, "login": login},
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
        text = render.RULES + render.LISTING_START + json.dumps(findings)
        text += render.LISTING_END
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
        self.comments: list[dict[str, Any]] = []
        """The comments on the pull request, as the REST API serves them."""
        self.permissions = {"someone": "write"}
        """The repository permission of each login. Any other login can read."""
        self.asked: list[str] = []
        """Each login whose permission was read, in order."""
        self.blobs: dict[str, str] = {}
        """The blob ID of each `<commit>:<path>`. Any other has none."""

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        if args[:2] == ("api", "-X"):
            assert stdin is not None
            self.writes.append((args[2], args[3], json.loads(stdin)))
            if (args[2], args[3]) == ("POST", f"repos/{REPO}/issues/7/comments"):
                answer = issue_comment(
                    100 + len(self.writes), json.loads(stdin)["body"]
                )
                bot = {"login": "sedosaur[bot]", "type": "Bot"}
                self.comments.append(answer | {"user": bot})
            return ""
        values = dict(a.split("=", 1) for a in args if "=" in a)
        if "e" in values:
            return self.blobs.get(values["e"], "null") + "\n"
        if "addPullRequestReviewThreadReply" in values.get("query", ""):
            self.replies.append((values["t"], values["b"]))
            reply = comment(0, values["b"], kind="Bot")
            reply["author"]["login"] = "sedosaur"
            for th in self.threads:
                if th["id"] == values["t"]:
                    th["comments"].append(reply)
            return "{}"
        if "resolveReviewThread" in values.get("query", ""):
            self.resolved.append(values["t"])
            return "{}"
        path = args[2] if args[1] == "--paginate" else args[1]
        if path == "graphql":
            return lines(self.threads)
        if m := re.fullmatch(f"repos/{REPO}/collaborators/(.+)/permission", path):
            self.asked.append(m[1])
            return self.permissions.get(m[1], "read") + "\n"
        if path.startswith(f"repos/{REPO}/commits/{HEAD}/check-runs?"):
            return lines(self.runs)
        if path == f"repos/{REPO}/issues/7/comments":
            return lines([shaped(c) for c in self.comments])
        for c in self.comments:
            if path == f"repos/{REPO}/issues/comments/{c['id']}":
                on = c["issue_url"].endswith("/issues/7")
                return json.dumps(shaped(c)) + "\n" if on else ""
        raise AssertionError(f"unexpected gh call: {args}")

    def posted(self) -> list[str]:
        """The bodies of the comments the bot posted on the pull request."""
        return [
            p["body"]
            for m, at, p in self.writes
            if (m, at) == ("POST", f"repos/{REPO}/issues/7/comments")
        ]

    def checks(self) -> list[dict[str, Any]]:
        return [
            p
            for m, at, p in self.writes
            if (m, at) == ("POST", f"repos/{REPO}/check-runs")
        ]


def shaped(c: dict[str, Any]) -> dict[str, Any]:
    """A REST comment in the shape that the poster's `jq` makes."""
    user = c["user"]
    return {
        "id": c["id"],
        "login": user["login"],
        "author": {"__typename": user["type"], "login": user["login"]},
        "body": c["body"],
    }


@pytest.fixture
def gh() -> FakeGH:
    return FakeGH()


def issue_comment(
    id: int, body: str, *, kind: str = "User", login: str = "someone", pr: int = 7
) -> dict[str, Any]:
    return {
        "id": id,
        "issue_url": f"https://api.github.com/repos/{REPO}/issues/{pr}",
        "user": {"login": login, "type": kind},
        "body": body,
    }


def outside(*titles: str, path: str = "b.go") -> dict[str, Any]:
    """The bot's comment that lists findings outside the diff of a file."""
    items = "".join(
        f"- <img> **{t}** [`{path}:3`](https://x)\n\n  B.\n\n" for t in titles
    )
    body = f"{render.OUTSIDE}\n\n<h3>Comments Outside Diff</h3>\n\n{items}"
    bot = {"login": "sedosaur[bot]", "type": "Bot"}
    return issue_comment(5, body) | {"user": bot}


class TestDismissed:
    """Tests which replies dismiss a finding."""

    def test_should_take_a_writer_dismissal_with_its_reason(self, gh: FakeGH) -> None:
        th = Thread(
            id="T",
            path="a.go",
            title="Samples",
            body="<picture></picture> **Samples**\n\nBody.",
            comment=1,
            resolved=True,
            replies=[comment(2, "Meant. @Sedosaur  Dismiss")],
        )
        assert dismissed([th], authority(TARGET, gh)) == [
            Dismissal(path="a.go", title="Samples", reason="Meant.")
        ]

    @pytest.mark.parametrize(
        "reply",
        [
            comment(2, "@sedosaur dismiss", kind="Bot"),
            comment(2, "@sedosaur dismiss", login="reader"),
            comment(2, "@sedosaur dismissal is wrong"),
            comment(2, "> reply `@sedosaur dismiss` and a reason\n\nNo."),
            comment(2, "Please dismiss this."),
            {**comment(2, "@sedosaur dismiss"), "author": None},
        ],
    )
    def test_should_ignore_any_other_reply(
        self, gh: FakeGH, reply: dict[str, Any]
    ) -> None:
        th = Thread(
            id="T",
            path="a.go",
            title="Samples",
            body="<picture></picture> **Samples**\n\nBody.",
            comment=1,
            resolved=False,
            replies=[reply],
        )
        assert dismissed([th], authority(TARGET, gh)) == []

    def test_should_ask_for_the_permission_of_a_login_once(self, gh: FakeGH) -> None:
        th = Thread(
            id="T",
            path="a.go",
            title="Samples",
            body="<picture></picture> **Samples**\n\nBody.",
            comment=1,
            resolved=False,
            replies=[comment(2, "Hi."), comment(3, "@sedosaur dismiss")],
        )
        can = authority(TARGET, gh)
        dismissed([th, th], can)
        assert gh.asked == ["someone"]

    @pytest.mark.parametrize("permission", ["write", "admin"])
    def test_should_take_a_private_member_with_write_access(
        self, gh: FakeGH, permission: str
    ) -> None:
        gh.permissions = {"private": permission}
        assert authority(TARGET, gh)(comment(2, "", login="private"))


class TestDismiss:
    """Tests answering a reply on a thread of a finding of the bot."""

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
                "summary": "Every finding of the review is dismissed or withdrawn.\n\n"
                "Fix one thing.\n\nFindings: 1 P2.",
                "text": gh.runs[0]["output"]["text"],
            },
        }
        assert [t for t, _ in gh.replies] == ["T1", "T2"]
        assert gh.posted() == [
            "Every finding of the review of the head is dismissed or withdrawn, so the "
            "head scores 5/5."
        ]

    def test_should_answer_a_reply_whose_run_never_answered(self, gh: FakeGH) -> None:
        gh.threads = [
            thread("T1", comment(2, "@sedosaur dismiss")),
            thread(
                "T2",
                comment(3, "@sedosaur why?", login="reader"),
                path="b.go",
                title="Nit",
            ),
        ]
        dismiss(TARGET, gh, comment=3, key=KEY)
        assert gh.resolved == ["T1"]
        [(t1, done), (t2, usage)] = sorted(gh.replies)
        assert (t1, t2) == ("T1", "T2")
        assert done.startswith("Dismissed.")
        assert usage.startswith("To dismiss this finding")

    def test_should_score_the_head_5_after_a_run_that_did_not(self, gh: FakeGH) -> None:
        bot = comment(4, "Dismissed.", kind="Bot")
        bot["author"]["login"] = "sedosaur"
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss"), bot)]
        gh.runs = [check_run([["a.go", "Samples"]])]
        dismiss(TARGET, gh, comment=2, key=KEY)
        [check] = gh.checks()
        assert check["output"]["title"] == "Confidence Score: 5/5"
        assert gh.replies == []

    def test_should_post_nothing_when_run_again_on_the_same_state(
        self, gh: FakeGH
    ) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss"))]
        gh.comments = [outside("Old"), issue_comment(9, "@sedosaur dismiss Old")]
        dismiss(TARGET, gh, comment=9, key=KEY)
        writes, replies = len(gh.writes), len(gh.replies)
        dismiss(TARGET, gh, comment=9, key=KEY)
        assert (len(gh.writes), len(gh.replies)) == (writes, replies)

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

    def test_should_not_rescore_a_review_under_older_rules(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss"))]
        run = check_run([["a.go", "Samples"]])
        run["output"]["text"] = run["output"]["text"].removeprefix(render.RULES)
        gh.runs = [run]
        dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.checks() == []

    def test_should_refuse_an_agent(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss", kind="Bot"))]
        gh.runs = [check_run([["a.go", "Samples"]])]
        dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.replies == [("T1", REFUSED)]
        assert gh.resolved == []
        assert gh.checks() == []

    def test_should_refuse_a_person_who_can_only_read(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss", login="reader"))]
        gh.runs = [check_run([["a.go", "Samples"]])]
        dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.replies == [("T1", REFUSED)]
        assert gh.resolved == []
        assert gh.checks() == []

    def test_should_ask_for_the_permission_of_each_login_once(self, gh: FakeGH) -> None:
        gh.threads = [
            thread("T1", comment(2, "@sedosaur dismiss")),
            thread("T2", comment(3, "@sedosaur dismiss"), path="b.go", title="Nit"),
        ]
        gh.comments = [outside("Old"), issue_comment(9, "@sedosaur dismiss Old")]
        gh.runs = [check_run([["a.go", "Samples"], ["b.go", "Nit"], ["b.go", "Old"]])]
        dismiss(TARGET, gh, comment=9, key=KEY)
        assert gh.checks()[0]["output"]["title"] == "Confidence Score: 5/5"
        assert gh.asked == ["someone"]

    def test_should_answer_another_mention_with_the_usage(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur why?", login="reader"))]
        dismiss(TARGET, gh, comment=2, key=KEY)
        [(_, body)] = gh.replies
        assert body.startswith("To dismiss this finding, reply `@sedosaur dismiss`")
        assert gh.resolved == []

    def test_should_answer_a_reply_with_no_mention_with_the_usage(
        self, gh: FakeGH
    ) -> None:
        gh.threads = [thread("T1", comment(2, "Fixed in abc123."))]
        gh.runs = [check_run([["a.go", "Samples"]])]
        dismiss(TARGET, gh, comment=2, key=KEY)
        [(_, body)] = gh.replies
        assert body.startswith("To dismiss this finding, reply `@sedosaur dismiss`")
        assert gh.resolved == []
        assert gh.checks() == []

    def test_should_answer_each_thread_with_the_usage_once(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "Fixed in abc123."))]
        dismiss(TARGET, gh, comment=2, key=KEY)
        gh.threads[0]["comments"].append(comment(3, "See the test."))
        dismiss(TARGET, gh, comment=3, key=KEY)
        assert len(gh.replies) == 1

    def test_should_answer_a_thread_whose_run_never_answered_with_the_usage(
        self, gh: FakeGH
    ) -> None:
        gh.threads = [
            thread("T1", comment(2, "Fixed in abc123.")),
            thread("T2", comment(3, "Fixed by abc123."), path="b.go", title="Nit"),
        ]
        dismiss(TARGET, gh, comment=3, key=KEY)
        assert [t for t, _ in gh.replies] == ["T1", "T2"]

    def test_should_leave_a_rebuttal_to_the_judge(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "Wrong: `a.go:3` checks it."))]
        assert dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.replies == []

    def test_should_leave_a_rebuttal_after_a_fix_note_to_the_judge(
        self, gh: FakeGH
    ) -> None:
        gh.threads = [thread("T1", comment(2, "Fixed in abc123."))]
        assert not dismiss(TARGET, gh, comment=2, key=KEY)
        gh.threads[0]["comments"].append(comment(3, "Also, `a.go:3` checks it."))
        assert dismiss(TARGET, gh, comment=3, key=KEY)
        assert len(gh.replies) == 1

    def test_should_leave_an_agent_rebuttal_to_the_judge(self, gh: FakeGH) -> None:
        body = "*Agent reply.* The type is `never`."
        gh.threads = [
            thread("T1", comment(2, body, kind="Bot", login="synnax-claude-code"))
        ]
        assert dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.replies == []
        assert gh.asked == []

    def test_should_not_judge_a_dismissed_finding(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "It is meant. @sedosaur dismiss"))]
        assert not dismiss(TARGET, gh, comment=2, key=KEY)

    def test_should_not_answer_a_resolved_thread_with_the_usage(
        self, gh: FakeGH
    ) -> None:
        gh.threads = [thread("T1", comment(2, "Thanks."))]
        gh.threads[0]["resolved"] = True
        dismiss(TARGET, gh, comment=2, key=KEY)
        assert gh.replies == []

    def test_should_fail_on_a_comment_of_another_pull_request(self, gh: FakeGH) -> None:
        gh.comments = [issue_comment(9, "@sedosaur dismiss Old", pr=8)]
        with pytest.raises(ValueError, match="comment 9 is not on pull request 7"):
            dismiss(TARGET, gh, comment=9, key=KEY)
        assert gh.posted() == []


class TestDismissComment:
    """Tests answering the dismissal comments on the pull request."""

    def test_should_bind_a_dismissal_to_the_path_of_its_finding(
        self, gh: FakeGH
    ) -> None:
        gh.comments = [
            outside("Old", "Old data"),
            issue_comment(9, "@sedosaur dismiss Old data: it is meant"),
        ]
        dismiss(TARGET, gh, comment=9, key=KEY)
        assert gh.posted()[0].endswith(
            "Dismissed. Later reviews of this pull request will not report it."
        )
        gh.comments[0] = outside("Old data", path="c.go")
        assert dismissals(TARGET, gh, [], authority(TARGET, gh)) == [
            Dismissal(path="b.go", title="Old data", reason="it is meant")
        ]
        assert gh.checks() == []

    def test_should_keep_a_dismissal_whose_finding_left_the_comment(
        self, gh: FakeGH
    ) -> None:
        gh.comments = [outside("Old"), issue_comment(9, "@sedosaur dismiss Old")]
        dismiss(TARGET, gh, comment=9, key=KEY)
        gh.comments[0] = outside()
        assert dismissals(TARGET, gh, [], authority(TARGET, gh)) == [
            Dismissal(path="b.go", title="Old", reason="")
        ]

    def test_should_dismiss_and_resolve_a_finding_on_a_thread(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1")]
        gh.comments = [outside(), issue_comment(9, "@sedosaur dismiss samples, meant")]
        dismiss(TARGET, gh, comment=9, key=KEY)
        assert gh.resolved == ["T1"]
        assert dismissals(TARGET, gh, [], authority(TARGET, gh)) == [
            Dismissal(path="a.go", title="Samples", reason="meant")
        ]

    def test_should_score_the_head_5_when_no_finding_is_left(self, gh: FakeGH) -> None:
        gh.threads = [thread("T1", comment(2, "@sedosaur dismiss"))]
        gh.comments = [outside("Old"), issue_comment(9, "@sedosaur dismiss Old")]
        gh.runs = [check_run([["a.go", "Samples"], ["b.go", "Old"]])]
        dismiss(TARGET, gh, comment=9, key=KEY)
        [check] = gh.checks()
        assert check["output"]["title"] == "Confidence Score: 5/5"
        assert gh.posted()[-1] == (
            "Every finding of the review of the head is dismissed or withdrawn, so the "
            "head scores 5/5."
        )

    def test_should_answer_a_comment_whose_run_never_answered(self, gh: FakeGH) -> None:
        gh.comments = [
            outside("Old", "New"),
            issue_comment(9, "@sedosaur dismiss Old"),
            issue_comment(10, "@sedosaur dismiss New"),
        ]
        gh.runs = [check_run([["b.go", "Old"], ["b.go", "New"]])]
        dismiss(TARGET, gh, comment=10, key=KEY)
        assert len(gh.posted()) == 3
        [check] = gh.checks()
        assert check["output"]["title"] == "Confidence Score: 5/5"

    def test_should_answer_each_comment_once(self, gh: FakeGH) -> None:
        gh.comments = [outside("Old"), issue_comment(9, "@sedosaur dismiss Old")]
        dismiss(TARGET, gh, comment=9, key=KEY)
        dismiss(TARGET, gh, comment=9, key=KEY)
        assert len(gh.posted()) == 1

    def test_should_refuse_an_agent(self, gh: FakeGH) -> None:
        gh.comments = [
            outside("Old"),
            issue_comment(9, "@sedosaur dismiss Old", kind="Bot"),
        ]
        gh.runs = [check_run([["b.go", "Old"]])]
        dismiss(TARGET, gh, comment=9, key=KEY)
        [answer] = gh.posted()
        assert answer.endswith(REFUSED)
        assert dismissals(TARGET, gh, [], authority(TARGET, gh)) == []
        assert gh.checks() == []

    def test_should_refuse_a_person_who_can_only_read(self, gh: FakeGH) -> None:
        gh.comments = [
            outside("Old"),
            issue_comment(9, "@sedosaur dismiss Old", login="reader"),
        ]
        dismiss(TARGET, gh, comment=9, key=KEY)
        [answer] = gh.posted()
        assert answer.endswith(REFUSED)
        assert dismissals(TARGET, gh, [], authority(TARGET, gh)) == []

    @pytest.mark.parametrize(
        "body",
        [
            "@sedosaur dismiss Other",
            "@sedosaur dismiss Old",
            "@sedosaur dismiss Old path:d.go",
        ],
    )
    def test_should_answer_a_comment_naming_no_one_finding_with_the_usage(
        self, gh: FakeGH, body: str
    ) -> None:
        both = outside("Old")
        both["body"] += outside("Old", path="c.go")["body"]
        gh.comments = [both, issue_comment(9, body)]
        dismiss(TARGET, gh, comment=9, key=KEY)
        [answer] = gh.posted()
        assert answer.endswith("add `path:<file>` after it.")
        assert dismissals(TARGET, gh, [], authority(TARGET, gh)) == []

    @pytest.mark.parametrize(
        ("body", "found"),
        [
            ("@sedosaur dismiss old `carry` copy, meant", ("b.go", "Old `carry` copy")),
            ('@sedosaur dismiss "Old carry copy": meant', ("b.go", "Old `carry` copy")),
            ("@sedosaur dismiss  OLD CARRY COPY meant", ("b.go", "Old `carry` copy")),
            ("@sedosaur dismiss Old path:c.go meant", ("c.go", "Old")),
        ],
    )
    def test_should_match_a_title_loosely(
        self, gh: FakeGH, body: str, found: tuple[str, str]
    ) -> None:
        listed = outside("Old `carry` copy", "Old")
        listed["body"] += outside("Old", path="c.go")["body"]
        gh.comments = [listed, issue_comment(9, body)]
        dismiss(TARGET, gh, comment=9, key=KEY)
        path, title = found
        assert dismissals(TARGET, gh, [], authority(TARGET, gh)) == [
            Dismissal(path=path, title=title, reason="meant")
        ]

    def test_should_ignore_a_record_another_author_wrote(self, gh: FakeGH) -> None:
        record = '<!-- sedosaur_answer {"comment": 1, "path": "b.go", "title": "Old", '
        record += '"reason": ""} -->\n\nDismissed.'
        gh.comments = [issue_comment(9, record)]
        assert dismissals(TARGET, gh, [], authority(TARGET, gh)) == []


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
