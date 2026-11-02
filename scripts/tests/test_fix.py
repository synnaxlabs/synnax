#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
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

    def fix(
        self, *args: str, cwd: Path | None = None
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["bash", str(self.path / "scripts/fix.sh"), *args],
            cwd=cwd or self.path,
            env=self.env,
            capture_output=True,
            text=True,
        )

    def install_hooks(self) -> None:
        shutil.copytree(ROOT / ".githooks", self.path / ".githooks")
        package = json.loads((ROOT / "package.json").read_text())
        subprocess.run(
            package["scripts"]["prepare"],
            shell=True,
            cwd=self.path,
            env=self.env,
            check=True,
        )

    def commit(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["git", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "c"],
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
    (tmp_path / "repo").mkdir()
    return Repo(tmp_path / "repo")


def clang_format(repo: Repo, *, body: str) -> None:
    """Puts scripts/clang_format.sh and a clang-format that runs body on PATH."""
    shutil.copy(ROOT / "scripts/clang_format.sh", repo.path / "scripts")
    bin = repo.path.parent / "bin"
    bin.mkdir()
    exe = bin / "clang-format"
    exe.write_text(f'#!/bin/bash\n[ "$1" = --version ] && exit 0\n{body}\n')
    exe.chmod(0o755)
    repo.env["PATH"] = f"{bin}:{repo.env['PATH']}"


class TestFix:
    """Runs fix.sh in each of its modes."""

    def test_should_add_headers_to_untracked_files(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        assert repo.fix().returncode == 0
        assert repo.header("a.sh")

    def test_should_stage_fixes_to_staged_files(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        repo.git("add", "a.sh")
        result = repo.fix("--staged")
        assert result.returncode == 0
        assert "fixed a.sh" in result.stdout
        assert "Copyright" in repo.git("show", ":a.sh")
        assert repo.git("diff", "--name-only") == ""

    def test_should_fix_staged_copy_of_partially_staged_files(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        repo.git("add", "a.sh")
        repo.write("a.sh", "echo a\necho b\n")
        assert repo.fix("--staged").returncode == 0
        staged = repo.git("show", ":a.sh")
        assert "Copyright" in staged
        assert "echo b" not in staged
        assert repo.git("diff", "-U0", "a.sh").splitlines()[-1] == "+echo b"

    def test_should_handle_quoted_file_names(self, repo: Repo) -> None:
        repo.write("é.sh", "echo a\n")
        repo.git("add", "é.sh")
        repo.write("é.sh", "echo a\necho b\n")
        assert repo.fix("--staged").returncode == 0
        staged = repo.git("show", ":é.sh")
        assert "Copyright" in staged
        assert "echo b" not in staged

    def test_should_fix_staged_copy_when_working_copy_is_fixed(
        self, repo: Repo
    ) -> None:
        repo.write("a.sh", "echo a\n")
        repo.git("add", "a.sh")
        repo.fix("a.sh")
        assert repo.fix("--staged").returncode == 0
        assert "Copyright" in repo.git("show", ":a.sh")
        assert repo.git("diff", "--name-only") == ""

    def test_should_keep_the_working_copy_when_a_fixer_fails(self, repo: Repo) -> None:
        (repo.path / "scripts/update_copyrights.sh").write_text("exit 1\n")
        repo.write("a.sh", "echo a\n")
        repo.git("add", "a.sh")
        repo.write("a.sh", "echo a\necho b\n")
        assert repo.fix("--staged").returncode == 1
        assert (repo.path / "a.sh").read_text() == "echo a\necho b\n"

    def test_should_pass_when_a_fixer_fails_on_the_working_copy(
        self, repo: Repo
    ) -> None:
        clang_format(
            repo, body='for f in "${@:2}"; do ! grep -q broken "$f" || exit 1; done'
        )
        repo.write("a.cpp", "int a;\n")
        repo.git("add", "a.cpp")
        repo.write("a.cpp", "int a;\nbroken\n")
        assert repo.fix("--staged").returncode == 0
        assert "Copyright" in repo.git("show", ":a.cpp")
        assert (repo.path / "a.cpp").read_text().endswith("int a;\nbroken\n")

    def test_should_resolve_paths_from_the_caller(self, repo: Repo) -> None:
        (repo.path / "sub").mkdir()
        repo.write("sub/a.sh", "echo a\n")
        assert repo.fix("a.sh", cwd=repo.path / "sub").returncode == 0
        assert repo.header("sub/a.sh")

    def test_should_keep_unstaged_mode_changes(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        repo.git("add", "a.sh")
        (repo.path / "a.sh").chmod(0o755)
        assert repo.fix("--staged").returncode == 0
        assert (repo.path / "a.sh").stat().st_mode & 0o111
        assert "Copyright" in repo.git("show", ":a.sh")

    def test_should_keep_unstaged_symlinks(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        repo.write("b.sh", "echo b\n")
        repo.git("add", "a.sh")
        (repo.path / "a.sh").unlink()
        (repo.path / "a.sh").symlink_to("b.sh")
        assert repo.fix("--staged").returncode == 0
        assert (repo.path / "a.sh").is_symlink()
        assert "Copyright" in repo.git("show", ":a.sh")
        assert not repo.header("b.sh")

    def test_should_pass_cpp_file_names_whole(self, repo: Repo) -> None:
        clang_format(repo, body='for f in "${@:2}"; do echo \'// ok\' >> "$f"; done')
        repo.write("a b.cpp", "int a;\n")
        assert repo.fix("a b.cpp").returncode == 0
        assert (repo.path / "a b.cpp").read_text().endswith("// ok\n")

    def test_should_fail_when_clang_format_fails(self, repo: Repo) -> None:
        clang_format(repo, body="exit 1")
        repo.write("a.cpp", "int a;\n")
        assert repo.fix("a.cpp").returncode != 0

    def test_should_ignore_unstaged_files(self, repo: Repo) -> None:
        repo.write("a.sh", "echo a\n")
        assert repo.fix("--staged").returncode == 0
        assert not repo.header("a.sh")


class TestPreCommitHook:
    """Commits through the pre-commit hook."""

    def test_should_commit_fixes(self, repo: Repo) -> None:
        repo.install_hooks()
        repo.write("a.sh", "echo a\n")
        repo.git("add", "a.sh")
        assert repo.commit().returncode == 0
        assert "Copyright" in repo.git("show", "HEAD:a.sh")
        assert repo.git("status", "--porcelain", "a.sh") == ""

    def test_should_commit_only_the_staged_copy(self, repo: Repo) -> None:
        repo.install_hooks()
        repo.write("a.sh", "echo a\n")
        repo.git("add", "a.sh")
        repo.write("a.sh", "echo a\necho b\n")
        assert repo.commit().returncode == 0
        committed = repo.git("show", "HEAD:a.sh")
        assert "Copyright" in committed
        assert "echo b" not in committed
        assert repo.git("diff", "-U0", "a.sh").splitlines()[-1] == "+echo b"
