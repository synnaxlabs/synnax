#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A writer that owns its own lifetime, so callers never hold a closed one."""

from collections.abc import Callable

import synnax as sy

from common import PUBLISH_AUTHORITY, close_quietly, warn

# Callers stamp their samples just before the writer opens, so it starts far enough
# back that those samples still land strictly after it.
OPEN_LEEWAY = 100 * sy.TimeSpan.MILLISECOND
# A change carries the previous value this far ahead of the new one, so plots step.
ANCHOR = sy.TimeSpan.MILLISECOND


def on_change[V](
    index: str, *, last: dict[str, V], current: dict[str, V], now: sy.TimeStamp
) -> dict[str, object] | None:
    """The frame for a change, None without one. The first is one sample per channel,
    later ones [previous, current] at [now - ANCHOR, now]."""
    if current == last:
        return None
    if not last:
        return {index: now, **current}
    return {
        index: [now - ANCHOR, now],
        **{c: [last.get(c, v), v] for c, v in current.items()},
    }


class Publisher:
    """Owns the writer for a set of channels, opening it on first use and reopening it
    on the next write after a failure, so callers never hold a closed one. A failed
    write reports and returns False rather than raising."""

    def __init__(
        self,
        *,
        client: sy.Synnax,
        name: str,
        channels: list[str],
        authority: sy.Authority = PUBLISH_AUTHORITY,
        report: Callable[[str, Exception], None] = warn,
        auto_index: bool = False,
    ) -> None:
        self.client = client
        self.name = name
        self.channels = channels
        self.authority = authority
        self.report = report
        # With auto_index the Core stamps the index at write time, so the caller
        # writes values only and cannot land a sample before the writer's start.
        self.auto_index = auto_index
        self.writer: sy.Writer | None = None

    def retarget(self, channels: list[str]) -> None:
        """Moves to a different set of channels, reopening on the next write."""
        if channels != self.channels:
            self.channels = channels
            self.close()

    def open(self) -> bool:
        """Opens ahead of the first write, to hold control while idle."""
        return self.write(None)

    def write(
        self, frame: dict | None, *, authority: sy.Authority | None = None
    ) -> bool:
        """Writes frame, raising authority for the write itself when one is given so
        control is held no longer than the write takes. None opens without writing."""
        if not self.channels:
            return False
        try:
            if self.writer is None:
                self.writer = self.client.open_writer(
                    sy.TimeStamp.now() - OPEN_LEEWAY,
                    self.channels,
                    authorities=self.authority,
                    name=self.name,
                    auto_index=self.auto_index,
                )
            if frame is None:
                return True
            if authority is not None:
                self.writer.set_authority(authority)
            # CrudeFrame's annotation omits name-keyed dicts, which write accepts.
            self.writer.write(frame)  # pyright: ignore[reportArgumentType]
            if authority is not None:
                self.writer.set_authority(self.authority)
            return True
        except Exception as e:
            self.report(self.name, e)
            self.close()
            return False

    def close(self) -> None:
        close_quietly(self.writer)
        self.writer = None
