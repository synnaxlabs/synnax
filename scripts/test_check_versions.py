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

SCRIPT = Path(__file__).resolve().parent / "check_versions.sh"

PYTHON_DIRS = ["alamos/py", "freighter/py", "client/py", "x/py"]
NODE_DIRS = [
    "alamos/ts",
    "arc/ts",
    "client/ts",
    "drift",
    "freighter/ts",
    "pluto",
    "x/media",
    "x/ts",
]
CPP_VERSION = "client/cpp/version/VERSION"


class Repo:
    """A scratch repo with every package manifest the script reads and Core tags."""

    def __init__(self, path: Path) -> None:
        self.path = path
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.email", "ci@synnaxlabs.com")
        self.git("config", "user.name", "ci")
        self.git("commit", "-q", "--allow-empty", "-m", "root")

    def git(self, *args: str) -> None:
        subprocess.run(["git", *args], cwd=self.path, check=True, capture_output=True)

    def tag(self, *names: str) -> None:
        for name in names:
            self.git("tag", name)

    def manifests(self, version: str, **overrides: str) -> None:
        """Writes every manifest at the version, with per-directory overrides."""
        for d in PYTHON_DIRS:
            v = overrides.get(d, version)
            self.write(
                f"{d}/pyproject.toml", f'[project]\nname = "x"\nversion = "{v}"\n'
            )
        for d in NODE_DIRS:
            v = overrides.get(d, version)
            self.write(
                f"{d}/package.json", f'{{\n  "name": "x",\n  "version": "{v}"\n}}\n'
            )
        self.write(CPP_VERSION, overrides.get("cpp", version) + "\n")

    def write(self, rel: str, content: str) -> None:
        file = self.path / rel
        file.parent.mkdir(parents=True, exist_ok=True)
        file.write_text(content)

    def check(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(SCRIPT), str(self.path)], capture_output=True, text=True, check=False
        )


@pytest.fixture
def repo(tmp_path: Path) -> Repo:
    repo = Repo(tmp_path)
    repo.tag("core/v0.58.2")
    repo.manifests("0.58.4")
    return repo


class TestTrainRule:
    """Tests for the one-minor window the packages must sit in."""

    def test_should_accept_the_core_minor(self, repo: Repo) -> None:
        result = repo.check()
        assert result.returncode == 0, result.stderr
        assert "All packages share 0.58.x." in result.stdout

    def test_should_accept_the_next_minor(self, repo: Repo) -> None:
        repo.manifests("0.59.0")
        assert repo.check().returncode == 0

    def test_should_reject_two_minors_ahead(self, repo: Repo) -> None:
        repo.manifests("0.60.0")
        result = repo.check()
        assert result.returncode != 0
        assert "not 0.58.x or 0.59.x" in result.stderr

    def test_should_reject_a_minor_behind(self, repo: Repo) -> None:
        repo.manifests("0.57.9")
        assert repo.check().returncode != 0

    def test_should_reject_a_split(self, repo: Repo) -> None:
        repo.manifests("0.58.4", pluto="0.59.0")
        result = repo.check()
        assert result.returncode != 0
        assert "Node (" in result.stderr and "pluto" in result.stderr

    def test_should_reject_a_split_cpp_client(self, repo: Repo) -> None:
        repo.manifests("0.58.4", cpp="0.59.0")
        result = repo.check()
        assert result.returncode != 0
        assert "C++ (" in result.stderr


class TestTags:
    """Tests for how the Core's latest stable minor is read."""

    def test_should_sort_tags_numerically(self, repo: Repo) -> None:
        repo.tag("core/v0.58.10", "core/v0.9.0")
        repo.manifests("0.59.0")
        result = repo.check()
        assert result.returncode == 0, result.stderr
        assert "from core/v0.58.10" in result.stdout

    def test_should_ignore_candidate_tags(self, repo: Repo) -> None:
        repo.tag("core/v0.59.0-rc.1")
        repo.manifests("0.60.0")
        assert repo.check().returncode != 0

    def test_should_ignore_other_products(self, repo: Repo) -> None:
        repo.tag("console/v0.60.0")
        repo.manifests("0.60.0")
        assert repo.check().returncode != 0

    def test_should_fail_without_a_stable_core_tag(self, tmp_path: Path) -> None:
        repo = Repo(tmp_path)
        repo.tag("core/v0.59.0-rc.1")
        repo.manifests("0.59.0")
        result = repo.check()
        assert result.returncode != 0
        assert "no stable core tag" in result.stderr
