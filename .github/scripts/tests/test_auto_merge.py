#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import subprocess
from collections.abc import Callable
from pathlib import Path
from typing import Any

import pytest
from scripts_common import FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "auto_merge.sh"
BOT = "synnax-claude-code[bot]"

Run = Callable[[dict[str, Any]], tuple[str, list[str]]]


def pull(
    labels: list[str],
    *,
    enabled_by: str | None = None,
    draft: bool = False,
    base: str = "main",
    stacked: bool = False,
    state: str = "open",
) -> dict[str, Any]:
    return {
        "state": state,
        "stack": {"number": 3980, "position": 1, "size": 2} if stacked else None,
        "labels": [{"name": n} for n in labels],
        "draft": draft,
        "base": {"ref": base},
        "auto_merge": None
        if enabled_by is None
        else {"enabled_by": {"login": enabled_by}},
    }


@pytest.fixture
def gh(tmp_path: Path) -> FakeGH:
    return FakeGH(tmp_path)


@pytest.fixture
def env(gh: FakeGH) -> dict[str, str]:
    return {
        **os.environ,
        "PATH": f"{gh.bin_dir}:{os.environ['PATH']}",
        "GITHUB_REPOSITORY": "synnaxlabs/synnax",
    }


@pytest.fixture
def run(gh: FakeGH, env: dict[str, str]) -> Run:
    gh.serve("pr merge")

    def run(pr: dict[str, Any]) -> tuple[str, list[str]]:
        gh.serve("api */pulls/7", pr)
        out = subprocess.run(
            [str(SCRIPT), "7"], env=env, capture_output=True, text=True, check=True
        )
        made = [" ".join(c) for c in gh.calls() if c[:2] == ["pr", "merge"]]
        return out.stdout.strip(), made

    return run


class TestAutoMerge:
    def test_turns_on_for_a_bot_tier_pull_request(self, run: Run) -> None:
        """Should turn auto-merge on for a review/bot pull request into main."""
        decision, calls = run(pull(["review/bot"]))
        assert decision == "enabled"
        assert calls == ["pr merge 7 --repo synnaxlabs/synnax --auto --squash"]

    def test_turns_on_for_a_release_branch(self, run: Run) -> None:
        """Should turn auto-merge on for a review/bot pull request into a release
        branch."""
        decision, _ = run(pull(["review/bot"], base="release/0.60"))
        assert decision == "enabled"

    def test_leaves_a_bot_tier_pull_request_already_on(self, run: Run) -> None:
        """Should not call gh when auto-merge is already on."""
        decision, calls = run(pull(["review/bot"], enabled_by=BOT))
        assert decision == "unchanged"
        assert calls == []

    @pytest.mark.parametrize(
        "pr",
        [
            pull(["review/human"]),
            pull(["review/bot", "review/human"]),
            pull(["review/bot"], draft=True),
            pull(["review/bot"], base="feature"),
            pull([]),
        ],
    )
    def test_never_turns_on_for_others(self, run: Run, pr: dict[str, Any]) -> None:
        """Should not turn auto-merge on for a pull request that does not qualify."""
        decision, calls = run(pr)
        assert decision == "unchanged"
        assert calls == []

    def test_leaves_a_stacked_pull_request(self, run: Run) -> None:
        """Should not turn auto-merge on for a pull request in a stack."""
        decision, calls = run(pull(["review/bot"], stacked=True))
        assert decision == "unchanged: the pull request is in a stack"
        assert calls == []

    def test_turns_off_its_own_auto_merge(self, run: Run) -> None:
        """Should turn off auto-merge it turned on once the pull request is raised to
        review/human."""
        decision, calls = run(pull(["review/bot", "review/human"], enabled_by=BOT))
        assert decision == "disabled"
        assert calls == ["pr merge 7 --repo synnaxlabs/synnax --disable-auto"]

    def test_keeps_a_person_auto_merge(self, run: Run) -> None:
        """Should keep auto-merge that the author turned on for a review/human pull
        request."""
        decision, calls = run(pull(["review/human"], enabled_by="emilbon99"))
        assert decision == "unchanged"
        assert calls == []

    @pytest.mark.parametrize(
        "pr",
        [
            pull(["review/bot"], enabled_by=BOT, base="feature", state="closed"),
            pull(["review/bot", "review/human"], enabled_by=BOT, state="closed"),
            pull(["review/bot"], state="closed"),
        ],
    )
    def test_leaves_a_closed_pull_request(self, run: Run, pr: dict[str, Any]) -> None:
        """Should not call gh for a merged or closed pull request, whose stale
        auto_merge field GitHub keeps."""
        decision, calls = run(pr)
        assert decision == "unchanged: the pull request is closed"
        assert calls == []

    def test_fails_when_an_open_pull_request_rejects_the_change(
        self, gh: FakeGH, env: dict[str, str]
    ) -> None:
        """Should fail when gh rejects the change on an open pull request."""
        gh.serve("api */pulls/7", pull(["review/human"], enabled_by=BOT))
        out = subprocess.run(
            [str(SCRIPT), "7"], env=env, capture_output=True, text=True, check=False
        )
        assert out.returncode != 0
        assert out.stdout == ""

    def test_retries_an_empty_read(
        self, gh: FakeGH, env: dict[str, str], run: Run
    ) -> None:
        """Should decide from the retry when GitHub answers the read with nothing."""
        env["GH_API_RETRY_DELAY"] = "0"
        gh.fail("api */pulls/7", "unexpected end of JSON input")
        decision, _ = run(pull(["review/bot"]))
        assert decision == "enabled"
