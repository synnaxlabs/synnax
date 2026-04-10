#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading

import synnax as sy


class Writer:
    """A persistent, thread-safe writer that stays alive across multiple writes."""

    def __init__(self, client: sy.Synnax) -> None:
        self._client = client
        self._writer: sy.Writer | None = None
        self._channels: set[str] = set()
        self._lock = threading.Lock()
        self._commit_ts = sy.TimeStamp.now()

    def write(
        self,
        channel_or_data: str | dict[str, int | float | str | None],
        value: int | float | str | None = None,
    ) -> None:
        if isinstance(channel_or_data, str):
            data = {channel_or_data: value}
        else:
            data = channel_or_data

        with self._lock:
            if not data.keys() <= self._channels:
                if self._writer is not None:
                    self._writer.close()
                self._channels |= data.keys()
                # Use the current commit timestamp as the start for the new
                # writer so we don't overlap with data we just committed.
                self._writer = self._client.open_writer(
                    start=self._commit_ts,
                    channels=list(self._channels),
                    enable_auto_commit=True,
                )
            assert self._writer is not None
            self._writer.write(data)
            self._commit_ts = sy.TimeStamp.now()

    def close(self) -> None:
        with self._lock:
            if self._writer is not None:
                self._writer.close()
                self._writer = None
                self._channels = set()
