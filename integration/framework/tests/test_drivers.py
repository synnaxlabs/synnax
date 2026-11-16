#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
from pathlib import Path

import pytest

from framework import drivers
from framework.models import SynnaxConnection

pytestmark = pytest.mark.skipif(
    sys.platform == "win32", reason="the fake Driver is a POSIX script"
)

# Prints its arguments, then stops on a STOP line unless told to ignore it.
FAKE_DRIVER = """#!/usr/bin/env python3
import os
import sys
import time

print(" ".join(sys.argv[1:]), flush=True)
if os.environ.get("FAKE_DRIVER_EXIT"):
    sys.exit(0)
if os.environ.get("FAKE_DRIVER_DEAF"):
    while True:
        time.sleep(0.1)
for line in sys.stdin:
    if line.strip() == "STOP":
        print("stopped", flush=True)
        sys.exit(0)
"""


@pytest.fixture
def binary(tmp_path: Path) -> Path:
    path = tmp_path / "synnax-driver"
    path.write_text(FAKE_DRIVER)
    path.chmod(0o755)
    return path


def start(binary: Path, log_path: Path, stop_timeout: float = 5.0) -> drivers.Process:
    return drivers.Process(
        binary=binary,
        connection=SynnaxConnection(port=9300, username="admin", password="pw"),
        log_path=log_path,
        stop_timeout=stop_timeout,
    )


class TestProcess:
    def test_arguments(self, binary: Path, tmp_path: Path) -> None:
        """Should start a standalone Driver against the Core with its login and NI
        off."""
        log_path = tmp_path / "logs" / "driver.log"
        process = start(binary, log_path)
        process.stop()
        args = log_path.read_text().splitlines()[0]
        assert args.startswith("start --standalone --disable-sig-stop")
        assert "--host localhost --port 9300" in args
        assert "--username admin --password pw" in args
        assert "--secure=false" in args
        assert "--state-file " in args
        assert "--disable-integrations ni" in args

    def test_stop(self, binary: Path, tmp_path: Path) -> None:
        """Should ask the Driver to stop with a STOP line on stdin."""
        log_path = tmp_path / "driver.log"
        process = start(binary, log_path)
        process.stop()
        assert process._popen.returncode == 0
        assert "stopped" in log_path.read_text()

    def test_kill(
        self, binary: Path, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should kill a Driver that does not stop in time."""
        monkeypatch.setenv("FAKE_DRIVER_DEAF", "1")
        process = start(binary, tmp_path / "driver.log", stop_timeout=0.2)
        process.stop()
        assert process._popen.returncode is not None
        assert process._popen.returncode < 0

    def test_exited(
        self, binary: Path, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should stop a Driver that already exited."""
        monkeypatch.setenv("FAKE_DRIVER_EXIT", "1")
        process = start(binary, tmp_path / "driver.log")
        process._popen.wait()
        process.stop()
        assert process._popen.returncode == 0
