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
from scripts_common import FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "dispatch_agent_task.sh"


class FakeGitHub:
    """Runs `dispatch_agent_task.sh` with fake `gh` and `sleep` binaries."""

    def __init__(self, path: Path) -> None:
        self.gh = FakeGH(path)
        self.gh.serve("api */dispatches")
        self.gh.serve("run list", [])
        sleep = self.gh.bin_dir / "sleep"
        sleep.write_text("#!/bin/bash\n")
        sleep.chmod(0o755)
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
        }

    def runs(self, *runs: tuple[str, str, str]) -> None:
        self.gh.serve(
            "run list",
            [{"displayTitle": t, "createdAt": c, "url": u} for t, c, u in runs],
        )

    def run(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["bash", str(SCRIPT), *args], env=self.env, capture_output=True, text=True
        )


@pytest.fixture
def fake_github(tmp_path: Path) -> FakeGitHub:
    return FakeGitHub(tmp_path)


class CopyrightRepo:
    """A Git repository with the copyright scripts and the license template."""

    def __init__(self, path: Path) -> None:
        self.path = path
        root = Path(__file__).resolve().parents[2]
        for f in [
            "scripts/check_copyrights.sh",
            "scripts/update_copyrights.sh",
            "scripts/common/copyright.sh",
            "licenses/headers/template.txt",
        ]:
            (path / f).parent.mkdir(parents=True, exist_ok=True)
            shutil.copy(root / f, path / f)
        self.year = int(self.run("date", "+%Y").stdout)
        self.run("git", "init", "-q")

    def run(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            args, cwd=self.path, capture_output=True, text=True, check=False
        )

    def write(self, name: str, year: int, body: str = "package x\n") -> None:
        """Writes name with a canonical Go header for year, then body."""
        template = (self.path / "licenses/headers/template.txt").read_text()
        header = "\n".join(
            f"// {line}" if line else "//" for line in template.rstrip("\n").split("\n")
        ).replace("{{YEAR}}", str(year))
        (self.path / name).write_text(f"{header}\n\n{body}")
        self.run("git", "add", name)

    def year_of(self, name: str) -> int:
        return int((self.path / name).read_text().split()[2])


@pytest.fixture
def copyright_repo(tmp_path: Path) -> CopyrightRepo:
    return CopyrightRepo(tmp_path)
