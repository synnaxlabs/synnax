#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import socket
import sys
from pathlib import Path

import pytest

from framework import cores
from framework.models import SynnaxConnection

pytestmark = pytest.mark.skipif(
    sys.platform == "win32", reason="the fake Core is a POSIX script"
)

# Prints its arguments, then the license and ready lines unless told otherwise, and
# runs until interrupted.
FAKE_CORE = """#!/usr/bin/env python3
import os
import signal
import sys
import time

print(" ".join(sys.argv[1:]), flush=True)
if os.environ.get("FAKE_CORE_EXIT"):
    sys.exit(3)
if not os.environ.get("FAKE_CORE_UNLICENSED"):
    print("license active", flush=True)
print("Synnax is running and available at here", flush=True)
signal.signal(signal.SIGINT, lambda *_: sys.exit(0))
while True:
    time.sleep(0.1)
"""


@pytest.fixture
def binary(tmp_path: Path) -> Path:
    path = tmp_path / "synnax"
    path.write_text(FAKE_CORE)
    path.chmod(0o755)
    return path


def free(port: int) -> bool:
    try:
        with socket.create_connection(("localhost", port), timeout=0.2):
            return False
    except OSError:
        return True


@pytest.fixture
def first_port() -> int:
    """A port with nothing listening on the next two ports."""
    return next(p for p in range(19300, 19400, 3) if free(p + 1) and free(p + 2))


class TestStart:
    def test_start_and_stop(
        self, binary: Path, first_port: int, tmp_path: Path
    ) -> None:
        """Should start Cores on the ports after the first, with NI off, and stop
        them."""
        first = SynnaxConnection(port=first_port)
        processes = cores.start(
            binary=binary, first=first, count=2, log_dir=tmp_path / "cores"
        )
        try:
            ports = [p.core.connection.port for p in processes]
            assert ports == [first_port + 1, first_port + 2]
            log = processes[0].core.log_path.read_text()
            assert f"-l localhost:{first_port + 1}" in log
            assert "--username synnax --password seldon" in log
            assert "--disable-integrations ni" in log
        finally:
            cores.stop(processes)
        assert all(p._popen.returncode == 0 for p in processes)

    def test_port_taken(self, binary: Path, first_port: int, tmp_path: Path) -> None:
        """Should refuse to start when a Core's port is taken."""
        with socket.socket() as s:
            s.bind(("localhost", first_port + 1))
            s.listen()
            with pytest.raises(RuntimeError, match=f"Port {first_port + 1}"):
                cores.start(
                    binary=binary,
                    first=SynnaxConnection(port=first_port),
                    count=1,
                    log_dir=tmp_path,
                )

    def test_core_exits(
        self,
        binary: Path,
        first_port: int,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        """Should raise with the log path when a Core exits before it runs."""
        monkeypatch.setenv("FAKE_CORE_EXIT", "1")
        with pytest.raises(RuntimeError, match="exited with code 3"):
            cores.start(
                binary=binary,
                first=SynnaxConnection(port=first_port),
                count=1,
                log_dir=tmp_path,
            )

    def test_login(self, binary: Path, first_port: int, tmp_path: Path) -> None:
        """Should start each Core with the first Core's login."""
        first = SynnaxConnection(port=first_port, username="admin", password="pw")
        processes = cores.start(binary=binary, first=first, count=1, log_dir=tmp_path)
        try:
            log = processes[0].core.log_path.read_text()
            assert "--username admin --password pw" in log
            assert processes[0].core.connection.username == "admin"
        finally:
            cores.stop(processes)

    def test_unlicensed(
        self,
        binary: Path,
        first_port: int,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        """Should raise when a Core runs without a license."""
        monkeypatch.setenv("FAKE_CORE_UNLICENSED", "1")
        with pytest.raises(RuntimeError, match="has no active license"):
            cores.start(
                binary=binary,
                first=SynnaxConnection(port=first_port),
                count=1,
                log_dir=tmp_path,
            )

    def test_secure(self, binary: Path, first_port: int, tmp_path: Path) -> None:
        """Should refuse to start Cores beside a secure Core."""
        with pytest.raises(ValueError, match="cannot test a secure Core"):
            cores.start(
                binary=binary,
                first=SynnaxConnection(port=first_port, secure=True),
                count=1,
                log_dir=tmp_path,
            )
