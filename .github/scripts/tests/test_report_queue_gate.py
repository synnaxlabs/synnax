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

SCRIPT = Path(__file__).resolve().parents[1] / "report_queue_gate.sh"
BASE = "0123456789abcdef0123456789abcdef01234567"
QUEUE_REF = f"gh-readonly-queue/main/pr-42-{BASE}"
SHA = "fedcba9876543210fedcba9876543210fedcba98"


class Harness:
    """A fake gh serving a Sedosaur review and the status endpoint."""

    def __init__(self, path: Path) -> None:
        self.gh = FakeGH(path)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
            "GITHUB_SERVER_URL": "https://github.com",
            "GITHUB_RUN_ID": "7",
            "GH_API_RETRY_DELAY": "0",
        }
        self.gh.serve("api */statuses/*")
        self.gh.serve("api */git/ref/heads/*", {"object": {"sha": SHA}})
        self.group(42)
        self.score(5)

    def group(self, *prs: int) -> None:
        """Serves the commits of a merge group holding the given pull requests."""
        commits = [
            {"commit": {"message": f"Change {pr} (#{pr})\n\nBody"}} for pr in prs
        ]
        self.gh.serve("api */compare/*", {"commits": commits})

    def labels(self, *labels: str, pr: int = 42) -> None:
        """Serves a pull request with the given labels."""
        self.gh.serve(
            f"api */pulls/{pr}",
            {
                "head": {"sha": f"head{pr}"},
                "base": {"ref": "main"},
                "user": {"login": "someone"},
                "labels": [{"name": n} for n in labels],
            },
        )

    def score(self, score: int, pr: int = 42) -> None:
        """Serves the Sedosaur review that gives a pull request's head a score."""
        run = {
            "app": {"slug": "sedosaur"},
            "external_id": f"{pr}:main",
            "started_at": "2026-11-07T20:07:30Z",
            "conclusion": "success",
            "output": {"title": f"Confidence Score: {score}/5"},
        }
        self.gh.serve(f"api */commits/head{pr}/check-runs*", {"check_runs": [run]})

    def run(self, ref: str = QUEUE_REF) -> subprocess.CompletedProcess[str]:
        """Runs the script for a queue ref."""
        return subprocess.run(
            [str(SCRIPT), ref, SHA], env=self.env, capture_output=True, text=True
        )

    def statuses(self) -> list[list[str]]:
        """Returns the arguments of each status POST."""
        return [c for c in self.gh.calls() if c[1].endswith(f"/statuses/{SHA}")]


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    harness = Harness(tmp_path)
    harness.labels("review/bot")
    return harness


class TestReport:
    """Tests for the status a merge group gets."""

    def test_posts_success_to_the_queue_commit(self, harness: Harness) -> None:
        assert harness.run().returncode == 0
        post = harness.statuses()[0]
        assert "state=success" in post
        assert "context=Review gate" in post
        assert "description=Sedosaur scored the head 5/5" in post
        assert "target_url=https://github.com/synnaxlabs/synnax/actions/runs/7" in post

    def test_fails_a_pending_verdict(self, harness: Harness) -> None:
        harness.labels()
        assert harness.run().returncode == 0
        post = harness.statuses()[0]
        assert "state=failure" in post
        assert "description=waiting for one label: review/bot or review/human" in post

    def test_fails_a_low_score(self, harness: Harness) -> None:
        harness.score(4)
        assert harness.run().returncode == 0
        post = harness.statuses()[0]
        assert "state=failure" in post
        assert "description=Sedosaur scored the head 4/5, not 5/5" in post

    def test_judges_a_pull_request_with_a_slash_in_its_base(
        self, harness: Harness
    ) -> None:
        assert harness.run(f"gh-readonly-queue/a/b/pr-42-{BASE}").returncode == 0
        assert "state=success" in harness.statuses()[0]

    def test_passes_when_every_pull_request_in_the_group_passes(
        self, harness: Harness
    ) -> None:
        harness.group(41, 42)
        harness.labels("review/bot", pr=41)
        harness.score(5, pr=41)
        assert harness.run().returncode == 0
        post = harness.statuses()[0]
        assert "state=success" in post
        assert "description=Sedosaur scored the head 5/5" in post

    def test_fails_when_a_pull_request_ahead_fails(self, harness: Harness) -> None:
        harness.group(41, 42)
        harness.labels("review/bot", pr=41)
        harness.score(4, pr=41)
        assert harness.run().returncode == 0
        post = harness.statuses()[0]
        assert "state=failure" in post
        assert (
            "description=pull request 41: Sedosaur scored the head 4/5, not 5/5" in post
        )

    def test_retries_an_empty_response_to_the_post(self, harness: Harness) -> None:
        harness.gh.fail("api */statuses/*", "unexpected end of JSON input")
        assert harness.run().returncode == 0
        assert len(harness.statuses()) == 2


class TestFailure:
    """Tests for what the script does when it cannot judge."""

    def test_posts_a_failure_and_exits_non_zero_on_an_api_error(
        self, tmp_path: Path
    ) -> None:
        harness = Harness(tmp_path)
        assert harness.run().returncode != 0
        post = harness.statuses()[0]
        assert "state=failure" in post
        assert "description=could not judge pull request 42" in post

    def test_posts_a_failure_when_a_pull_request_ahead_cannot_be_judged(
        self, harness: Harness
    ) -> None:
        harness.group(41, 42)
        assert harness.run().returncode != 0
        assert "description=could not judge pull request 42" in harness.statuses()[0]

    def test_posts_a_failure_when_the_group_lacks_the_pull_request(
        self, harness: Harness
    ) -> None:
        harness.group(41)
        assert harness.run().returncode != 0
        assert "state=failure" in harness.statuses()[0]

    def test_posts_a_failure_when_the_ref_points_at_another_commit(
        self, harness: Harness
    ) -> None:
        harness.gh.serve("api */git/ref/heads/*", {"object": {"sha": BASE}})
        assert harness.run().returncode != 0
        post = harness.statuses()[0]
        assert "state=failure" in post
        assert "description=could not judge pull request 42" in post

    def test_posts_a_failure_when_the_ref_is_gone(self, harness: Harness) -> None:
        harness.gh.fail("api */git/ref/heads/*", "gh: Not Found (HTTP 404)")
        assert harness.run().returncode != 0
        assert "state=failure" in harness.statuses()[0]

    @pytest.mark.parametrize(
        "ref",
        [
            "main",
            "sy-5474-pr-relay",
            f"gh-readonly-queue/main/pr-x-{BASE}",
            "gh-readonly-queue/main/pr-42-short",
        ],
    )
    def test_rejects_a_ref_outside_the_queue(self, harness: Harness, ref: str) -> None:
        result = harness.run(ref)
        assert result.returncode == 1
        assert "not a merge queue ref" in result.stderr
        assert harness.gh.calls() == []
