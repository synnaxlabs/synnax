#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import contextlib
from collections.abc import Generator

from freighter.websocket import ConnectionClosedError


def is_websocket_error(error: BaseException) -> bool:
    """Whether error is a dropped WebSocket connection, such as a keepalive ping
    timeout (close code 1011), which the telemetry threads retry.
    """
    return isinstance(error, ConnectionClosedError)


@contextlib.contextmanager
def suppress_websocket_errors() -> Generator[None, None, None]:
    """Suppress dropped WebSocket connections, re-raising everything else."""
    try:
        yield
    except ConnectionClosedError:
        pass
