#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
import json
import os
import re
import subprocess
from pathlib import Path

import pytest

DEPLOY = Path(__file__).resolve().parents[3] / "deploy" / "preview"
SCRIPT = DEPLOY / "install.sh"

FAKE_TERRAFORM = """#!/usr/bin/env bash
case "$*" in
    *instance_id) echo i-123 ;;
    *host_env) printf 'PREVIEW_MAX=25\\nPREVIEW_MEMORY=256m\\n' ;;
esac
"""

# Saves the SSM command and reports that it succeeded.
FAKE_AWS = """#!/usr/bin/env bash
case "$*" in
    *send-command*)
        while [[ $1 != --parameters ]]; do shift; done
        echo "$2" > "$FAKE_DIR/parameters"
        echo command-1
        ;;
    *"--query Status"*) echo Success ;;
    *) echo "host output" ;;
esac
"""


@pytest.fixture
def sent(tmp_path: Path) -> str:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    for name, body in (
        ("terraform", FAKE_TERRAFORM),
        ("aws", FAKE_AWS),
        ("sleep", "#!/bin/sh\n"),
    ):
        fake = bin_dir / name
        fake.write_text(body)
        fake.chmod(0o755)
    env = {
        **os.environ,
        "PATH": f"{bin_dir}{os.pathsep}{os.environ['PATH']}",
        "FAKE_DIR": str(tmp_path),
    }
    result = subprocess.run([str(SCRIPT)], env=env, capture_output=True, text=True)
    assert result.returncode == 0, result.stderr
    parameters = json.loads((tmp_path / "parameters").read_text())
    command: str = parameters["commands"][0]
    return command


def decoded(command: str, target: str) -> str:
    match = re.search(rf"echo (\S+) \| base64 -d > {re.escape(target)}\n", command)
    assert match is not None, f"{target} not written"
    return base64.b64decode(match.group(1)).decode()


class TestInstall:
    """Tests for installing the preview script and settings on the running host."""

    def test_should_write_the_applied_settings(self, sent: str) -> None:
        assert decoded(sent, "/etc/synnax-preview.env") == (
            "PREVIEW_MAX=25\nPREVIEW_MEMORY=256m\n"
        )

    def test_should_write_this_checkouts_script(self, sent: str) -> None:
        script = decoded(sent, "/usr/local/bin/preview.new")
        assert script == (DEPLOY / "preview.sh").read_text()

    def test_should_check_the_script_before_replacing_the_old_one(
        self, sent: str
    ) -> None:
        lines = sent.splitlines()
        assert lines[0] == "set -e"
        check = lines.index("bash -n /usr/local/bin/preview.new")
        swap = lines.index("mv /usr/local/bin/preview.new /usr/local/bin/preview")
        assert check < swap

    def test_should_upgrade_and_smoke_test_the_host(self, sent: str) -> None:
        lines = sent.splitlines()
        swap = lines.index("mv /usr/local/bin/preview.new /usr/local/bin/preview")
        assert lines[swap + 1 : swap + 4] == [
            "/usr/local/bin/preview upgrade",
            "/usr/local/bin/preview mark 1 none 0",
            "/usr/local/bin/preview down 1",
        ]
