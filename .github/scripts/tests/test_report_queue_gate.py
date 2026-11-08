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
    """A fake gh serving a Greptile review and the status endpoint."""

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
        self.gh.serve(
            "api */commits/head/check-runs*",
            {"check_runs": [{"name": "Greptile Review", "conclusion": "success"}]},
        )
        self.score(5)

    def labels(self, *labels: str) -> None:
        """Serves pull request 42 with the given labels."""
        self.gh.serve(
            "api */pulls/42",
            {
                "head": {"sha": "head"},
                "user": {"login": "someone"},
                "labels": [{"name": n} for n in labels],
            },
        )

    def score(self, score: int) -> None:
        """Serves the description revision that gives the head a Greptile score."""
        link = "https://github.com/synnaxlabs/synnax/commit/head)"
        diff = (
            f"<!-- greptile_comment -->\n<!-- greptile_confidence_score:{score} -->\n"
            f"{link}\n<!-- /greptile_comment -->"
        )
        edit = {"editor": {"__typename": "Bot", "login": "greptile-apps"}, "diff": diff}
        pull = {"userContentEdits": {"nodes": [edit]}}
        self.gh.serve("api graphql", {"data": {"repository": {"pullRequest": pull}}})

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
        assert "description=Greptile scored the head 5/5" in post
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
        assert "description=Greptile scored the head 4/5, not 5/5" in post

    def test_judges_a_pull_request_with_a_slash_in_its_base(
        self, harness: Harness
    ) -> None:
        assert harness.run(f"gh-readonly-queue/a/b/pr-42-{BASE}").returncode == 0
        assert "state=success" in harness.statuses()[0]

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
