#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import queue
import threading

import synnax as sy


class Stream:
    """A stream that returns pushed frames and raises pushed errors, and raises
    close_error on close when set."""

    def __init__(self) -> None:
        self.closed = False
        self.close_error: Exception | None = None
        self._items: queue.Queue[sy.Frame | Exception] = queue.Queue()

    def push(self, item: sy.Frame | Exception) -> None:
        self._items.put(item)

    def read(self, *, timeout: sy.TimeSpan) -> sy.Frame | None:
        try:
            item = self._items.get(timeout=timeout.seconds)
        except queue.Empty:
            return None
        if isinstance(item, Exception):
            raise item
        return item

    def close(self) -> None:
        self.closed = True
        if self.close_error is not None:
            raise self.close_error


class Channels:
    def retrieve(self, channel: list[str]) -> list[sy.Channel]:
        return [
            sy.Channel(name=name, data_type=sy.DataType.FLOAT64, virtual=True)
            for name in channel
        ]


class Client:
    """A client whose every channel exists and whose streams are stream, or raise
    open_error when set."""

    def __init__(self) -> None:
        self.stream = Stream()
        self.open_error: Exception | None = None
        self.opened_by: threading.Thread | None = None
        self._channels = Channels()

    @property
    def channels(self) -> Channels:
        return self._channels

    def open_streamer(self, channels: list[str]) -> Stream:
        if self.open_error is not None:
            raise self.open_error
        self.opened_by = threading.current_thread()
        return self.stream
