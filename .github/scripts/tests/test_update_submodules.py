#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import shutil
import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "update_submodules.sh"

# Isolates Git from the host's config, and allows file:// submodule URLs.
ENV = {
    **os.environ,
    "GIT_CONFIG_GLOBAL": os.devnull,
    "GIT_CONFIG_NOSYSTEM": "1",
    "GIT_CONFIG_COUNT": "1",
    "GIT_CONFIG_KEY_0": "protocol.file.allow",
    "GIT_CONFIG_VALUE_0": "always",
    "GIT_AUTHOR_NAME": "test",
    "GIT_AUTHOR_EMAIL": "test@synnaxlabs.com",
    "GIT_COMMITTER_NAME": "test",
    "GIT_COMMITTER_EMAIL": "test@synnaxlabs.com",
}


def git(cwd: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args], cwd=cwd, env=ENV, capture_output=True, text=True, check=True
    ).stdout.strip()


def repo(path: Path, file: str) -> Path:
    git(path.parent, "init", "-q", "-b", "main", path.name)
    commit(path, file)
    return path


def commit(path: Path, file: str) -> None:
    (path / file).write_text(file)
    git(path, "add", file)
    git(path, "commit", "-q", "-m", file)


def run(work: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [str(SCRIPT)], cwd=work, env=ENV, capture_output=True, text=True
    )


def update(work: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", "submodule", "update", "--init", "--recursive"],
        cwd=work,
        env=ENV,
        capture_output=True,
        text=True,
    )


def gitdir(work: Path, path: str) -> Path:
    return Path(git(work / path, "rev-parse", "--absolute-git-dir"))


def empty_worktree(work: Path, path: str) -> None:
    for child in (work / path).iterdir():
        if child.name != ".git":
            child.unlink()


def kill_fetch(work: Path, path: str) -> None:
    """Leaves the submodule as a clone killed during its fetch: no objects, no refs, and
    a HEAD that names an unborn branch."""
    directory = gitdir(work, path)
    for entry in ("objects", "refs"):
        shutil.rmtree(directory / entry)
        (directory / entry).mkdir()
    (directory / "packed-refs").unlink(missing_ok=True)
    (directory / "HEAD").write_text("ref: refs/heads/main\n")
    empty_worktree(work, path)


def kill_checkout(work: Path, path: str, *, locked: bool) -> None:
    """Leaves the submodule as a checkout killed partway: HEAD on the fetched branch,
    part of the files, and a stale index.lock when locked, else no index."""
    git(work / path, "fetch", "-q", "origin", "+main:refs/heads/main")
    git(work / path, "symbolic-ref", "HEAD", "refs/heads/main")
    directory = gitdir(work, path)
    empty_worktree(work, path)
    (work / path / "plain.txt").write_text("partial")
    if locked:
        (directory / "index.lock").touch()
    else:
        (directory / "index").unlink()


@pytest.fixture
def work(tmp_path: Path) -> Path:
    """A clone of a superproject with two submodules, one of which nests a third. The
    plain submodule's main branch is a commit past the one the superproject records."""
    leaf = repo(tmp_path / "leaf", "leaf.txt")
    outer = repo(tmp_path / "outer", "outer.txt")
    git(outer, "submodule", "add", "-q", str(leaf), "deps/leaf")
    git(outer, "commit", "-q", "-m", "leaf")
    plain = repo(tmp_path / "plain", "plain.txt")
    root = repo(tmp_path / "root", "root.txt")
    git(root, "submodule", "add", "-q", str(outer), "vendor/outer")
    git(root, "submodule", "add", "-q", str(plain), "vendor/plain")
    git(root, "commit", "-q", "-m", "submodules")
    commit(plain, "later.txt")
    git(tmp_path, "clone", "-q", str(root), "work")
    work = tmp_path / "work"
    assert run(work).returncode == 0
    return work


class TestUpdateSubmodules:
    """Tests for checking out submodules in a reused workspace."""

    def test_should_check_out_every_submodule_in_a_fresh_clone(
        self, work: Path
    ) -> None:
        assert (work / "vendor/outer/outer.txt").read_text() == "outer.txt"
        assert (work / "vendor/outer/deps/leaf/leaf.txt").read_text() == "leaf.txt"
        assert (work / "vendor/plain/plain.txt").read_text() == "plain.txt"

    def test_should_clone_a_submodule_killed_during_its_fetch_again(
        self, work: Path
    ) -> None:
        recorded = git(work, "rev-parse", "HEAD:vendor/plain")
        kill_fetch(work, "vendor/plain")
        assert "Unable to find current revision" in update(work).stderr
        result = run(work)
        assert result.returncode == 0, result.stderr
        assert "::warning::Deleting unfinished submodule" in result.stdout
        assert git(work / "vendor/plain", "rev-parse", "HEAD") == recorded
        assert (work / "vendor/plain/plain.txt").read_text() == "plain.txt"

    @pytest.mark.parametrize(
        ("locked", "error"),
        [(True, "index.lock': File exists"), (False, "would be overwritten")],
    )
    def test_should_clone_a_submodule_killed_during_its_checkout_again(
        self, work: Path, locked: bool, error: str
    ) -> None:
        recorded = git(work, "rev-parse", "HEAD:vendor/plain")
        kill_checkout(work, "vendor/plain", locked=locked)
        assert error in update(work).stderr
        result = run(work)
        assert result.returncode == 0, result.stderr
        assert "::warning::Deleting unfinished submodule" in result.stdout
        assert git(work / "vendor/plain", "rev-parse", "HEAD") == recorded
        assert (work / "vendor/plain/plain.txt").read_text() == "plain.txt"

    def test_should_clone_an_unfinished_nested_submodule_again(
        self, work: Path
    ) -> None:
        kill_fetch(work, "vendor/outer/deps/leaf")
        result = run(work)
        assert result.returncode == 0, result.stderr
        assert (work / "vendor/outer/deps/leaf/leaf.txt").read_text() == "leaf.txt"

    def test_should_keep_finished_submodules(self, work: Path) -> None:
        (work / "vendor/outer/untracked.txt").write_text("kept")
        kill_fetch(work, "vendor/plain")
        result = run(work)
        assert result.returncode == 0, result.stderr
        assert result.stdout.count("::warning::") == 1
        assert (work / "vendor/outer/untracked.txt").read_text() == "kept"
