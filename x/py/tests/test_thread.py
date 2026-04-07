#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio

from x.thread import AsyncThread


class TestAsyncThread:
    def test_runs_async_code(self) -> None:
        results: list[int] = []

        class Worker(AsyncThread):
            async def run_async(self) -> None:
                await asyncio.sleep(0.01)
                results.append(42)

        t = Worker()
        t.start()
        t.join(timeout=5)
        assert results == [42]

    def test_cleans_up_event_loop(self) -> None:
        class Worker(AsyncThread):
            async def run_async(self) -> None:
                pass

        t = Worker()
        t.start()
        t.join(timeout=5)
        assert t.loop.is_closed()
