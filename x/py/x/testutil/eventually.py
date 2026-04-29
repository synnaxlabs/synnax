#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from collections.abc import Callable


def assert_eventually(
    condition: Callable[[], object],
    timeout: float = 5.0,
    interval: float = 0.1,
) -> None:
    """Invoke condition repeatedly until it stops raising, or the timeout elapses.

    Useful for asserting against state that becomes consistent asynchronously
    (e.g. distributed indexes, gossip propagation, search indexers). The
    callable may signal "not yet" by raising AssertionError or any other
    exception. Both are swallowed and retried until the deadline, at which
    point the final exception from condition is propagated unchanged.

    Args:
        condition: Callable invoked on each poll. Its return value is ignored.
        timeout: Maximum total time, in seconds, to wait. Defaults to 5.0.
        interval: Delay, in seconds, between retries. Defaults to 0.1.

    Raises:
        Whatever exception condition raises on its final attempt after the
        timeout has elapsed. Most commonly AssertionError.
    """
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            condition()
            return
        except Exception:
            time.sleep(interval)
    condition()
