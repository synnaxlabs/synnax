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
from pathlib import Path
from typing import Any

import pytest
from scripts_common import FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "greptile_outside_diff.sh"


GREPTILE = "greptile-apps[bot]"
MARKER = "<!-- greptile_outside_diff -->\n\n<h3>Comments Outside Diff</h3>\n\n"
FINDINGS = MARKER + "These findings could not be posted inline.\n\n- **Bug** `a.go:1`\n"
CLEARED = MARKER + "No findings outside the diff remain.\n"
ANSWER = "<!-- agent:outside-diff -->\nFixed the bug."


def comment(
    login: str, body: str, at: str, updated: str = "", kind: str = "Bot"
) -> dict[str, Any]:
    return {
        "user": {"login": login, "type": kind},
        "body": body,
        "created_at": at,
        "updated_at": updated or at,
    }


class Harness:
    """A fake gh serving one pull request's top-level comments."""

    def __init__(self, path: Path) -> None:
        self.gh = FakeGH(path)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }
        self.comments()

    def comments(self, *comments: dict[str, Any]) -> None:
        self.gh.serve("api */comments*", list(comments))

    def run(self) -> str:
        result = subprocess.run(
            [str(SCRIPT), "7"], env=self.env, capture_output=True, text=True, check=True
        )
        return result.stdout.strip()


@pytest.fixture
def gh(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestGreptileOutsideDiff:
    """Prints Greptile findings outside the diff that need action."""

    def test_should_print_nothing_without_the_comment(self, gh: Harness) -> None:
        assert gh.run() == ""

    def test_should_print_open_findings(self, gh: Harness) -> None:
        gh.comments(comment(GREPTILE, FINDINGS, "2026-01-01T00:00:00Z"))
        assert gh.run() == FINDINGS.strip()

    def test_should_print_nothing_once_greptile_clears_them(self, gh: Harness) -> None:
        gh.comments(comment(GREPTILE, CLEARED, "2026-01-01T00:00:00Z"))
        assert gh.run() == ""

    def test_should_print_nothing_after_an_agent_answers(self, gh: Harness) -> None:
        gh.comments(
            comment(GREPTILE, FINDINGS, "2026-01-01T00:00:00Z"),
            comment("synnax-codex[bot]", ANSWER, "2026-01-02T00:00:00Z"),
        )
        assert gh.run() == ""

    def test_should_print_findings_greptile_edits_after_an_answer(
        self, gh: Harness
    ) -> None:
        gh.comments(
            comment(GREPTILE, FINDINGS, "2026-01-01T00:00:00Z", "2026-01-03T00:00:00Z"),
            comment("synnax-codex[bot]", ANSWER, "2026-01-02T00:00:00Z"),
        )
        assert gh.run() == FINDINGS.strip()

    def test_should_ignore_comments_that_do_not_answer(self, gh: Harness) -> None:
        gh.comments(
            comment(GREPTILE, FINDINGS, "2026-01-01T00:00:00Z"),
            comment("synnax-codex[bot]", "Fixed CI", "2026-01-02T00:00:00Z"),
            comment("someone", ANSWER, "2026-01-02T00:00:00Z", kind="User"),
        )
        assert gh.run() == FINDINGS.strip()
