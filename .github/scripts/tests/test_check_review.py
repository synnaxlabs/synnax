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

SCRIPT = Path(__file__).resolve().parents[1] / "check_review.sh"


EARLIER = "2026-11-07T20:07:30Z"
LATER = "2026-11-07T20:09:05Z"


def review(
    score: int | None = 5,
    conclusion: str | None = "success",
    *,
    started: str = EARLIER,
    key: str = "42:main",
    app: str = "sedosaur",
) -> dict[str, Any]:
    """A check run of a Sedosaur review."""
    title = f"Confidence Score: {score}/5"
    if score is None:
        title = "The review ended with failure"
    return {
        "app": {"slug": app},
        "external_id": key,
        "started_at": started,
        "conclusion": conclusion,
        "output": {"title": title},
    }


class Harness:
    """A fake gh serving one pull request and the records the gate reads."""

    def __init__(self, path: Path) -> None:
        self.gh = FakeGH(path)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }
        self.pull("someone")
        self.commits(commit())
        self.reviews("head", review())

    def pull(self, author: str, *labels: str, fork: bool = False) -> None:
        """Sets the account that opened the pull request, its labels, and whether its
        head is in a fork."""
        repo = "synnaxlabs/synnax"
        payload = {
            "head": {
                "sha": "head",
                "repo": {"full_name": f"{author}/synnax" if fork else repo},
            },
            "base": {"ref": "main", "repo": {"full_name": repo}},
            "user": {"login": author},
            "labels": [{"name": n} for n in labels or ("review/bot",)],
        }
        self.gh.serve("api */pulls/42", payload)

    def commits(self, *commits: dict[str, object]) -> None:
        """Sets the commits on the pull request's branch."""
        self.gh.serve("api */pulls/42/commits*", list(commits))

    def reviews(self, sha: str, *runs: dict[str, Any]) -> None:
        """Sets the Sedosaur Review check runs of a commit."""
        self.gh.serve(f"api */commits/{sha}/check-runs*", {"check_runs": list(runs)})

    def run(self) -> tuple[str, str]:
        """Runs the script and returns the (state, description) it reports."""
        result = subprocess.run(
            [str(SCRIPT), "42"], env=self.env, capture_output=True, text=True
        )
        assert result.returncode == 0, result.stderr
        state, description = result.stdout.rstrip("\n").split("\t")
        return state, description


def commit(
    author: str = "dependabot[bot]",
    *,
    committer: str = "web-flow",
    verified: bool = True,
) -> dict[str, object]:
    """A commit on the pull request's branch."""
    return {
        "sha": "sha",
        "author": {"login": author},
        "committer": {"login": committer},
        "commit": {"verification": {"verified": verified}},
    }


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestReview:
    """Tests for the Sedosaur review requirement."""

    def test_passes_a_reviewed_head(self, harness: Harness) -> None:
        assert harness.run() == ("success", "Sedosaur scored the head 5/5")

    def test_pends_without_a_review(self, harness: Harness) -> None:
        harness.reviews("head")
        state, description = harness.run()
        assert state == "pending"
        assert description == "waiting for the Sedosaur review of the head"

    def test_pends_while_the_review_runs(self, harness: Harness) -> None:
        harness.reviews("head", review(None, None))
        assert harness.run()[0] == "pending"

    def test_fails_a_review_that_failed(self, harness: Harness) -> None:
        harness.reviews("head", review(None, "failure"))
        state, description = harness.run()
        assert state == "failure"
        assert description == "the Sedosaur review of the head ended failure"

    def test_rejects_a_review_of_an_earlier_commit(self, harness: Harness) -> None:
        harness.reviews("head")
        harness.reviews("earlier", review())
        assert harness.run()[0] == "pending"

    def test_ignores_another_app(self, harness: Harness) -> None:
        harness.reviews("head", review(app="someone"))
        assert harness.run()[0] == "pending"

    def test_ignores_a_review_for_another_base(self, harness: Harness) -> None:
        harness.reviews("head", review(key="42:release/0.57"))
        assert harness.run()[0] == "pending"

    def test_ignores_a_review_of_another_pull_request(self, harness: Harness) -> None:
        harness.reviews("head", review(key="43:main"))
        assert harness.run()[0] == "pending"

    def test_takes_the_newest_review(self, harness: Harness) -> None:
        harness.reviews("head", review(5, started=LATER), review(2))
        assert harness.run()[0] == "success"

    def test_rejects_an_earlier_success(self, harness: Harness) -> None:
        harness.reviews("head", review(5), review(None, "cancelled", started=LATER))
        assert harness.run() == (
            "failure",
            "the Sedosaur review of the head ended cancelled",
        )

    def test_asks_only_for_the_sedosaur_check_runs(self, harness: Harness) -> None:
        harness.run()
        [call] = [c for c in harness.gh.calls() if "check-runs" in c[1]]
        assert "check_name=Sedosaur%20Review" in call[1]
        assert "filter=all" in call[1]


class TestScore:
    """Tests for the Sedosaur confidence score requirement."""

    def test_fails_below_five(self, harness: Harness) -> None:
        harness.reviews("head", review(4))
        state, description = harness.run()
        assert state == "failure"
        assert description == "Sedosaur scored the head 4/5, not 5/5"

    def test_fails_without_a_score(self, harness: Harness) -> None:
        harness.reviews("head", review(None))
        assert harness.run() == ("failure", "Sedosaur left no score on the head")


class TestFork:
    """Tests for pull requests from forks, which Sedosaur does not review."""

    def test_fails_a_fork(self, harness: Harness) -> None:
        harness.pull("someone", "review/human", fork=True)
        harness.reviews("head")
        assert harness.run() == (
            "failure",
            "Sedosaur does not review forks, so an admin merges this one",
        )

    def test_fails_a_deleted_fork(self, harness: Harness) -> None:
        harness.pull("someone", "review/human")
        payload = {
            "head": {"sha": "head", "repo": None},
            "base": {"ref": "main", "repo": {"full_name": "synnaxlabs/synnax"}},
            "user": {"login": "someone"},
            "labels": [{"name": "review/human"}],
        }
        harness.gh.serve("api */pulls/42", payload)
        assert harness.run()[0] == "failure"

    def test_waits_for_a_tier_first(self, harness: Harness) -> None:
        harness.pull("someone", "core", fork=True)
        assert harness.run()[0] == "pending"


class TestDependabot:
    """Tests for the Dependabot exemption from the Sedosaur review."""

    @pytest.fixture(autouse=True)
    def unreviewed(self, harness: Harness) -> None:
        harness.pull("dependabot[bot]")
        harness.reviews("head")

    def test_passes_without_a_review(self, harness: Harness) -> None:
        assert harness.run() == ("success", "Dependabot wrote every commit")

    def test_pends_on_a_commit_someone_else_wrote(self, harness: Harness) -> None:
        harness.commits(commit(), commit("someone", committer="someone"))
        assert harness.run()[0] == "pending"

    def test_pends_on_a_forged_author(self, harness: Harness) -> None:
        harness.commits(commit(committer="someone"))
        assert harness.run()[0] == "pending"

    def test_pends_on_an_unsigned_commit(self, harness: Harness) -> None:
        harness.commits(commit(verified=False))
        assert harness.run()[0] == "pending"

    def test_requires_dependabot_to_open_it(self, harness: Harness) -> None:
        harness.pull("someone")
        assert harness.run()[0] == "pending"


class TestTier:
    """Tests for the review tier label requirement."""

    def test_pends_without_a_tier(self, harness: Harness) -> None:
        harness.pull("someone", "core")
        state, description = harness.run()
        assert state == "pending"
        assert description == "waiting for one label: review/bot or review/human"

    def test_pends_on_a_retired_tier(self, harness: Harness) -> None:
        harness.pull("someone", "review/light")
        assert harness.run()[0] == "pending"

    def test_pends_on_two_tiers(self, harness: Harness) -> None:
        harness.pull("someone", "review/bot", "review/human")
        assert harness.run()[0] == "pending"

    def test_passes_a_bot_tier_without_a_human(self, harness: Harness) -> None:
        harness.pull("someone", "core", "review/bot")
        assert harness.run()[0] == "success"

    def test_passes_a_human_tier_without_an_approval(self, harness: Harness) -> None:
        harness.pull("someone", "review/human")
        assert harness.run() == ("success", "Sedosaur scored the head 5/5")

    @pytest.mark.parametrize("route", ["api */pulls/42", "api */check-runs*"])
    def test_retries_an_empty_response(self, harness: Harness, route: str) -> None:
        """Should decide from the retry when GitHub answers a read with nothing."""
        harness.env["GH_API_RETRY_DELAY"] = "0"
        harness.gh.fail(route, "unexpected end of JSON input")
        assert harness.run() == ("success", "Sedosaur scored the head 5/5")
