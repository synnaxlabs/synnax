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
from collections.abc import Callable

from framework.watchdog import Slots, Timeout, Watchdog

POLL = 0.01
WAIT = 5.0


def until(condition: Callable[[], bool]) -> bool:
    deadline = time.monotonic() + WAIT
    while time.monotonic() < deadline:
        if condition():
            return True
        time.sleep(POLL)
    return False


class Worker:
    """A thread that loops until released and records a Timeout it receives."""

    def __init__(self, *, survive_timeout: bool = False) -> None:
        self.released = threading.Event()
        self.timed_out = threading.Event()
        self._survive_timeout = survive_timeout
        self.thread = threading.Thread(target=self._run, daemon=True)
        self.thread.start()

    def _run(self) -> None:
        while not self.released.is_set():
            try:
                while not self.released.is_set():
                    time.sleep(POLL)
            except Timeout:
                self.timed_out.set()
                if not self._survive_timeout:
                    return


class TestWatchdog:
    def test_interrupt_past_deadline(self) -> None:
        """Should raise Timeout in a thread that runs past its deadline."""
        watchdog = Watchdog(grace=WAIT)
        worker = Worker()
        watchdog.watch(worker.thread, timeout=0.05)
        assert until(lambda: worker.thread in watchdog.check())
        worker.thread.join(WAIT)
        assert worker.timed_out.is_set()
        assert not worker.thread.is_alive()

    def test_no_interrupt_before_deadline(self) -> None:
        """Should leave a thread alone before its deadline."""
        watchdog = Watchdog(grace=WAIT)
        worker = Worker()
        watchdog.watch(worker.thread, timeout=WAIT * 10)
        assert watchdog.check() == []
        worker.released.set()
        worker.thread.join(WAIT)
        assert not worker.timed_out.is_set()

    def test_no_interrupt_after_unwatch(self) -> None:
        """Should not interrupt a thread it no longer watches."""
        watchdog = Watchdog(grace=WAIT)
        worker = Worker()
        watchdog.watch(worker.thread, timeout=0)
        watchdog.unwatch(worker.thread)
        assert watchdog.check() == []
        worker.released.set()
        worker.thread.join(WAIT)
        assert not worker.timed_out.is_set()

    def test_abandon_after_grace(self) -> None:
        """Should report a thread that survives its Timeout past the grace period as
        abandoned."""
        watchdog = Watchdog(grace=0.05)
        worker = Worker(survive_timeout=True)
        watchdog.watch(worker.thread, timeout=0)
        assert worker.thread in watchdog.check()
        assert not watchdog.abandoned(worker.thread)
        assert until(lambda: watchdog.abandoned(worker.thread))
        assert worker.timed_out.is_set()
        worker.released.set()
        worker.thread.join(WAIT)
        assert not watchdog.abandoned(worker.thread)

    def test_no_interrupt_after_disarm(self) -> None:
        """Should not interrupt a disarmed thread, but still abandon it."""
        watchdog = Watchdog(grace=0.05)
        worker = Worker()
        watchdog.watch(worker.thread, timeout=0)
        watchdog.disarm(worker.thread)
        assert watchdog.check() == []
        assert until(lambda: watchdog.abandoned(worker.thread))
        worker.released.set()
        worker.thread.join(WAIT)
        assert not worker.timed_out.is_set()


class TestSlots:
    def test_free_once(self) -> None:
        """Should free a holder's slot only once, so a second free does not add a
        slot."""
        slots = Slots(1)
        holder = threading.Thread(target=slots.acquire)
        holder.start()
        holder.join(WAIT)
        waiter = threading.Thread(target=slots.acquire, daemon=True)
        waiter.start()
        waiter.join(0.05)
        assert waiter.is_alive()
        slots.free(holder)
        slots.free(holder)
        waiter.join(WAIT)
        assert not waiter.is_alive()
        third = threading.Thread(target=slots.acquire, daemon=True)
        third.start()
        third.join(0.05)
        assert third.is_alive()
        slots.free(waiter)
        third.join(WAIT)
