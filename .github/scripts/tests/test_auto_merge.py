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
from collections.abc import Callable
from pathlib import Path
from typing import Any

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "auto_merge.sh"
BOT = "synnax-claude-code[bot]"

Run = Callable[[dict[str, Any]], tuple[str, list[str]]]

# Stands in for gh: serves the pull request from FAKE_GH_DIR with the requested jq
# expression, and records every `gh pr merge` call.
FAKE_GH = """#!/usr/bin/env bash
set -euo pipefail
if [ "$1" = pr ]; then
    echo "$*" >> "$FAKE_GH_DIR/merge-calls"
    exit 0
fi
EXPR="."
while [ $# -gt 0 ]; do
    case "$1" in
        --jq)
            EXPR=$2
            shift 2
            ;;
        *) shift ;;
    esac
done
jq -r "$EXPR" "$FAKE_GH_DIR/pull.json"
"""


def pull(
    labels: list[str],
    *,
    enabled_by: str | None = None,
    draft: bool = False,
    base: str = "main",
    stacked: bool = False,
) -> dict[str, Any]:
    return {
        "stack": {"number": 3980, "position": 1, "size": 2} if stacked else None,
        "labels": [{"name": n} for n in labels],
        "draft": draft,
        "base": {"ref": base},
        "auto_merge": None
        if enabled_by is None
        else {"enabled_by": {"login": enabled_by}},
    }


@pytest.fixture
def run(tmp_path: Path) -> Run:
    data = tmp_path / "gh"
    data.mkdir()
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    gh = bin_dir / "gh"
    gh.write_text(FAKE_GH)
    gh.chmod(0o755)
    env = {
        **os.environ,
        "PATH": f"{bin_dir}:{os.environ['PATH']}",
        "FAKE_GH_DIR": str(data),
        "GITHUB_REPOSITORY": "synnaxlabs/synnax",
    }

    def run(pr: dict[str, Any]) -> tuple[str, list[str]]:
        (data / "pull.json").write_text(json.dumps(pr))
        calls = data / "merge-calls"
        calls.unlink(missing_ok=True)
        out = subprocess.run(
            [str(SCRIPT), "7"], env=env, capture_output=True, text=True, check=True
        )
        made = calls.read_text().splitlines() if calls.exists() else []
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
        decision, _ = run(pull(["review/bot"], base="release/core-0.60"))
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
