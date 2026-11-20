#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "desktop_build_inputs.sh"


def git(path: Path, *args: str) -> None:
    subprocess.run(["git", *args], cwd=path, check=True, capture_output=True)


def run(path: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [str(SCRIPT), *args], cwd=path, capture_output=True, text=True
    )


def inputs(path: Path, *args: str) -> dict[str, str]:
    result = run(path, *args)
    assert result.returncode == 0, result.stderr
    return dict(line.split("=", 1) for line in result.stdout.splitlines())


@pytest.fixture
def repo(tmp_path: Path) -> Path:
    """A repository at a commit after core/v0.61.3, with a newer tag on a branch."""
    git(tmp_path, "init", "-q", "-b", "main")
    git(tmp_path, "config", "user.email", "ci@synnaxlabs.com")
    git(tmp_path, "config", "user.name", "ci")
    for message in ["root", "0.61", "work"]:
        git(tmp_path, "commit", "-q", "--allow-empty", "-m", message)
    git(tmp_path, "tag", "core/v0.61.0", "HEAD~1")
    git(tmp_path, "tag", "core/v0.61.3", "HEAD~1")
    git(tmp_path, "tag", "core/v0.62.0-rc.1")
    git(tmp_path, "checkout", "-q", "-b", "other", "HEAD~2")
    git(tmp_path, "commit", "-q", "--allow-empty", "-m", "other")
    git(tmp_path, "tag", "core/v0.70.0")
    git(tmp_path, "checkout", "-q", "main")
    return tmp_path


class TestDesktopBuildInputs:
    """Tests for the version and the Driver that a Synnax Desktop build takes."""

    def test_should_take_the_driver_and_the_core_version_of_the_run(
        self, repo: Path
    ) -> None:
        assert inputs(repo, "0.62.0", "0.62.0") == {
            "version": "0.62.0",
            "driver_release": "driver/v0.62.0",
        }

    def test_should_pair_candidates_on_one_minor(self, repo: Path) -> None:
        assert inputs(repo, "0.62.0-rc.2", "0.62.0-rc.5") == {
            "version": "0.62.0-rc.2",
            "driver_release": "driver/v0.62.0-rc.5",
        }

    def test_should_build_the_driver_when_the_run_releases_only_the_core(
        self, repo: Path
    ) -> None:
        assert inputs(repo, "0.62.0", "") == {
            "version": "0.62.0",
            "driver_release": "",
        }

    def test_should_build_the_driver_when_the_run_releases_only_the_driver(
        self, repo: Path
    ) -> None:
        assert inputs(repo, "", "0.62.0") == {
            "version": "0.61.3",
            "driver_release": "",
        }

    def test_should_take_the_newest_reachable_stable_core_tag(self, repo: Path) -> None:
        assert inputs(repo) == {"version": "0.61.3", "driver_release": ""}

    def test_should_build_the_driver_when_the_minors_differ(self, repo: Path) -> None:
        assert inputs(repo, "0.62.0", "0.61.4") == {
            "version": "0.62.0",
            "driver_release": "",
        }

    def test_should_fail_without_a_reachable_stable_core_tag(
        self, tmp_path: Path
    ) -> None:
        git(tmp_path, "init", "-q", "-b", "main")
        git(tmp_path, "config", "user.email", "ci@synnaxlabs.com")
        git(tmp_path, "config", "user.name", "ci")
        git(tmp_path, "commit", "-q", "--allow-empty", "-m", "root")
        result = run(tmp_path)
        assert result.returncode != 0
        assert "no stable core tag is reachable from HEAD" in result.stderr
