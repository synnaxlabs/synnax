#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
from pathlib import Path
from typing import Any

from classify import (
    BOT,
    HUMAN,
    MARKER,
    Answer,
    Surface,
    ask_agent,
    console_prompt,
    fence,
    needs_human,
    parse_answer,
    plan_labels,
    read_surfaces,
    render,
)


def same(name: str) -> Surface:
    return Surface(name, "same", "")


ALL_SAME = [same(n) for n in ("oracle", "client-ts", "client-py", "core", "driver")]


class TestReadSurfaces:
    def test_reads_status_and_diff(self, tmp_path: Path) -> None:
        d = tmp_path / "surface-oracle"
        d.mkdir()
        (d / "status").write_text("changed\n")
        (d / "diff").write_text("+a\n")
        surfaces = {s.name: s for s in read_surfaces(tmp_path)}
        assert surfaces["oracle"] == Surface("oracle", "changed", "+a\n")

    def test_treats_a_missing_report_as_an_error(self, tmp_path: Path) -> None:
        surfaces = {s.name: s for s in read_surfaces(tmp_path)}
        assert surfaces["core"].status == "error"


class TestNeedsHuman:
    def test_is_false_when_nothing_changed(self) -> None:
        assert not needs_human(ALL_SAME, None)
        assert not needs_human(ALL_SAME, Answer(False, "refactor"))

    def test_is_true_when_a_surface_changed_or_failed(self) -> None:
        assert needs_human([*ALL_SAME, Surface("core", "changed", "")], None)
        assert needs_human([*ALL_SAME, Surface("core", "error", "")], None)

    def test_is_true_when_the_console_changed_visibly(self) -> None:
        assert needs_human(ALL_SAME, Answer(True, "new button"))


class TestPlanLabels:
    def test_sets_human_and_removes_bot(self) -> None:
        assert plan_labels({BOT}, True) == ([HUMAN], [BOT])

    def test_sets_bot_when_no_tier_is_set(self) -> None:
        assert plan_labels(set(), False) == ([BOT], [])

    def test_never_removes_human(self) -> None:
        assert plan_labels({HUMAN}, False) == ([], [])
        assert plan_labels({HUMAN, BOT}, False) == ([], [BOT])

    def test_changes_nothing_when_the_label_is_right(self) -> None:
        assert plan_labels({HUMAN}, True) == ([], [])
        assert plan_labels({BOT}, False) == ([], [])


class TestConsolePrompt:
    def test_is_none_without_console_files(self) -> None:
        files = [{"filename": "core/main.go", "status": "modified", "patch": "+x"}]
        assert console_prompt(files) is None

    def test_holds_only_console_patches(self) -> None:
        files: list[dict[str, Any]] = [
            {"filename": "console/src/a.tsx", "status": "modified", "patch": "+btn"},
            {"filename": "core/main.go", "status": "modified", "patch": "+go"},
            {"filename": "console/src/b.png", "status": "added", "patch": None},
        ]
        prompt = console_prompt(files)
        assert prompt is not None
        assert "+btn" in prompt
        assert "+go" not in prompt
        assert "console/src/b.png (added)\n(no patch" in prompt


class TestParseAnswer:
    def test_reads_the_last_json_answer(self) -> None:
        text = 'thinking {"x": 1}\n{"human": false, "reason": "Tests only."}'
        assert parse_answer(text) == Answer(False, "Tests only.")

    def test_answers_human_when_unclear(self) -> None:
        assert parse_answer("no idea").human
        assert parse_answer('{"human": "no", "reason": "x"}').human


class TestAskAgent:
    def test_sends_the_prompt_on_stdin(self, tmp_path: Path) -> None:
        agent = tmp_path / "agent.py"
        agent.write_text(
            "import json, sys\n"
            "print(json.dumps({'human': False, 'reason': sys.stdin.read()}))\n"
        )
        assert ask_agent("hello", f"{sys.executable} {agent}") == Answer(False, "hello")

    def test_answers_human_when_the_agent_fails(self) -> None:
        assert ask_agent("prompt", f"{sys.executable} -c 'exit(3)'").human


class TestRender:
    def test_shows_the_tier_and_the_changed_diff(self) -> None:
        surfaces = [*ALL_SAME[:4], Surface("driver", "changed", "+flag --x\n")]
        body = render(surfaces, None, kept=False)
        assert body.startswith(MARKER)
        assert f"`{HUMAN}`" in body
        assert "| Driver commands and config keys | Changed |" in body
        assert "+flag --x" in body
        assert "Console screenshots" not in body

    def test_explains_a_kept_human_label(self) -> None:
        body = render(ALL_SAME, Answer(False, "Refactor."), kept=True)
        assert f"## Review tier: `{HUMAN}`" in body
        assert "already set" in body
        assert "| Console | No visible change: Refactor. |" in body
        assert "Console screenshots" in body

    def test_fences_a_diff_that_holds_backticks(self) -> None:
        assert fence("+```\n").startswith("````diff")
