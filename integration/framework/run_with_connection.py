#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Launch Python scripts with Synnax connection parameters.

This module serves two roles:

1. **As a subprocess entry point** (``python -m framework.run_with_connection
   <module>``): patches the Synnax client so ``sy.Synnax()`` uses connection
   parameters from environment variables instead of ~/.synnax / system keyring.

2. **As a library** (``from framework.run_with_connection import run_scripts``):
   spawns subprocesses that run the given modules with the connection parameters
   injected via environment variables.
"""

from __future__ import annotations

import os
import subprocess
import sys
import threading
from collections.abc import Callable
from typing import IO

from framework.models import SynnaxConnection
from synnax.options import Options


def _forward(stream: IO[str], module: str, log: Callable[[str], None]) -> None:
    for line in stream:
        log(f"{module}: {line.rstrip()}")


def run_scripts(
    conn: SynnaxConnection,
    modules: list[str],
    *,
    log: Callable[[str], None],
) -> list[subprocess.Popen[str]]:
    """Launches each module as a subprocess with the Synnax connection injected.

    :param conn: Connection parameters to pass to the subprocesses.
    :param modules: Fully-qualified module names, such as
        ``examples.opcua.read_task``.
    :param log: Receives each line the subprocesses write to stdout or stderr.
    :returns: The ``Popen`` handles. The caller must terminate them.
    """
    env = {
        **os.environ,
        "SYNNAX_HOST": conn.server_address,
        "SYNNAX_PORT": str(conn.port),
        "SYNNAX_USERNAME": conn.username,
        "SYNNAX_PASSWORD": conn.password,
        "SYNNAX_SECURE": "1" if conn.secure else "0",
    }
    env.pop("SYNNAX_CA_CERT", None)
    if conn.ca_cert is not None:
        env["SYNNAX_CA_CERT"] = conn.ca_cert
    procs = []
    for module in modules:
        proc = subprocess.Popen(
            [sys.executable, "-u", "-m", "framework.run_with_connection", module],
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            env=env,
        )
        assert proc.stdout is not None
        threading.Thread(
            target=_forward, args=(proc.stdout, module, log), daemon=True
        ).start()
        procs.append(proc)
    return procs


def options_from_env() -> Options:
    """Reads the connection parameters that ``run_scripts`` injects.

    :raises KeyError: If a required variable is not set.
    """
    return Options(
        host=os.environ["SYNNAX_HOST"],
        port=int(os.environ["SYNNAX_PORT"]),
        username=os.environ["SYNNAX_USERNAME"],
        password=os.environ["SYNNAX_PASSWORD"],
        secure=os.environ["SYNNAX_SECURE"] == "1",
        ca_cert=os.environ.get("SYNNAX_CA_CERT"),
    )


if __name__ == "__main__":
    import importlib

    import synnax.synnax as _syn

    _opts = options_from_env()
    vars(_syn)["try_load_options_if_none_provided"] = lambda *a, **k: _opts

    _module = sys.argv[1]
    sys.argv = [_module]  # Clean argv so argparse in imported modules works
    importlib.import_module(_module)
