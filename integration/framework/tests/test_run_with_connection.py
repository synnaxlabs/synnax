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

import pytest

from framework.models import SynnaxConnection
from framework.run_with_connection import run_scripts

SCRIPT = "framework.tests.connection_script"


def run(conn: SynnaxConnection) -> list[str]:
    """Returns the two lines the script writes before it blocks."""
    lines: list[str] = []
    lock = threading.Lock()

    def log(line: str) -> None:
        with lock:
            lines.append(line)

    procs = run_scripts(conn, [SCRIPT], log=log)
    try:
        deadline = time.monotonic() + 30
        while len(lines) < 2 and time.monotonic() < deadline:
            time.sleep(0.01)
    finally:
        for proc in procs:
            proc.kill()
            proc.wait()
    return sorted(lines)


class TestRunScripts:
    """run_scripts forwards the connection and the script output."""

    def test_should_forward_output_before_the_script_exits(self) -> None:
        assert run(SynnaxConnection()) == [
            f"{SCRIPT}: ca_cert=None",
            f"{SCRIPT}: secure=False",
        ]

    def test_should_pass_secure_and_ca_cert(self) -> None:
        assert run(SynnaxConnection(secure=True, ca_cert="/certs/ca.crt")) == [
            f"{SCRIPT}: ca_cert=/certs/ca.crt",
            f"{SCRIPT}: secure=True",
        ]

    def test_should_not_inherit_the_ca_cert_of_the_parent(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setenv("SYNNAX_CA_CERT", "/parent/ca.crt")
        assert run(SynnaxConnection()) == [
            f"{SCRIPT}: ca_cert=None",
            f"{SCRIPT}: secure=False",
        ]
