#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
from collections.abc import Awaitable


async def ticks_while[T](aw: Awaitable[T]) -> tuple[T, int]:
    """Awaits aw and counts the ticks a concurrent task gets on the event loop. A call
    that blocks the loop leaves the count near zero.
    """
    ticks = 0

    async def tick() -> None:
        nonlocal ticks
        while True:
            await asyncio.sleep(0.01)
            ticks += 1

    ticker = asyncio.create_task(tick())
    try:
        return await aw, ticks
    finally:
        ticker.cancel()
