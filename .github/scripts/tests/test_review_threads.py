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
from pathlib import Path
from typing import Any

import pytest
from scripts_common import FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "review_threads.sh"


AUTHOR = "synnax-claude-code"


def comment(login: str, kind: str = "User") -> dict[str, Any]:
    return {"author": {"login": login, "__typename": kind}, "body": "", "url": ""}


def thread(
    key: str, *comments: dict[str, Any], resolved: bool = False
) -> dict[str, Any]:
    return {
        "id": key,
        "isResolved": resolved,
        "isOutdated": False,
        "path": "a.go",
        "line": 1,
        "comments": {"nodes": list(comments)},
    }


class Harness:
    """A fake gh serving pages of one pull request's review threads."""

    def __init__(self, path: Path) -> None:
        self.gh = FakeGH(path)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }

    def pages(self, *pages: list[dict[str, Any]]) -> None:
        bodies = []
        for threads in pages:
            pull = {"author": {"login": AUTHOR}, "reviewThreads": {"nodes": threads}}
            bodies.append({"data": {"repository": {"pullRequest": pull}}})
        self.gh.serve("api graphql", *bodies)

    def run(self) -> list[str]:
        result = subprocess.run(
            [str(SCRIPT), "7"], env=self.env, capture_output=True, text=True, check=True
        )
        return [json.loads(line)["id"] for line in result.stdout.splitlines()]


@pytest.fixture
def gh(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestReviewThreads:
    """Selects the review threads that need action."""

    def test_should_select_a_greptile_thread(self, gh: Harness) -> None:
        gh.pages([thread("a", comment("greptile-apps", "Bot"))])
        assert gh.run() == ["a"]

    def test_should_select_a_thread_a_person_spoke_last_in(self, gh: Harness) -> None:
        gh.pages([thread("a", comment("greptile-apps", "Bot"), comment("someone"))])
        assert gh.run() == ["a"]

    def test_should_skip_a_resolved_thread(self, gh: Harness) -> None:
        gh.pages([thread("a", comment("someone"), resolved=True)])
        assert gh.run() == []

    def test_should_skip_a_thread_the_author_answered(self, gh: Harness) -> None:
        gh.pages([thread("a", comment("someone"), comment(AUTHOR, "Bot"))])
        assert gh.run() == []

    def test_should_skip_a_thread_another_bot_answered(self, gh: Harness) -> None:
        gh.pages([thread("a", comment("someone"), comment("synnax-codex", "Bot"))])
        assert gh.run() == []

    def test_should_read_every_page(self, gh: Harness) -> None:
        gh.pages(
            [thread("a", comment("someone"))],
            [thread("b", comment("someone"), resolved=True)],
            [thread("c", comment("greptile-apps", "Bot"))],
        )
        assert gh.run() == ["a", "c"]
