# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parent / "resolve_version.sh"


class Repo:
    """A scratch Git repository with product tags."""

    def __init__(self, path: Path) -> None:
        self.path = path
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.email", "ci@synnaxlabs.com")
        self.git("config", "user.name", "ci")
        self.commit("root")

    def git(self, *args: str) -> str:
        return subprocess.run(
            ["git", *args], cwd=self.path, check=True, capture_output=True, text=True
        ).stdout.strip()

    def commit(self, message: str) -> None:
        self.git("commit", "-q", "--allow-empty", "-m", message)

    def tag(self, *names: str) -> None:
        for name in names:
            self.git("tag", name)

    def resolve(self, product: str, bump: str, prerelease: bool = False) -> dict:
        result = subprocess.run(
            [str(SCRIPT), product, bump, "true" if prerelease else "false"],
            cwd=self.path,
            check=True,
            capture_output=True,
            text=True,
        )
        return dict(line.split("=", 1) for line in result.stdout.splitlines())

    def resolve_error(self, product: str, bump: str) -> str:
        result = subprocess.run(
            [str(SCRIPT), product, bump],
            cwd=self.path,
            capture_output=True,
            text=True,
        )
        assert result.returncode != 0
        return result.stderr


@pytest.fixture
def repo(tmp_path: Path) -> Repo:
    repo = Repo(tmp_path)
    repo.tag("core/v0.58.2", "console/v0.58.2", "driver/v0.58.2")
    repo.commit("work")
    return repo


class TestBumps:
    """Tests for patch and minor bumps from the reachable base."""

    def test_patch_bump(self, repo: Repo) -> None:
        out = repo.resolve("console", "patch")
        assert out["version"] == "0.58.3"
        assert out["tag"] == "console/v0.58.3"
        assert out["minor"] == "0.58"
        assert out["previous_tag"] == "console/v0.58.2"

    def test_minor_bump(self, repo: Repo) -> None:
        out = repo.resolve("driver", "minor")
        assert out["version"] == "0.59.0"
        assert out["minor"] == "0.59"
        assert out["previous_tag"] == "driver/v0.58.2"

    def test_should_take_the_base_from_the_reachable_tag(self, repo: Repo) -> None:
        repo.tag("console/v0.59.0")
        repo.commit("more")
        assert repo.resolve("console", "patch")["version"] == "0.59.1"

    def test_should_ignore_candidates_when_picking_the_base(self, repo: Repo) -> None:
        repo.tag("console/v0.59.0-rc.1", "console/v0.59.0-rc.2")
        assert repo.resolve("console", "minor")["version"] == "0.59.0"


class TestHotfixes:
    """Tests for patches that live on release branches."""

    def test_should_count_patches_from_unreachable_hotfix_tags(
        self, repo: Repo
    ) -> None:
        repo.git("checkout", "-q", "-b", "release/console-0.58", "console/v0.58.2")
        repo.commit("hotfix")
        repo.tag("console/v0.58.3")
        repo.git("checkout", "-q", "main")
        out = repo.resolve("console", "patch")
        assert out["version"] == "0.58.4"
        assert out["previous_tag"] == "console/v0.58.2"

    def test_should_resolve_the_minor_from_the_branch(self, repo: Repo) -> None:
        repo.tag("core/v0.59.0", "console/v0.59.0")
        repo.git("checkout", "-q", "-b", "release/console-0.58", "console/v0.58.2")
        repo.commit("hotfix")
        assert repo.resolve("console", "patch")["version"] == "0.58.3"


class TestCandidates:
    """Tests for the pre-release channel."""

    def test_first_candidate(self, repo: Repo) -> None:
        assert (
            repo.resolve("core", "minor", prerelease=True)["version"] == "0.59.0-rc.1"
        )

    def test_should_count_up_from_existing_candidates(self, repo: Repo) -> None:
        repo.tag("core/v0.59.0-rc.1", "core/v0.59.0-rc.2")
        out = repo.resolve("core", "minor", prerelease=True)
        assert out["version"] == "0.59.0-rc.3"
        assert out["tag"] == "core/v0.59.0-rc.3"

    def test_should_promote_with_the_same_bump(self, repo: Repo) -> None:
        repo.tag("core/v0.59.0-rc.3")
        out = repo.resolve("core", "minor")
        assert out["version"] == "0.59.0"
        assert out["previous_tag"] == "core/v0.58.2"


class TestTrainRule:
    """Tests for the one-minor-ahead cap."""

    def test_should_allow_one_minor_ahead_of_core(self, repo: Repo) -> None:
        assert repo.resolve("console", "minor")["version"] == "0.59.0"

    def test_should_reject_two_minors_ahead_of_core(self, repo: Repo) -> None:
        repo.tag("console/v0.59.0")
        repo.commit("more")
        assert "train rule" in repo.resolve_error("console", "minor")

    def test_should_let_core_open_the_next_train(self, repo: Repo) -> None:
        repo.tag("console/v0.59.0", "driver/v0.59.0")
        assert repo.resolve("core", "minor")["version"] == "0.59.0"


class TestErrors:
    """Tests for unusable inputs."""

    def test_should_fail_without_a_reachable_tag(self, tmp_path: Path) -> None:
        repo = Repo(tmp_path)
        repo.tag("core/v0.58.2")
        assert "no stable console tag" in repo.resolve_error("console", "patch")

    def test_should_reject_an_unknown_bump(self, repo: Repo) -> None:
        assert "unknown bump" in repo.resolve_error("core", "major")

    def test_should_reject_an_unknown_product(self, repo: Repo) -> None:
        assert "unknown product" in repo.resolve_error("python", "patch")


LATEST = Path(__file__).resolve().parent / "latest_version.sh"


def latest(repo: Repo, product: str, minor: str, candidates: bool = False) -> str:
    return subprocess.run(
        [str(LATEST), product, minor, "true" if candidates else "false"],
        cwd=repo.path,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()


class TestLatestVersion:
    """Tests for the highest release on a train."""

    def test_should_pick_the_highest_stable_on_the_train(self, repo: Repo) -> None:
        repo.tag("console/v0.58.3", "console/v0.58.10", "console/v0.59.0")
        assert latest(repo, "console", "0.58") == "0.58.10"

    def test_should_skip_candidates_by_default(self, repo: Repo) -> None:
        repo.tag("console/v0.59.0-rc.1")
        assert latest(repo, "console", "0.59") == ""
        assert latest(repo, "console", "0.59", candidates=True) == "0.59.0-rc.1"

    def test_should_rank_a_stable_above_its_candidates(self, repo: Repo) -> None:
        repo.tag("driver/v0.59.0-rc.1", "driver/v0.59.0-rc.2", "driver/v0.59.0")
        assert latest(repo, "driver", "0.59", candidates=True) == "0.59.0"

    def test_should_rank_a_later_candidate_above_an_earlier_stable(
        self, repo: Repo
    ) -> None:
        repo.tag("driver/v0.59.0", "driver/v0.59.1-rc.1")
        assert latest(repo, "driver", "0.59", candidates=True) == "0.59.1-rc.1"
        assert latest(repo, "driver", "0.59") == "0.59.0"

    def test_should_print_nothing_for_an_empty_train(self, repo: Repo) -> None:
        assert latest(repo, "core", "0.60") == ""
