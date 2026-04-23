#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Shared constants and helpers for migration setup scripts.

Setup scripts run in isolated venvs with version-pinned synnax clients from
PyPI. Only code inside ``if __name__ == "__main__"`` executes in the venv;
module-level constants are safe to import from verify scripts.

Some setup scripts import ``examples`` simulators (OPCUASim, ModbusSim) inside
the ``__main__`` guard. These are standalone servers, not test dependencies. We
use the latest workspace version intentionally. The simulator just needs to
present a valid endpoint, and pinning would prevent us from updating the sims.
"""

import platform
import socket
import subprocess
import sys
import time
from collections.abc import Callable
from datetime import datetime

import synnax as sy

HOST = "localhost"
PORT = 9090
USERNAME = "synnax"
PASSWORD = "seldon"

S = sy.TimeSpan.SECOND
MS = sy.TimeSpan.MILLISECOND


def log(msg: str) -> None:
    ts = datetime.now().strftime("%H:%M:%S.%f")[:-4]
    print(f"{ts} | {msg}")


def kill_port(port: int) -> None:
    """Kill any process listening on the given port."""
    try:
        with socket.create_connection(("localhost", port), timeout=0.5):
            pass
    except (ConnectionRefusedError, OSError):
        return
    log(f"Killing process on port {port}...")
    if platform.system() == "Windows":
        subprocess.run(
            [
                "powershell",
                "-Command",
                f"Stop-Process -Id (Get-NetTCPConnection -LocalPort {port}).OwningProcess -Force",
            ],
            capture_output=True,
        )
    else:
        subprocess.run(
            ["bash", "-c", f"lsof -ti:{port} | xargs kill -9 2>/dev/null"],
            capture_output=True,
        )
    time.sleep(1)


def run(setup_fn: Callable[[sy.Synnax], None]) -> None:
    """Run a setup function with connection handling and error reporting."""
    client = sy.Synnax(
        host=HOST,
        port=PORT,
        username=USERNAME,
        password=PASSWORD,
    )
    try:
        setup_fn(client)
    except Exception as e:
        log(f"FAILED: {e}")
        sys.exit(1)
    finally:
        client.close()
