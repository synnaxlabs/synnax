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
import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "dispatch_agent_task.sh"

FAKE_GH = """#!/bin/bash
printf '%s\\x1f' "$@" >> "$GH_CALLS"
printf '\\x1e' >> "$GH_CALLS"
if [ "$1" = run ]; then
    while [ "$1" != --jq ]; do shift; done
    jq -r "$2" <<< "$GH_RUNS"
fi
"""


class FakeGitHub:
    """Runs `dispatch_agent_task.sh` with fake `gh` and `sleep` binaries."""

    def __init__(self, path: Path) -> None:
        bin_dir = path / "bin"
        bin_dir.mkdir()
        (bin_dir / "gh").write_text(FAKE_GH)
        (bin_dir / "sleep").write_text("#!/bin/bash\n")
        for f in bin_dir.iterdir():
            f.chmod(0o755)
        self.calls = path / "calls"
        self.calls.touch()
        self.env = {
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "GH_CALLS": str(self.calls),
            "GH_RUNS": "[]",
        }

    def runs(self, *runs: tuple[str, str, str]) -> None:
        self.env["GH_RUNS"] = json.dumps(
            [{"displayTitle": t, "createdAt": c, "url": u} for t, c, u in runs]
        )

    def run(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["bash", str(SCRIPT), *args], env=self.env, capture_output=True, text=True
        )

    def gh_calls(self) -> list[list[str]]:
        calls = self.calls.read_text().split("\x1e")[:-1]
        return [call.split("\x1f")[:-1] for call in calls]


@pytest.fixture
def fake_github(tmp_path: Path) -> FakeGitHub:
    return FakeGitHub(tmp_path)
