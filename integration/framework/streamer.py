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
from typing import Literal, overload

import synnax as sy
from x import is_websocket_error, suppress_websocket_errors


class Streamer:
    """Background streamer that reads subscribed channels into a shared frame."""

    WEBSOCKET_RETRY_DELAY: sy.CrudeTimeSpan = 500 * sy.TimeSpan.MILLISECOND

    def __init__(
        self,
        client: sy.Synnax,
        read_timeout: sy.CrudeTimeSpan,
        log: Callable[[str], None],
        set_failed: Callable[[], None],
    ) -> None:
        self._client = client
        self._read_timeout = read_timeout
        self._log = log
        self._set_failed = set_failed
        self._channels: set[str] = set()
        self._frame: dict[str, int | float | str] = {}
        self._thread: threading.Thread = threading.Thread()
        self._should_stop = False

    @property
    def read_timeout(self) -> sy.CrudeTimeSpan:
        return self._read_timeout

    @read_timeout.setter
    def read_timeout(self, value: sy.CrudeTimeSpan) -> None:
        self._read_timeout = value

    def subscribe(
        self,
        channels: str | list[str],
        timeout: sy.CrudeTimeSpan = 10 * sy.TimeSpan.SECOND,
    ) -> None:
        timeout_span = sy.TimeSpan.from_seconds(timeout)
        self._log(f"Subscribing to channels: {channels} ({timeout_span} timeout)")

        loop = sy.Loop(200 * sy.TimeSpan.MILLISECOND)
        time_start = sy.TimeStamp.now()
        found = False

        while loop.wait():
            elapsed = sy.TimeStamp.now() - time_start
            if elapsed > timeout_span:
                break
            try:
                existing = self._client.channels.retrieve(channels)
                if isinstance(channels, str):
                    found = existing is not None
                else:
                    found = isinstance(existing, list) and len(existing) == len(
                        channels
                    )
                if found:
                    break
            except Exception as e:
                self._log(f"Channel retrieval failed: {e}")
                continue

        if not found:
            raise TimeoutError(f"Unable to retrieve channels: {channels}")

        self._log("Channels retrieved")
        if isinstance(channels, str):
            self._channels.add(channels)
        elif isinstance(channels, list):
            self._channels.update(channels)

    @overload
    def read(self, key: str, default: Literal[None] = None) -> int | float | None: ...

    @overload
    def read(self, key: str, default: int | float) -> int | float: ...

    @overload
    def read(self, key: str, default: str) -> str: ...

    def read(
        self, key: str, default: int | float | str | None = None
    ) -> int | float | str | None:
        try:
            return self._frame.get(key, default)
        except Exception:
            return default

    def start(self) -> None:
        self._thread = threading.Thread(target=self._loop, daemon=True)
        self._thread.start()

    def stop(self) -> None:
        self._should_stop = True

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

        streamer = None
        try:
            streamer = self._client.open_streamer(list(self._channels))
            while not self._should_stop:
                try:
                    frame = streamer.read(self._read_timeout)
                    if frame is not None:
                        for key, value in frame.items():
                            self._frame[key] = value[-1]
                except Exception as e:
                    if is_websocket_error(e):
                        sy.sleep(self.WEBSOCKET_RETRY_DELAY)
                    else:
                        self._log(f"Streamer error: {e}")
                        break
        except Exception as e:
            if not is_websocket_error(e):
                import traceback

                self._log(f"Streamer thread error: {e}\n {traceback.format_exc()}")
                self._set_failed()
                raise
        finally:
            if streamer is not None:
                with suppress_websocket_errors():
                    streamer.close()
