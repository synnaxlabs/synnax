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
from collections.abc import Iterator
from typing import cast

import pytest

import synnax as sy
from framework.streamer import Streamer

WAIT = 5.0


class FakeStream:
    def __init__(self, frames: "queue.Queue[sy.Frame]") -> None:
        self._frames = frames

    def read(self, timeout: sy.CrudeTimeSpan) -> sy.Frame | None:
        try:
            return self._frames.get(timeout=sy.TimeSpan.from_seconds(timeout).seconds)
        except queue.Empty:
            return None

    def close(self) -> None:
        pass


class FakeChannels:
    def retrieve(self, channels: str | list[str]) -> str | list[str]:
        return channels


class FakeClient:
    """Stands in for the Synnax client: channels always exist and the stream
    delivers only the frames a test puts on it."""

    def __init__(self) -> None:
        self.channels = FakeChannels()
        self.frames: queue.Queue[sy.Frame] = queue.Queue()
        self.opened = threading.Event()

    def open_streamer(self, channels: list[str]) -> FakeStream:
        self.opened.set()
        return FakeStream(self.frames)


@pytest.fixture
def client() -> FakeClient:
    return FakeClient()


@pytest.fixture
def streamer(client: FakeClient) -> Iterator[Streamer]:
    streamer = Streamer(
        client=cast(sy.Synnax, client),
        read_timeout=0.01,
        log=lambda _: None,
        set_failed=lambda: None,
    )
    streamer.subscribe("a", timeout=WAIT)
    streamer.start()
    assert client.opened.wait(WAIT)
    yield streamer
    streamer.stop()
    streamer.join(WAIT)


class TestStreamer:
    def test_unseen_channel_has_no_value(self, streamer: Streamer) -> None:
        """Should read None and not satisfy a wait for 0 before any sample."""
        assert streamer.read("a") is None
        assert not streamer.wait_for("a", lambda v: v == 0, 0.1)

    def test_wait_for_sample(self, streamer: Streamer, client: FakeClient) -> None:
        """Should satisfy a wait once a matching sample arrives."""
        client.frames.put(sy.Frame({"a": sy.Series([0], data_type=sy.DataType.INT64)}))
        assert streamer.wait_for("a", lambda v: v == 0, WAIT)
        assert streamer.read("a") == 0
