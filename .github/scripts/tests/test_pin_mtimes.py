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
import time
from pathlib import Path

import pytest

import pin_mtimes


class Repo:
    """A scratch Git repository with a module directory and a file outside it."""

    def __init__(self, path: Path) -> None:
        self.path = path
        path.mkdir()
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.email", "ci@synnaxlabs.com")
        self.git("config", "user.name", "ci")
        self.write("mod/go.mod", "module mod\n")
        self.write("mod/testdata/fixture.json", '{"a": 1}\n')
        self.write("other.txt", "other\n")
        os.symlink("testdata/fixture.json", path / "mod" / "link.json")
        self.commit()

    def git(self, *args: str) -> str:
        return subprocess.run(
            ["git", *args], cwd=self.path, check=True, capture_output=True, text=True
        ).stdout.strip()

    def write(self, name: str, content: str) -> None:
        file = self.path / name
        file.parent.mkdir(parents=True, exist_ok=True)
        file.write_text(content)

    def commit(self) -> None:
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "commit")

    def clone(self, path: Path) -> "Repo":
        subprocess.run(["git", "clone", "-q", str(self.path), str(path)], check=True)
        clone = Repo.__new__(Repo)
        clone.path = path
        return clone

    def pin(self, path: str = "mod") -> int:
        cwd = Path.cwd()
        os.chdir(self.path)
        try:
            return pin_mtimes.pin(path)
        finally:
            os.chdir(cwd)

    def expected(self, name: str) -> int:
        return pin_mtimes.mtime_ns(self.git("rev-parse", f"HEAD:{name}"))

    def mtime(self, name: str) -> int:
        return os.lstat(self.path / name).st_mtime_ns


@pytest.fixture
def repo(tmp_path: Path) -> Repo:
    return Repo(tmp_path / "repo")


class TestPin:
    """Tests for setting modification times from Git object IDs."""

    @pytest.mark.parametrize(
        "name", ["mod", "mod/testdata", "mod/go.mod", "mod/testdata/fixture.json"]
    )
    def test_should_set_each_entry_from_its_object_id(
        self, repo: Repo, name: str
    ) -> None:
        repo.pin()
        assert repo.mtime(name) == repo.expected(name)

    def test_should_return_how_many_entries_it_set(self, repo: Repo) -> None:
        assert repo.pin() == 5

    def test_should_set_a_symbolic_link_and_not_its_target(self, repo: Repo) -> None:
        repo.pin()
        assert repo.mtime("mod/link.json") == repo.expected("mod/link.json")
        assert repo.expected("mod/link.json") != repo.expected(
            "mod/testdata/fixture.json"
        )

    def test_should_leave_entries_outside_the_path(self, repo: Repo) -> None:
        before = repo.mtime("other.txt")
        repo.pin()
        assert repo.mtime("other.txt") == before

    def test_should_give_two_checkouts_the_same_times(
        self, repo: Repo, tmp_path: Path
    ) -> None:
        clone = repo.clone(tmp_path / "clone")
        repo.pin()
        clone.pin()
        assert clone.mtime("mod/testdata/fixture.json") == repo.mtime(
            "mod/testdata/fixture.json"
        )

    def test_should_change_the_time_when_content_of_the_same_size_changes(
        self, repo: Repo
    ) -> None:
        repo.pin()
        before = repo.mtime("mod/testdata/fixture.json")
        repo.write("mod/testdata/fixture.json", '{"a": 2}\n')
        repo.commit()
        repo.pin()
        assert repo.mtime("mod/testdata/fixture.json") != before

    def test_should_set_times_older_than_the_go_test_cutoff(self, repo: Repo) -> None:
        repo.pin()
        cutoff = time.time_ns() - 2 * 10**9
        assert repo.mtime("mod/testdata/fixture.json") < cutoff
