#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
import time

from framework.models import SynnaxConnection
from framework.run_with_connection import run_scripts

SCRIPT = "framework.tests.connection_script"


def run(conn: SynnaxConnection) -> list[str]:
    lines: list[str] = []
    lock = threading.Lock()

    def log(line: str) -> None:
        with lock:
            lines.append(line)

    for proc in run_scripts(conn, [SCRIPT], log):
        assert proc.wait(timeout=30) == 0
    deadline = time.monotonic() + 5
    while len(lines) < 2 and time.monotonic() < deadline:
        time.sleep(0.01)
    return lines


class TestRunScripts:
    """run_scripts forwards the connection and the script output."""

    def test_should_forward_stdout_and_stderr_to_log(self) -> None:
        lines = run(SynnaxConnection())
        assert sorted(lines) == [f"{SCRIPT}: ca_cert=None", f"{SCRIPT}: secure=False"]

    def test_should_pass_secure_and_ca_cert(self) -> None:
        lines = run(SynnaxConnection(secure=True, ca_cert="/certs/ca.crt"))
        assert sorted(lines) == [
            f"{SCRIPT}: ca_cert=/certs/ca.crt",
            f"{SCRIPT}: secure=True",
        ]
