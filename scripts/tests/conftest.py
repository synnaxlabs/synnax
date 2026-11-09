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
