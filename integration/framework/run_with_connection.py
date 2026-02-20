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
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from framework.test_case import SynnaxConnection


def run_scripts(
    conn: SynnaxConnection,
    modules: list[str],
) -> list[subprocess.Popen[bytes]]:
    """Launch each module as a subprocess with Synnax connection injected.

    Args:
        conn: Connection parameters to pass to the subprocesses.
        modules: Fully-qualified module names (e.g. ``examples.opcua.read_task``).

    Returns:
        A list of ``Popen`` handles. The caller is responsible for terminating them.
    """
    env = {
        **os.environ,
        "SYNNAX_HOST": conn.server_address,
        "SYNNAX_PORT": str(conn.port),
        "SYNNAX_USERNAME": conn.username,
        "SYNNAX_PASSWORD": conn.password,
    }
    return [
        subprocess.Popen(
            [sys.executable, "-m", "framework.run_with_connection", module],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
            env=env,
        )
        for module in modules
    ]


if __name__ == "__main__":
    import importlib

    import synnax.synnax as _syn
    from synnax.options import SynnaxOptions

    _opts = SynnaxOptions(
        host=os.environ["SYNNAX_HOST"],
        port=int(os.environ["SYNNAX_PORT"]),
        username=os.environ["SYNNAX_USERNAME"],
        password=os.environ["SYNNAX_PASSWORD"],
    )
    _syn.try_load_options_if_none_provided = lambda *a, **k: _opts

    importlib.import_module(sys.argv[1])
