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

from common import channel_name, create_channels, create_index, warn
from publisher import Publisher

SOURCE = "events"


class Events:
    """Writes only for the runners that have new lines, stamped at write time on each
    runner's own <runner>_events_time index. The Core rejects a timestamp at or before
    the previous one on an index, or before the writer start, and the writer then
    fails asynchronously, so timestamps are clamped here."""

    def __init__(
        self,
        client: sy.Synnax,
        runners: set[str],
        report: Callable[[str, Exception], None] = warn,
    ) -> None:
        self.client = client
        self.start = sy.TimeStamp.now()
        self.last: dict[str, sy.TimeStamp] = {}
        self.publisher = Publisher(client, f"runner_{SOURCE}", [], report=report)
        self.retarget(runners)

    def retarget(self, runners: set[str]) -> None:
        """Creates the channels for the current runner set and moves the writer onto
        them."""
        names = []
        for r in sorted(runners):
            index_name = channel_name(f"{SOURCE}_time", r)
            index = create_index(self.client, index_name)
            name = channel_name(SOURCE, r)
            create_channels(self.client, index, {name: sy.DataType.STRING})
            names += [index_name, name]
        self.publisher.retarget(names)

    def write(self, lines: dict[str, list[str]]) -> None:
        now = sy.TimeStamp.now()
        frame: dict[str, list] = {}
        for runner, texts in lines.items():
            if not texts:
                continue
            last = self.last.get(runner, self.start)
            stamps = []
            for _ in texts:
                last = max(now, last + 1)
                stamps.append(last)
            self.last[runner] = last
            frame[channel_name(SOURCE, runner)] = texts
            frame[channel_name(f"{SOURCE}_time", runner)] = stamps
        if frame:
            self.publisher.write(frame)

    def close(self) -> None:
        self.publisher.close()
