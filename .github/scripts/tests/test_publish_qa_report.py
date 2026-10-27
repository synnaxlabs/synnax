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
import sys
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "publish_qa_report.py"

# Records its arguments and the body file, then prints an issue URL.
FAKE_GH = """#!/usr/bin/env bash
echo "$*" > "$FAKE_DIR/args"
while [ $# -gt 0 ]; do
    if [ "$1" = --body-file ]; then cp "$2" "$FAKE_DIR/body"; fi
    shift
done
echo https://github.com/synnaxlabs/synnax/issues/1
"""

REPORT = """# Console QA: 1 finding

## High: plot legend overlaps the axis

![legend](images/legend.png)
"""


def git(cwd: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args], cwd=cwd, check=True, capture_output=True, text=True
    ).stdout.strip()


@pytest.fixture
def env(tmp_path: Path) -> dict[str, str]:
    origin = tmp_path / "origin.git"
    git(tmp_path, "init", "-q", "--bare", str(origin))
    clone = tmp_path / "clone"
    git(tmp_path, "init", "-q", str(clone))
    git(clone, "remote", "add", "origin", str(origin))
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    gh = bin_dir / "gh"
    gh.write_text(FAKE_GH)
    gh.chmod(0o755)
    base = {k: v for k, v in os.environ.items() if not k.startswith("GITHUB_")}
    return {
        **base,
        "PATH": f"{bin_dir}{os.pathsep}{base['PATH']}",
        "FAKE_DIR": str(tmp_path),
        "GIT_AUTHOR_NAME": "ci",
        "GIT_AUTHOR_EMAIL": "ci@synnaxlabs.com",
        "GIT_COMMITTER_NAME": "ci",
        "GIT_COMMITTER_EMAIL": "ci@synnaxlabs.com",
        "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        "GITHUB_RUN_ID": "42",
    }


@pytest.fixture
def report(tmp_path: Path) -> Path:
    qa = tmp_path / "qa"
    (qa / "images").mkdir(parents=True)
    (qa / "report.md").write_text(REPORT)
    (qa / "images" / "legend.png").write_bytes(b"png")
    return qa


def publish(
    env: dict[str, str], qa: Path, *args: str
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, str(SCRIPT), str(qa), *args],
        cwd=Path(env["FAKE_DIR"]) / "clone",
        env=env,
        capture_output=True,
        text=True,
    )


class TestPublish:
    def test_should_open_an_issue_with_hosted_images(
        self, env: dict[str, str], report: Path, tmp_path: Path
    ) -> None:
        done = publish(env, report)
        assert done.returncode == 0, done.stderr
        assert done.stdout.strip() == "https://github.com/synnaxlabs/synnax/issues/1"
        args = (tmp_path / "args").read_text()
        assert "--title Console QA: 1 finding" in args
        assert "--label console-qa" in args
        body = (tmp_path / "body").read_text()
        commit = git(tmp_path / "origin.git", "rev-parse", "agent-assets")
        url = f"https://raw.githubusercontent.com/synnaxlabs/synnax/{commit}"
        assert f"]({url}/console-qa-42/legend.png)" in body
        assert "actions/runs/42" in body
        tree = git(tmp_path / "origin.git", "ls-tree", "-r", "--name-only", commit)
        assert tree == "console-qa-42/legend.png"

    def test_should_keep_earlier_images_on_the_assets_branch(
        self, env: dict[str, str], report: Path, tmp_path: Path
    ) -> None:
        assert publish(env, report).returncode == 0
        env["GITHUB_RUN_ID"] = "43"
        done = publish(env, report)
        assert done.stdout == "https://github.com/synnaxlabs/synnax/issues/1\n"
        commit = git(tmp_path / "origin.git", "rev-parse", "agent-assets")
        tree = git(tmp_path / "origin.git", "ls-tree", "-r", "--name-only", commit)
        assert tree.split() == ["console-qa-42/legend.png", "console-qa-43/legend.png"]

    def test_should_retry_a_rejected_push(
        self, env: dict[str, str], report: Path, tmp_path: Path
    ) -> None:
        hook = tmp_path / "origin.git" / "hooks" / "pre-receive"
        reject = tmp_path / "reject"
        hook.write_text(f'#!/bin/sh\nif rm "{reject}" 2> /dev/null; then exit 1; fi\n')
        hook.chmod(0o755)
        reject.touch()
        done = publish(env, report)
        assert done.returncode == 0, done.stderr
        assert not reject.exists()
        assert git(tmp_path / "origin.git", "branch", "--list") == "agent-assets"

    def test_should_write_the_issue_to_a_file_in_a_dry_run(
        self, env: dict[str, str], report: Path, tmp_path: Path
    ) -> None:
        done = publish(env, report, "--dry-run")
        assert done.returncode == 0, done.stderr
        issue = (report / "issue.md").read_text()
        assert issue.startswith("# Console QA: 1 finding\n")
        assert "](images/legend.png)" in issue
        assert not (tmp_path / "args").exists()
        assert git(tmp_path / "origin.git", "branch", "--list") == ""

    def test_should_reject_a_report_without_a_title(
        self, env: dict[str, str], report: Path
    ) -> None:
        (report / "report.md").write_text("No title\n")
        done = publish(env, report, "--dry-run")
        assert done.returncode != 0
        assert "# <title>" in done.stderr

    def test_should_reject_a_link_to_a_missing_image(
        self, env: dict[str, str], report: Path
    ) -> None:
        (report / "images" / "legend.png").unlink()
        done = publish(env, report, "--dry-run")
        assert done.returncode != 0
        assert "images/legend.png" in done.stderr
