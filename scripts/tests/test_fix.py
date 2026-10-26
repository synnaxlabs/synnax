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

ROOT = Path(__file__).resolve().parents[2]


class Repo:
    """A Git repository with the fix scripts and no formatters on PATH."""

    def __init__(self, path: Path) -> None:
        self.path = path
        for f in ["scripts/fix.sh", "scripts/update_copyrights.sh"]:
            (path / f).parent.mkdir(parents=True, exist_ok=True)
            shutil.copy(ROOT / f, path / f)
        (path / "licenses/headers").mkdir(parents=True)
        shutil.copy(ROOT / "licenses/headers/template.txt", path / "licenses/headers")
        self.env = {**os.environ, "PATH": "/usr/bin:/bin"}
        self.git("init", "-q", "-b", "main")
        self.git("add", ".")
        self.git("-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "i")
        self.git("update-ref", "refs/remotes/origin/main", "HEAD")

    def git(self, *args: str) -> str:
        return subprocess.run(
            ["git", *args],
            cwd=self.path,
            env=self.env,
            check=True,
            capture_output=True,
            text=True,
        ).stdout

    def fix(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["bash", "scripts/fix.sh", *args],
            cwd=self.path,
            env=self.env,
            capture_output=True,
            text=True,
        )

    def write(self, name: str, text: str) -> None:
        (self.path / name).write_text(text)

    def header(self, name: str) -> bool:
        return "Copyright" in (self.path / name).read_text()


@pytest.fixture
def repo(tmp_path: Path) -> Repo:
    return Repo(tmp_path)


class TestFix:
    def test_adds_headers_to_untracked_files(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        assert repo.fix().returncode == 0
        assert repo.header("a.sh")

    def test_stages_fixes_to_staged_files(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        repo.git("add", "a.sh")
        result = repo.fix("--staged")
        assert result.returncode == 0
        assert "fixed a.sh" in result.stdout
        assert "Copyright" in repo.git("show", ":a.sh")
        assert repo.git("diff", "--name-only") == ""

    def test_rejects_partially_staged_files(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        repo.git("add", "a.sh")
        repo.write("a.sh", "echo a\necho b\n")
        result = repo.fix("--staged")
        assert result.returncode == 1
        assert "unstaged changes" in result.stderr
        assert "Copyright" not in repo.git("show", ":a.sh")

    def test_ignores_unstaged_files(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        assert repo.fix("--staged").returncode == 0
        assert not repo.header("a.sh")
