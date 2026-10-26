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

SCRIPT = Path(__file__).resolve().parents[1] / "run_preview_command.sh"

# Answers each status query with the next line of statuses, InProgress once they run
# out, and no invocation for the line "missing".
FAKE_AWS = """#!/usr/bin/env bash
echo "$*" >> "$FAKE_DIR/calls"
case "$*" in
    *send-command*) echo command-1 ;;
    *"--query Status"*)
        status=$(head -n 1 "$FAKE_DIR/statuses")
        sed -i.bak 1d "$FAKE_DIR/statuses"
        if [[ $status == missing ]]; then
            echo "InvocationDoesNotExist" >&2
            exit 254
        fi
        echo "${status:-InProgress}"
        ;;
    *) echo "host output" ;;
esac
"""


class Host:
    """A fake AWS CLI that plays back command statuses."""

    def __init__(self, path: Path) -> None:
        self.path = path
        bin_dir = path / "bin"
        bin_dir.mkdir()
        for name, body in (("aws", FAKE_AWS), ("sleep", "#!/bin/sh\n")):
            fake = bin_dir / name
            fake.write_text(body)
            fake.chmod(0o755)
        (path / "calls").touch()
        self.env = {
            **os.environ,
            "PATH": f"{bin_dir}{os.pathsep}{os.environ['PATH']}",
            "FAKE_DIR": str(path),
            "COMMAND": "/usr/local/bin/preview up 12 image",
            "INSTANCE": "i-123",
        }

    def run(self, *statuses: str) -> subprocess.CompletedProcess[str]:
        (self.path / "statuses").write_text("".join(f"{s}\n" for s in statuses))
        return subprocess.run(
            [str(SCRIPT)], env=self.env, capture_output=True, text=True
        )

    def calls(self) -> list[str]:
        return (self.path / "calls").read_text().splitlines()


@pytest.fixture
def host(tmp_path: Path) -> Host:
    return Host(tmp_path)


class TestRunPreviewCommand:
    """Tests for sending a command to the preview host and waiting for it."""

    def test_success(self, host: Host) -> None:
        result = host.run("InProgress", "Success")
        assert result.returncode == 0
        assert "host output" in result.stdout
        assert "Status: Success" in result.stdout
        send = host.calls()[0]
        assert "--instance-ids i-123" in send
        params = json.loads(send.split("--parameters ")[1].split(" --timeout")[0])
        assert params == {"commands": ["/usr/local/bin/preview up 12 image"]}

    def test_failure(self, host: Host) -> None:
        result = host.run("Failed")
        assert result.returncode != 0
        assert "Status: Failed" in result.stdout

    def test_missing_invocation(self, host: Host) -> None:
        result = host.run("missing", "Success")
        assert result.returncode == 0
        assert "Status: Success" in result.stdout

    def test_timeout(self, host: Host) -> None:
        result = host.run()
        assert result.returncode != 0
        assert "Status: InProgress" in result.stdout
        polls = [c for c in host.calls() if "--query Status" in c]
        assert len(polls) == 120

    @pytest.mark.parametrize("missing", ["COMMAND", "INSTANCE"])
    def test_missing_variable(self, host: Host, missing: str) -> None:
        del host.env[missing]
        result = host.run("Success")
        assert result.returncode != 0
        assert missing in result.stderr
        assert host.calls() == []
