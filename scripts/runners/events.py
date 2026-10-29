#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The <runner>_events channels: what each runner's current job did, one line per job
or step transition."""

from collections.abc import Callable

import synnax as sy

from common import Channels, channel_name, warn
from publisher import Publisher

SOURCE = "events"
# Bounds what a long Core outage holds in memory.
MAX_PENDING = 1000


class Events:
    """Writes only for the runners that have new lines, stamped at write time on each
    runner's own <runner>_events_time index. The Core rejects a timestamp at or before
    the previous one on an index, or before the writer start, and the writer then
    fails asynchronously, so timestamps are clamped here."""

    def __init__(
        self,
        *,
        client: sy.Synnax,
        channels: Channels,
        runners: set[str],
        report: Callable[[str, Exception], None] = warn,
    ) -> None:
        self.channels = channels
        self.start = sy.TimeStamp.now()
        self.last: dict[str, sy.TimeStamp] = {}
        self.pending: dict[str, list[str]] = {}
        self.publisher = Publisher(
            client=client, name=f"runner_{SOURCE}", channels=[], report=report
        )
        self.retarget(runners)

    def retarget(self, runners: set[str]) -> None:
        """Creates the channels for the current runner set and moves the writer onto
        them."""
        names = []
        for r in sorted(runners):
            index_name = channel_name(f"{SOURCE}_time", runner=r)
            index = self.channels.index(index_name)
            name = channel_name(SOURCE, runner=r)
            self.channels.data(index=index, channels={name: sy.DataType.STRING})
            names += [index_name, name]
        self.pending = {r: t for r, t in self.pending.items() if r in runners}
        self.publisher.retarget(names)

    def write(self, lines: dict[str, list[str]]) -> None:
        """Writes lines after any that failed to write before. Lines past
        MAX_PENDING per runner drop, oldest first."""
        for runner, texts in lines.items():
            pending = self.pending.setdefault(runner, []) + texts
            self.pending[runner] = pending[-MAX_PENDING:]
        now = sy.TimeStamp.now()
        frame: dict[str, list] = {}
        last: dict[str, sy.TimeStamp] = {}
        for runner, texts in self.pending.items():
            if not texts:
                continue
            stamp = self.last.get(runner, self.start)
            stamps = []
            for _ in texts:
                stamp = max(now, stamp + 1)
                stamps.append(stamp)
            last[runner] = stamp
            frame[channel_name(SOURCE, runner=runner)] = texts
            frame[channel_name(f"{SOURCE}_time", runner=runner)] = stamps
        if frame and self.publisher.write(frame):
            self.last.update(last)
            self.pending.clear()

    def close(self) -> None:
        self.publisher.close()
