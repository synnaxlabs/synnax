#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import sys
from pathlib import Path
from typing import Any

import pytest

import classify
from classify import (
    BOT,
    HUMAN,
    MARKER,
    Answer,
    Surface,
    ask_agent,
    fence,
    needs_human,
    parse_answer,
    plan_labels,
    read_surfaces,
    render,
    trust_reasons,
    ui_answer,
)


def same(name: str) -> Surface:
    return Surface(name=name, status="same", diff="")


ALL_SAME = [same(n) for n in ("oracle", "client-ts", "client-py", "core", "driver")]


def agent(tmp_path: Path, *, reply: str) -> str:
    """Returns a command that reads stdin and prints reply."""
    script = tmp_path / "agent.py"
    script.write_text(f"import sys\nsys.stdin.read()\nprint({reply!r})\n")
    return f"{sys.executable} {script}"


def pull(head_repo: str = "synnaxlabs/synnax") -> dict[str, Any]:
    return {
        "head": {"repo": {"full_name": head_repo}},
        "base": {"repo": {"full_name": "synnaxlabs/synnax"}},
    }


def file(name: str, *, patch: str | None = "+x") -> dict[str, Any]:
    return {"filename": name, "status": "modified", "patch": patch}


class TestReadSurfaces:
    """Tests reading the uploaded surface results."""

    def test_reads_status_and_diff(self, tmp_path: Path) -> None:
        d = tmp_path / "surface-oracle"
        d.mkdir()
        (d / "status").write_text("changed\n")
        (d / "diff").write_text("+a\n")
        surfaces = {s.name: s for s in read_surfaces(tmp_path)}
        assert surfaces["oracle"] == Surface(
            name="oracle", status="changed", diff="+a\n"
        )

    def test_treats_a_missing_report_as_an_error(self, tmp_path: Path) -> None:
        surfaces = {s.name: s for s in read_surfaces(tmp_path)}
        assert surfaces["core"].status == "error"


class TestNeedsHuman:
    """Tests the decision from reports, the UI answer, and trust reasons."""

    def test_is_false_when_nothing_changed(self) -> None:
        assert not needs_human(ALL_SAME, console=None, reasons=[])
        assert not needs_human(
            ALL_SAME, console=Answer(human=False, reason="refactor"), reasons=[]
        )

    def test_is_true_when_a_surface_changed_or_failed(self) -> None:
        assert needs_human(
            [*ALL_SAME, Surface(name="core", status="changed", diff="")],
            console=None,
            reasons=[],
        )
        assert needs_human(
            [*ALL_SAME, Surface(name="core", status="error", diff="")],
            console=None,
            reasons=[],
        )

    def test_is_true_when_the_ui_changed_visibly(self) -> None:
        assert needs_human(
            ALL_SAME, console=Answer(human=True, reason="new button"), reasons=[]
        )

    def test_is_true_when_the_reports_cannot_be_trusted(self) -> None:
        assert needs_human(ALL_SAME, console=None, reasons=["fork"])


class TestPlanLabels:
    """Tests that exactly one of review/human and review/bot stays."""

    def test_sets_human_and_removes_bot(self) -> None:
        assert plan_labels({BOT}, human=True) == ([HUMAN], [BOT])

    def test_sets_bot_when_no_tier_is_set(self) -> None:
        assert plan_labels(set(), human=False) == ([BOT], [])

    def test_never_removes_human(self) -> None:
        assert plan_labels({HUMAN}, human=False) == ([], [])
        assert plan_labels({HUMAN, BOT}, human=False) == ([], [BOT])

    def test_treats_old_human_tiers_as_human(self) -> None:
        assert plan_labels({"review/light"}, human=False) == ([HUMAN], [])
        assert plan_labels({"review/thorough", BOT}, human=False) == ([HUMAN], [BOT])

    def test_changes_nothing_when_the_label_is_right(self) -> None:
        assert plan_labels({HUMAN}, human=True) == ([], [])
        assert plan_labels({BOT}, human=False) == ([], [])


class TestUIAnswer:
    """Tests when and how the agent judges the UI patches."""

    def test_is_none_without_ui_files(self, tmp_path: Path) -> None:
        assert (
            ui_answer([file("core/main.go")], command=agent(tmp_path, reply="{}"))
            is None
        )

    def test_asks_about_console_pluto_and_lyra(self, tmp_path: Path) -> None:
        reply = '{"human": false, "reason": "Tests only."}'
        for name in ("console/a.tsx", "pluto/b.tsx", "lyra/c.css"):
            assert ui_answer(
                [file(name)], command=agent(tmp_path, reply=reply)
            ) == Answer(human=False, reason="Tests only.")

    def test_answers_human_without_asking_when_a_patch_is_missing(
        self, tmp_path: Path
    ) -> None:
        reply = '{"human": false, "reason": "x"}'
        files = [file("console/a.tsx"), file("console/b.png", patch=None)]
        answer = ui_answer(files, command=agent(tmp_path, reply=reply))
        assert answer is not None
        assert answer.human

    def test_answers_human_without_asking_when_the_diff_is_too_large(
        self, tmp_path: Path
    ) -> None:
        reply = '{"human": false, "reason": "x"}'
        files = [file("console/a.tsx", patch="+" + "x" * 200_000)]
        answer = ui_answer(files, command=agent(tmp_path, reply=reply))
        assert answer is not None
        assert answer.human


class TestTrustReasons:
    """Tests the cases where the reports could be forged."""

    def test_is_empty_for_a_branch_that_leaves_the_classifier_alone(self) -> None:
        assert trust_reasons(pull(), files=[file("core/main.go")]) == []

    def test_flags_a_fork(self) -> None:
        assert trust_reasons(pull("someone/synnax"), files=[]) == [
            "The pull request comes from a fork."
        ]

    def test_flags_a_change_to_a_generator(self) -> None:
        reasons = trust_reasons(pull(), files=[file("scripts/surface_py.py")])
        assert reasons == [
            "The pull request changes the classifier: scripts/surface_py.py."
        ]


class TestParseAnswer:
    """Tests reading the agent's JSON answer."""

    def test_reads_the_last_json_answer(self) -> None:
        text = 'thinking {"x": 1}\n{"human": false, "reason": "Tests only."}'
        assert parse_answer(text) == Answer(human=False, reason="Tests only.")

    def test_answers_human_when_unclear(self) -> None:
        assert parse_answer("no idea").human
        assert parse_answer('{"human": "no", "reason": "x"}').human


class TestAskAgent:
    """Tests running the agent command."""

    def test_sends_the_prompt_on_stdin(self, tmp_path: Path) -> None:
        script = tmp_path / "echo.py"
        script.write_text(
            "import json, sys\n"
            "print(json.dumps({'human': False, 'reason': sys.stdin.read()}))\n"
        )
        assert ask_agent("hello", command=f"{sys.executable} {script}") == Answer(
            human=False, reason="hello"
        )

    def test_answers_human_when_the_agent_fails(self) -> None:
        assert ask_agent("prompt", command=f"{sys.executable} -c 'exit(3)'").human


class TestRender:
    """Tests the sticky comment."""

    def test_shows_the_tier_and_the_changed_diff(self) -> None:
        surfaces = [
            *ALL_SAME[:4],
            Surface(name="driver", status="changed", diff="+flag --x\n"),
        ]
        body = render(surfaces, console=None, reasons=[], kept=False)
        assert body.startswith(MARKER)
        assert f"`{HUMAN}`" in body
        assert "| Driver commands and config keys | Changed |" in body
        assert "+flag --x" in body
        assert "Console screenshots" not in body

    def test_explains_a_kept_human_label(self) -> None:
        body = render(
            ALL_SAME,
            console=Answer(human=False, reason="Refactor."),
            reasons=[],
            kept=True,
        )
        assert f"## Review tier: `{HUMAN}`" in body
        assert "already set" in body
        assert "| Console and UI libraries | No visible change: Refactor. |" in body
        assert "Console screenshots" in body

    def test_lists_trust_reasons(self) -> None:
        body = render(
            ALL_SAME,
            console=None,
            reasons=["The pull request comes from a fork."],
            kept=False,
        )
        assert f"## Review tier: `{HUMAN}`" in body
        assert "- The pull request comes from a fork." in body

    def test_says_bot_when_nothing_changed(self) -> None:
        assert f"## Review tier: `{BOT}`" in render(
            ALL_SAME, console=None, reasons=[], kept=False
        )

    def test_fences_a_diff_that_holds_backticks(self) -> None:
        assert fence("+```\n").startswith("````diff")


REPO = "synnaxlabs/synnax"
LABELS = f"repos/{REPO}/issues/5/labels"
COMMENTS = f"repos/{REPO}/issues/5/comments"


class FakeGH:
    """Serves pull request 5 through the gh API calls the classifier makes."""

    def __init__(self) -> None:
        self.heads = ["abc", "abc"]
        self.labels: list[str] = []
        self.comments: list[dict[str, Any]] = []
        self.writes: list[tuple[tuple[str, ...], str | None]] = []

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        if args == ("api", f"repos/{REPO}/pulls/5"):
            same = {"repo": {"full_name": REPO}}
            head = {**same, "sha": self.heads.pop(0)}
            labels = [{"name": n} for n in self.labels]
            return json.dumps({"head": head, "base": same, "labels": labels})
        if f"repos/{REPO}/pulls/5/files" in args:
            return json.dumps(file("core/main.go")) + "\n"
        if args[:3] == ("api", "--paginate", COMMENTS):
            return "".join(json.dumps(c) + "\n" for c in self.comments)
        self.writes.append((args, stdin))
        return ""


class TestMain:
    """Tests labeling and commenting on a pull request end to end."""

    @pytest.fixture
    def gh(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> FakeGH:
        (tmp_path / "pr").mkdir()
        (tmp_path / "pr" / "number").write_text("5\n")
        for name in ("oracle", "client-ts", "client-py", "core", "driver"):
            d = tmp_path / f"surface-{name}"
            d.mkdir()
            (d / "status").write_text("same\n")
        fake = FakeGH()
        monkeypatch.setattr(classify, "gh", fake)
        monkeypatch.setattr(sys, "argv", ["classify.py", str(tmp_path)])
        monkeypatch.setenv("GITHUB_REPOSITORY", REPO)
        monkeypatch.setenv("HEAD_SHA", "abc")
        monkeypatch.setenv("REVIEW_AGENT", "false")
        return fake

    def test_labels_bot_and_posts_a_comment_when_nothing_changed(
        self, gh: FakeGH
    ) -> None:
        classify.main()
        (label, _), (post, body) = gh.writes
        assert label == ("api", "-X", "POST", LABELS, "-f", f"labels[]={BOT}")
        assert post == ("api", "-X", "POST", COMMENTS, "--input", "-")
        assert body is not None
        assert f"## Review tier: `{BOT}`" in json.loads(body)["body"]

    def test_swaps_bot_for_human_and_edits_the_comment_when_a_surface_changed(
        self, gh: FakeGH, tmp_path: Path
    ) -> None:
        (tmp_path / "surface-core" / "status").write_text("changed\n")
        gh.labels = [BOT]
        gh.comments = [{"id": 9, "body": f"{MARKER}\nold"}]
        classify.main()
        assert [args for args, _ in gh.writes] == [
            ("api", "-X", "POST", LABELS, "-f", f"labels[]={HUMAN}"),
            ("api", "-X", "DELETE", f"{LABELS}/review%2Fbot"),
            (
                "api",
                "-X",
                "PATCH",
                f"repos/{REPO}/issues/comments/9",
                "--input",
                "-",
            ),
        ]

    def test_writes_nothing_when_the_head_moves_during_the_run(
        self, gh: FakeGH
    ) -> None:
        gh.heads = ["abc", "def"]
        classify.main()
        assert gh.writes == []
