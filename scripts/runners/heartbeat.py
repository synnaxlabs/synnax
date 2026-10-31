#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Liveness for a long-running script, on a stored boolean channel."""

import atexit
import threading
from collections.abc import Callable

import synnax as sy

from common import HEARTBEAT_INTERVAL, STOP_TIMEOUT, Channels
from publisher import Publisher


class Heartbeat:
    """Writes value() (true by default) to the boolean channel every HEARTBEAT_INTERVAL
    from a daemon thread, and false on clean exit, so a lasting true means the
    process died. beat() writes at once for a state change; stop() also runs atexit
    and is safe to call twice. Stored on its own <name>_time index, so the last
    value survives a restart."""

    def __init__(
        self,
        *,
        client: sy.Synnax,
        channels: Channels,
        name: str,
        value: Callable[[], bool] = lambda: True,
        index: str | None = None,
    ) -> None:
        self.name = name
        self.value = value
        self.stopped = threading.Event()
        self.wake = threading.Event()
        channels.data(
            index=channels.index(index or f"{name}_time"),
            channels={name: sy.DataType.BOOLEAN},
        )
        self.publisher = Publisher(
            client=client, name=name, channels=[name], auto_index=True
        )
        atexit.register(self.stop)
        self.thread = threading.Thread(target=self._run, daemon=True)
        self.thread.start()

    def beat(self) -> None:
        self.wake.set()

    def _run(self) -> None:
        # A false first, so a chart steps up from the last run instead of sloping.
        self.publisher.write({self.name: False})
        while True:
            done = self.stopped.is_set()
            self.publisher.write({self.name: False if done else self.value()})
            if done:
                break
            self.wake.wait(HEARTBEAT_INTERVAL.seconds)
            self.wake.clear()
        self.publisher.close()

    def stop(self) -> None:
        if self.stopped.is_set():
            return
        self.stopped.set()
        self.wake.set()
        self.thread.join(timeout=STOP_TIMEOUT.seconds)
