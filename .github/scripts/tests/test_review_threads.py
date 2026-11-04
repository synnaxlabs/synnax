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

SCRIPT = Path(__file__).resolve().parents[1] / "review_threads.sh"

# Stands in for gh: applies the --jq expression to each canned page in FAKE_GH_DIR, as
# gh --paginate does.
FAKE_GH = """#!/usr/bin/env bash
set -euo pipefail
EXPR="."
while [ $# -gt 0 ]; do
    case "$1" in
        --jq)
            EXPR=$2
            shift 2
            ;;
        -f | -F) shift 2 ;;
        *) shift ;;
    esac
done
for page in "$FAKE_GH_DIR"/page-*.json; do
    jq -r "$EXPR" "$page"
done
"""

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
        self.data = path / "gh"
        self.data.mkdir()
        bin_dir = path / "bin"
        bin_dir.mkdir()
        gh = bin_dir / "gh"
        gh.write_text(FAKE_GH)
        gh.chmod(0o755)
        self.env = {
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "FAKE_GH_DIR": str(self.data),
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }

    def pages(self, *pages: list[dict[str, Any]]) -> None:
        for i, threads in enumerate(pages):
            pull = {"author": {"login": AUTHOR}, "reviewThreads": {"nodes": threads}}
            body = {"data": {"repository": {"pullRequest": pull}}}
            (self.data / f"page-{i}.json").write_text(json.dumps(body))

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
