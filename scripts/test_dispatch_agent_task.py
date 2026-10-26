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

SCRIPT = Path(__file__).resolve().parent / "dispatch_agent_task.sh"

FAKE_GH = """#!/bin/bash
printf '%s\\x1f' "$@" >> "$GH_CALLS"
printf '\\x1e' >> "$GH_CALLS"
if [ "$1" = run ]; then
    while [ "$1" != --jq ]; do shift; done
    jq -r "$2" <<< "$GH_RUNS"
fi
"""

FUTURE = "2999-01-01T00:00:00Z"
PAST = "2000-01-01T00:00:00Z"


class Env:
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
def env(tmp_path: Path) -> Env:
    return Env(tmp_path)


@pytest.mark.parametrize(
    "args",
    [
        [],
        ["SY-1; rm -rf /"],
        ["sy-1"],
        ["--agent", "gpt", "SY-1"],
        ["--force", "SY-1"],
    ],
)
def test_rejects_bad_input(env: Env, args: list[str]) -> None:
    result = env.run(*args)
    assert result.returncode == 2
    assert env.gh_calls() == []


def test_dry_run_prints_payload_and_sends_nothing(env: Env) -> None:
    result = env.run("--dry-run", "--agent", "codex", "SY-42")
    assert result.returncode == 0
    assert "issue=SY-42" in result.stdout
    assert "agent=codex" in result.stdout
    assert "prompt=Work on Linear issue SY-42." in result.stdout
    assert env.gh_calls() == []


def test_sends_dispatch_and_prints_matching_run(env: Env) -> None:
    env.runs(
        ("agent-task SY-42", PAST, "old"),
        ("agent-task SY-420", FUTURE, "other-issue"),
        ("agent-task SY-42", FUTURE, "match"),
    )
    result = env.run("SY-42")
    assert result.returncode == 0
    assert "Run: match" in result.stdout
    send = env.gh_calls()[0]
    assert send[:4] == ["api", "repos/synnaxlabs/synnax/dispatches", "--method", "POST"]
    assert "event_type=agent-task" in send
    assert "client_payload[issue]=SY-42" in send
    assert "client_payload[agent]=claude" in send
    assert "client_payload[output]=pr" in send


def test_prints_actions_link_when_no_run_matches(env: Env) -> None:
    env.runs(("agent-task SY-7", FUTURE, "other-issue"))
    result = env.run("SY-42")
    assert result.returncode == 0
    assert "No run found yet" in result.stdout
    assert len(env.gh_calls()) == 13
