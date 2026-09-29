#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import multiprocessing
import queue
from collections.abc import Callable

import synnax as sy
from examples.simulators.device_sim import DeviceSim


class BusSim[T](DeviceSim):
    """Base class for simulators of devices on buses and byte streams.

    The server process signals readiness itself, so a subclass can serve a UDP socket,
    a pseudo-terminal, or a CAN bus as well as a TCP port. The server records each
    frame the Driver sends with ``_record``, and the test reads them back with
    ``received`` and ``wait_for``.

    Subclasses set ``_ready`` once the Driver can reach the device.
    """

    def __init__(self, rate: sy.Rate = 50 * sy.Rate.HZ, verbose: bool = False):
        super().__init__(rate=rate, verbose=verbose)
        self._ready = multiprocessing.Event()
        self._records: multiprocessing.Queue[T] = multiprocessing.Queue()
        self._received: list[T] = []

    def _wait_for_ready(self) -> None:
        timer = sy.Timer()
        while timer.elapsed() < self.startup_timeout:
            if self.process is None or not self.process.is_alive():
                code = None if self.process is None else self.process.exitcode
                raise RuntimeError(
                    f"Server process died during startup (exit code: {code})"
                )
            if self._ready.wait(0.1):
                self.log("Server ready")
                return
        raise RuntimeError(f"Server not ready after {self.startup_timeout}")

    async def _send_at_rate(self, send: Callable[[int], None]) -> None:
        """Calls send with the counts 0, 1, 2, and on at the simulator's rate, forever.

        Each call is timed from the start, so a slow call does not drift the rate.
        """
        loop = asyncio.get_running_loop()
        period = 1 / float(self.rate)
        start = loop.time()
        count = 0
        while True:
            send(count)
            count += 1
            await asyncio.sleep(max(0.0, start + count * period - loop.time()))

    def _record(self, item: T) -> None:
        """Records a frame the Driver sent. Call it from the server process."""
        self._records.put(item)

    def received(self) -> list[T]:
        """Returns every frame the Driver has sent, oldest first."""
        while True:
            try:
                self._received.append(self._records.get_nowait())
            except queue.Empty:
                return list(self._received)

    def wait_for(
        self,
        expected: T,
        count: int = 1,
        timeout: sy.TimeSpan = 5 * sy.TimeSpan.SECOND,
    ) -> None:
        """Blocks until the Driver has sent expected at least count times.

        :raises AssertionError: If expected falls short when the timeout expires.
        """
        timer = sy.Timer()
        seen = self.received().count(expected)
        while seen < count and (remaining := timeout - timer.elapsed()) > 0:
            try:
                item = self._records.get(timeout=remaining.seconds)
            except queue.Empty:
                break
            self._received.append(item)
            seen += item == expected
        if seen < count:
            raise AssertionError(
                f"{self.device_name} received {expected!r} {seen} of {count} times "
                f"within {timeout}. Last received: {self._received[-3:]!r}"
            )
