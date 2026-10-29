#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Seconds each runner has been idle or busy, else since its instance launched."""

import threading
from collections.abc import Callable

import synnax as sy

from common import (
    IDLE,
    STOP_TIMEOUT,
    Channels,
    channel_name,
    instance_for,
    uptime_channels,
    warn,
)
from publisher import Publisher

INDEX = "state_uptime_time"
TICK = 1 * sy.TimeSpan.SECOND


class Uptime:
    """Ticks every runner's count once a second from a daemon thread, so the counts
    read as seconds rather than stepping by the poll interval. controller.py calls
    update() each poll with the states it just read; the thread only ever reads the
    map, which update() replaces whole, so it never sees a half-rebuilt one."""

    def __init__(
        self,
        *,
        client: sy.Synnax,
        channels: Channels,
        report: Callable[[str, Exception], None] = warn,
    ) -> None:
        self.channels = channels
        self.index = channels.index(INDEX)
        self.states: dict[str, tuple[int, sy.TimeStamp | None]] = {}
        # The runners the writer covers. A frame must carry each of their channels.
        self.known: set[str] = set()
        self.stopped = threading.Event()
        self.lock = threading.Lock()
        self.publisher = Publisher(
            client=client,
            name="runner_uptime",
            channels=[],
            report=report,
            auto_index=True,
        )
        self.thread = threading.Thread(target=self._run, daemon=True)
        self.thread.start()

    def retarget(self, known: set[str]) -> None:
        self.channels.data(index=self.index, channels=uptime_channels(known))
        with self.lock:
            self.known = set(known)
            self.publisher.retarget(list(uptime_channels(known)))

    def update(
        self, states: dict[str, int], *, found: dict[str, dict], now: sy.TimeStamp
    ) -> None:
        """Restarts an online runner's count when its state changes."""
        last = self.states
        self.states = {
            r: (s, last[r][1] if r in last and last[r][0] == s else now)
            if s >= IDLE
            else (s, self.launched(r, found=found))
            for r, s in states.items()
        }

    def launched(self, runner: str, *, found: dict[str, dict]) -> sy.TimeStamp | None:
        iid = instance_for(runner, found=found)
        if iid is None or not found[iid]["running"]:
            return None
        return found[iid]["launched"]

    def frame(self, now: sy.TimeStamp) -> dict[str, int]:
        """Seconds in its current state for every runner the writer covers. A runner
        new since the last update() reads 0 until the next one."""
        return {
            channel_name("state_uptime", runner=r): (
                0
                if (entered := self.states.get(r, (0, None))[1]) is None
                else int(now.span(entered).seconds)
            )
            for r in self.known
        }

    def _run(self) -> None:
        while not self.stopped.wait(TICK.seconds):
            states = self.states
            if not states:
                continue
            with self.lock:
                self.publisher.write(self.frame(sy.TimeStamp.now()))
        self.publisher.close()

    def stop(self) -> None:
        self.stopped.set()
        self.thread.join(timeout=STOP_TIMEOUT.seconds)
