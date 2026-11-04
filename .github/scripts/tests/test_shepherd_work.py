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

SCRIPT = Path(__file__).resolve().parents[1] / "shepherd_work.sh"


FINDINGS = "<!-- greptile_outside_diff -->\n\n- **Bug** `a.go:1`\n"


class Harness:
    """A fake gh serving one pull request, its CI runs, threads, and comments."""

    def __init__(self, path: Path) -> None:
        self.gh = FakeGH(path)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
            "SHEPHERD_RETRY_SECONDS": "0",
        }
        self.pull()
        self.ok()
        self.runs()
        self.threads()
        self.comments()

    def pull(self, state: str = "clean") -> None:
        self.gh.serve(
            "api */pulls/7", {"head": {"sha": "head"}, "mergeable_state": state}
        )

    def runs(self, *conclusions: str) -> None:
        runs = [{"id": i + 1, "conclusion": c} for i, c in enumerate(conclusions)]
        self.gh.serve("api */actions/*", {"workflow_runs": runs})

    def ok(self, conclusion: str | None = None) -> None:
        self.gh.serve("api */check-runs*", {"check_runs": [{"conclusion": conclusion}]})

    def threads(self, count: int = 0) -> None:
        last = {"author": {"login": "someone", "__typename": "User"}}
        node = {"id": "t", "isResolved": False, "comments": {"nodes": [last]}}
        pull = {
            "author": {"login": "agent"},
            "reviewThreads": {"nodes": [node] * count},
        }
        self.gh.serve("api graphql", {"data": {"repository": {"pullRequest": pull}}})

    def comments(self, outside: bool = False) -> None:
        user = {"login": "greptile-apps[bot]", "type": "Bot"}
        at = "2026-01-01T00:00:00Z"
        found = [{"user": user, "body": FINDINGS, "created_at": at, "updated_at": at}]
        self.gh.serve("api */comments*", found if outside else [])

    def run(self) -> list[str]:
        result = subprocess.run(
            [str(SCRIPT), "7"], env=self.env, capture_output=True, text=True, check=True
        )
        return result.stdout.splitlines()


@pytest.fixture
def gh(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestShepherdWork:
    """Lists the work a pull request needs before it is done."""

    def test_should_print_nothing_for_a_done_pull_request(self, gh: Harness) -> None:
        gh.runs("success")
        assert gh.run() == []

    def test_should_report_a_conflict(self, gh: Harness) -> None:
        gh.pull(state="dirty")
        assert gh.run() == ["conflict"]

    def test_should_report_no_conflict_while_github_computes_it(
        self, gh: Harness
    ) -> None:
        gh.pull(state="unknown")
        assert gh.run() == []

    def test_should_report_a_failed_ci_run_on_the_head(self, gh: Harness) -> None:
        gh.runs("failure")
        assert gh.run() == ["ci 1"]

    def test_should_ignore_a_ci_run_still_in_progress(self, gh: Harness) -> None:
        gh.runs("")
        assert gh.run() == []

    def test_should_report_a_failed_ok_job_in_a_run_in_progress(
        self, gh: Harness
    ) -> None:
        gh.runs(None)
        gh.ok("failure")
        assert gh.run() == ["ci 1"]

    def test_should_report_threads_that_need_action(self, gh: Harness) -> None:
        gh.threads(count=2)
        assert gh.run() == ["threads 2"]

    def test_should_report_findings_outside_the_diff(self, gh: Harness) -> None:
        gh.comments(outside=True)
        assert gh.run() == ["outside-diff"]

    def test_should_report_every_kind_of_work_in_order(self, gh: Harness) -> None:
        gh.pull(state="dirty")
        gh.runs("failure")
        gh.threads(count=1)
        gh.comments(outside=True)
        assert gh.run() == ["conflict", "ci 1", "threads 1", "outside-diff"]
