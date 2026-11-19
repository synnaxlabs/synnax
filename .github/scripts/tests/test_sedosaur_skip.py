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

SCRIPT = Path(__file__).resolve().parents[1] / "sedosaur_skip.sh"
BEFORE = "2026-11-07T20:07:30Z"
AFTER = "2026-11-07T20:09:05Z"


def review(
    started: str,
    conclusion: str | None = "success",
    *,
    key: str = "7:main",
    app: str = "sedosaur",
    text: str | None = "<!-- sedosaur_rules:2 -->",
) -> dict[str, object]:
    """A check run of a Sedosaur review."""
    return {
        "app": {"slug": app},
        "external_id": key,
        "started_at": started,
        "conclusion": conclusion,
        "output": {"text": text},
    }


class Harness:
    """A fake gh serving the check runs of the head."""

    def __init__(self, path: Path) -> None:
        self.gh = FakeGH(path)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
            "HEAD_SHA": "head",
            "KEY": "7:main",
            "FORK": "false",
            "REPEAT": "false",
            "REPLAY": "false",
        }
        self.reviews()

    def reviews(self, *runs: dict[str, object]) -> None:
        """Sets the check runs of Sedosaur's reviews of the head."""
        self.gh.serve("api */commits/head/check-runs*", {"check_runs": list(runs)})

    def run(self, **env: str) -> bool:
        """Runs the script with the environment changes, and returns whether it skips
        the review."""
        result = subprocess.run(
            [str(SCRIPT)], env=self.env | env, capture_output=True, text=True
        )
        assert result.returncode == 0, result.stderr
        assert result.stdout in ("true\n", "false\n"), result.stdout
        return result.stdout == "true\n"


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestRequest:
    def test_should_review_a_head_with_no_review(self, harness: Harness) -> None:
        assert not harness.run()

    def test_should_skip_a_head_whose_last_review_succeeded(
        self, harness: Harness
    ) -> None:
        harness.reviews(review(BEFORE))
        assert harness.run()

    @pytest.mark.parametrize("text", [None, "<!-- sedosaur_findings [] -->"])
    def test_should_review_a_head_reviewed_under_older_rules(
        self, harness: Harness, text: str | None
    ) -> None:
        harness.reviews(review(BEFORE, text=text))
        assert not harness.run()

    def test_should_review_a_head_whose_last_review_failed(
        self, harness: Harness
    ) -> None:
        harness.reviews(review(BEFORE), review(AFTER, "failure"))
        assert not harness.run()

    def test_should_skip_after_a_success_that_follows_a_failure(
        self, harness: Harness
    ) -> None:
        harness.reviews(review(BEFORE, "failure"), review(AFTER))
        assert harness.run()

    def test_should_ignore_a_review_against_another_base(
        self, harness: Harness
    ) -> None:
        harness.reviews(review(BEFORE, key="7:release/0.57"))
        assert not harness.run()

    def test_should_ignore_a_check_run_of_another_app(self, harness: Harness) -> None:
        harness.reviews(review(BEFORE, app="greptile-apps"))
        assert not harness.run()

    def test_should_ask_for_every_check_run_of_the_review(
        self, harness: Harness
    ) -> None:
        harness.run()
        [call] = [c for c in harness.gh.calls() if "check-runs" in c[1]]
        assert "check_name=Sedosaur%20Review" in call[1]
        assert "filter=all" in call[1]


class TestRepeat:
    def test_should_review_a_repeat_without_reading_the_reviews(
        self, harness: Harness
    ) -> None:
        harness.reviews(review(AFTER))
        assert not harness.run(REPEAT="true")
        assert harness.gh.calls() == []


class TestFork:
    @pytest.fixture(autouse=True)
    def maintainer(self, harness: Harness) -> None:
        harness.gh.serve(
            "api */collaborators/maintainer/permission", {"permission": "write"}
        )
        harness.gh.serve(
            "api */collaborators/someone/permission", {"permission": "read"}
        )

    def test_should_review_a_fork_a_maintainer_asked_for(
        self, harness: Harness
    ) -> None:
        assert not harness.run(FORK="true", REQUESTER="maintainer")

    def test_should_skip_a_fork_a_maintainer_reviewed(self, harness: Harness) -> None:
        harness.reviews(review(BEFORE))
        assert harness.run(FORK="true", REQUESTER="maintainer")

    @pytest.mark.parametrize("requester", ["", "someone", "synnax-claude-code[bot]"])
    def test_should_skip_a_fork_no_maintainer_asked_for(
        self, harness: Harness, requester: str
    ) -> None:
        harness.reviews()
        assert harness.run(FORK="true", REQUESTER=requester, REPEAT="true")
        assert all("check-runs" not in c[1] for c in harness.gh.calls())


class TestReplay:
    def test_should_review_a_replay_of_a_reviewed_head(self, harness: Harness) -> None:
        harness.reviews(review(AFTER))
        assert not harness.run(REPLAY="true")
        assert harness.gh.calls() == []
