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
PyPI. Each script defines ``setup(client, log)`` at module level and a minimal
``__main__`` guard that calls ``run(setup)``. Module-level constants are safe
to import from verify scripts.
"""

import sys
from collections.abc import Callable
from datetime import datetime

import synnax as sy

HOST = "localhost"
PORT = 9090
USERNAME = "synnax"
PASSWORD = "seldon"


def log(msg: str) -> None:
    ts = datetime.now().strftime("%H:%M:%S.%f")[:-4]
    print(f"{ts} | {msg}")


LogFn = Callable[[str], None]


def run(setup_fn: Callable[[sy.Synnax, LogFn], None]) -> None:
    """Run a setup function with connection handling and error reporting."""
    client = sy.Synnax(
        host=HOST,
        port=PORT,
        username=USERNAME,
        password=PASSWORD,
    )
    try:
        setup_fn(client, log)
    except Exception as e:
        log(f"FAILED: {e}")
        sys.exit(1)
    finally:
        client.close()
