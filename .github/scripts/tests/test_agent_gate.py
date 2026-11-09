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

import pytest
from scripts_common import FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "agent_gate.sh"


AGENT = "synnax-claude-code[bot]"
MARKER = "<!-- agent:fix -->"


class Harness:
    """A fake gh serving one pull request, its comments, and its label events."""

    def __init__(self, path: Path) -> None:
        self.gh = FakeGH(path)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }
        self.pull()
        self.comments()
        self.events()

    def pull(
        self,
        author: str = AGENT,
        labels: tuple[str, ...] = (),
        state: str = "open",
        repo: str = "synnaxlabs/synnax",
    ) -> None:
        pull = {
            "state": state,
            "user": {"login": author},
            "labels": [{"name": name} for name in labels],
            "head": {"sha": "head", "repo": {"full_name": repo}},
        }
        self.gh.serve("api */pulls/7", pull)

    def comments(self, *comments: tuple[str, str, str]) -> None:
        body = [
            {"user": {"login": user}, "body": text, "created_at": at}
            for user, text, at in comments
        ]
        self.gh.serve("api */comments*", body)

    def events(self, *unlabeled_at: str) -> None:
        body = [
            {
                "event": "unlabeled",
                "label": {"name": "agent/needs-human"},
                "created_at": at,
            }
            for at in unlabeled_at
        ]
        self.gh.serve("api */events*", body)

    def run(self, sha: str = "") -> str:
        args = [str(SCRIPT), "7", "fix", "2"] + ([sha] if sha else [])
        result = subprocess.run(
            args, env=self.env, capture_output=True, text=True, check=True
        )
        return result.stdout.strip()


def attempt(at: str, user: str = "github-actions[bot]") -> tuple[str, str, str]:
    return (user, f"{MARKER}\nAgent attempt", at)


@pytest.fixture
def gh(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestAgentGate:
    """Decides whether an agent may act on a pull request."""

    def test_should_run_the_first_attempt_on_an_agent_pull_request(
        self, gh: Harness
    ) -> None:
        assert gh.run() == "run 1"

    def test_should_run_on_a_person_pull_request_with_the_autofix_label(
        self, gh: Harness
    ) -> None:
        gh.pull(author="someone", labels=("agent/autofix",))
        assert gh.run() == "run 1"

    def test_should_skip_a_person_pull_request(self, gh: Harness) -> None:
        gh.pull(author="someone")
        assert gh.run().startswith("skip")

    def test_should_skip_a_fork(self, gh: Harness) -> None:
        gh.pull(repo="someone/synnax")
        assert gh.run().startswith("skip")

    def test_should_skip_a_closed_pull_request(self, gh: Harness) -> None:
        gh.pull(state="closed")
        assert gh.run().startswith("skip")

    def test_should_skip_a_pull_request_that_waits_for_a_person(
        self, gh: Harness
    ) -> None:
        gh.pull(labels=("agent/needs-human",))
        assert gh.run().startswith("skip")

    def test_should_skip_when_the_head_moved(self, gh: Harness) -> None:
        assert gh.run(sha="old").startswith("skip")
        assert gh.run(sha="head") == "run 1"

    def test_should_count_earlier_attempts(self, gh: Harness) -> None:
        gh.comments(attempt("2026-01-01T00:00:00Z"))
        assert gh.run() == "run 2"

    def test_should_stop_at_the_cap(self, gh: Harness) -> None:
        gh.comments(attempt("2026-01-01T00:00:00Z"), attempt("2026-01-02T00:00:00Z"))
        assert gh.run() == "stop"

    def test_should_ignore_markers_other_users_post(self, gh: Harness) -> None:
        gh.comments(
            attempt("2026-01-01T00:00:00Z", user="someone"),
            attempt("2026-01-02T00:00:00Z", user="someone"),
        )
        assert gh.run() == "run 1"

    def test_should_restart_the_count_when_a_person_removes_the_label(
        self, gh: Harness
    ) -> None:
        gh.comments(attempt("2026-01-01T00:00:00Z"), attempt("2026-01-02T00:00:00Z"))
        gh.events("2026-01-03T00:00:00Z")
        assert gh.run() == "run 1"
