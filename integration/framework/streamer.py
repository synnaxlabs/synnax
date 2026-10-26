#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from collections.abc import Callable
from typing import Literal, Protocol, overload

import numpy as np

import synnax as sy
from framework.websocket import suppress_websocket_errors
from synnax.telem import SampleValue


class Stream(Protocol):
    def read(self, timeout: sy.TimeSpan) -> sy.Frame | None: ...

    def close(self) -> None: ...


class Source(Protocol):
    """The part of a Synnax client that opens streams."""

    def open_streamer(self, channels: list[str]) -> Stream: ...


class Channels(Protocol):
    def retrieve(self, channel: list[str]) -> list[sy.Channel]: ...


class Client(Source, Protocol):
    """The part of a Synnax client that a Streamer uses."""

    @property
    def channels(self) -> Channels: ...


def _scalar(value: SampleValue) -> int | float | str:
    """:raises TypeError: If value is not a number or a string."""
    if isinstance(value, (np.integer, np.bool_)):
        return int(value)
    if isinstance(value, np.floating):
        return float(value)
    if isinstance(value, (int, float, str)):
        return value
    raise TypeError(f"Unsupported telemetry value {value!r}")


class Streamer:
    """Background streamer that reads subscribed channels into a shared frame.

    An error stops the stream. read, wait_for, and check then raise it.
    """

    def __init__(
        self,
        client: Client,
        read_timeout: sy.CrudeTimeSpan,
        log: Callable[[str], None],
    ) -> None:
        self._client = client
        self._read_timeout = sy.TimeSpan.from_seconds(read_timeout)
        self._log = log
        self._error: Exception | None = None
        self._channels: set[str] = set()
        self._frame: dict[str, int | float | str] = {}
        self._cond = threading.Condition()
        self._thread: threading.Thread = threading.Thread()
        self._should_stop = False

    @property
    def read_timeout(self) -> sy.TimeSpan:
        return self._read_timeout

    @read_timeout.setter
    def read_timeout(self, value: sy.CrudeTimeSpan) -> None:
        self._read_timeout = sy.TimeSpan.from_seconds(value)

    def subscribe(
        self,
        channels: str | list[str],
        timeout: sy.CrudeTimeSpan = 10 * sy.TimeSpan.SECOND,
    ) -> None:
        timeout_span = sy.TimeSpan.from_seconds(timeout)
        self._log(f"Subscribing to channels: {channels} ({timeout_span} timeout)")
        names = [channels] if isinstance(channels, str) else channels

        loop = sy.Loop(200 * sy.TimeSpan.MILLISECOND)
        time_start = sy.TimeStamp.now()
        found = False

        while loop.wait():
            elapsed = sy.TimeStamp.now() - time_start
            if elapsed > timeout_span:
                break
            try:
                found = len(self._client.channels.retrieve(names)) == len(names)
                if found:
                    break
            except Exception as e:
                self._log(f"Channel retrieval failed: {e}")
                continue

        if not found:
            raise TimeoutError(f"Unable to retrieve channels: {channels}")

        self._log("Channels retrieved")
        self._channels.update(names)

    @overload
    def read(self, key: str, default: Literal[None] = None) -> int | float | None: ...

    @overload
    def read(self, key: str, default: int | float) -> int | float: ...

    @overload
    def read(self, key: str, default: str) -> str: ...

    def read(
        self, key: str, default: int | float | str | None = None
    ) -> int | float | str | None:
        """:raises RuntimeError: If the stream failed."""
        self.check()
        return self._frame.get(key, default)

    def wait_for(
        self,
        key: str,
        predicate: Callable[[int | float | str], bool],
        timeout: sy.CrudeTimeSpan,
    ) -> bool:
        """Block until key's latest value satisfies predicate, or timeout.

        :raises RuntimeError: If the stream failed.
        """
        deadline = sy.TimeSpan.from_seconds(timeout).seconds
        timer = sy.Timer()
        with self._cond:
            while True:
                self.check()
                if self._should_stop:
                    return False
                value = self._frame.get(key)
                if value is not None and predicate(value):
                    return True
                remaining = deadline - timer.elapsed().seconds
                if remaining <= 0:
                    return False
                self._cond.wait(remaining)

    def check(self) -> None:
        """:raises RuntimeError: If the stream failed. Its cause is the error that
        stopped the stream.
        """
        if self._error is not None:
            raise RuntimeError(f"Streamer failed: {self._error}") from self._error

    def is_subscribed(self, key: str) -> bool:
        return key in self._channels

    def start(self) -> None:
        self._thread = threading.Thread(target=self._loop, daemon=True)
        self._thread.start()

    def stop(self) -> None:
        self._should_stop = True
        with self._cond:
            self._cond.notify_all()

    def join(self, timeout: float = 5) -> None:
        if self._thread.is_alive():
            self._thread.join(timeout=timeout)
            if self._thread.is_alive():
                self._log("Warning: streamer thread did not stop within timeout")

    @property
    def has_channels(self) -> bool:
        return len(self._channels) > 0

    def _loop(self) -> None:
        if not self._channels:
            return

        for ch in self._channels:
            self._frame[ch] = 0

        try:
            self._stream()
        except Exception as e:
            self._log(f"Streamer failed: {e}")
            with self._cond:
                self._error = e
                self._cond.notify_all()

    def _stream(self) -> None:
        streamer = self._client.open_streamer(list(self._channels))
        try:
            while not self._should_stop:
                frame = streamer.read(self._read_timeout)
                if frame is None:
                    continue
                with self._cond:
                    for name in self._channels:
                        series = frame.get(name)
                        if series is not None and len(series) > 0:
                            self._frame[name] = _scalar(series[-1])
                    self._cond.notify_all()
        finally:
            with suppress_websocket_errors():
                streamer.close()
