#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A streamer that owns its own lifetime, reopening after a failure."""

from collections.abc import Callable

import synnax as sy

from common import close_quietly, warn


class Subscription:
    """Owns the streamer for a set of channels, opening it on first use and reopening
    it after a failure. A failed read reports, waits out the timeout so a broken
    streamer cannot spin, and returns None."""

    def __init__(
        self,
        *,
        client: sy.Synnax,
        name: str,
        channels: list[str],
        report: Callable[[str, Exception], None] = warn,
    ) -> None:
        self.client = client
        self.name = name
        self.channels = channels
        self.report = report
        self.streamer: sy.Streamer | None = None

    def retarget(self, channels: list[str]) -> None:
        """Moves to a different set of channels, reopening on the next read."""
        if channels != self.channels:
            self.channels = channels
            self.close()

    def read(self, *, timeout: sy.TimeSpan) -> sy.Frame | None:
        try:
            if self.streamer is None:
                self.streamer = self.client.open_streamer(self.channels)
            return self.streamer.read(timeout=timeout)
        except Exception as e:
            self.report(self.name, e)
            self.close()
            sy.sleep(timeout)
            return None

    def close(self) -> None:
        close_quietly(self.streamer)
        self.streamer = None
