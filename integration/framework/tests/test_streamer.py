#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from framework.streamer import Streamer
from framework.tests.fake import Client

CHANNEL = "value"
WAIT = 5.0


@pytest.fixture
def client() -> Client:
    return Client()


@pytest.fixture
def streamer(client: Client) -> Streamer:
    streamer = Streamer(client=client, read_timeout=0.01, log=lambda _: None)
    streamer.subscribe(CHANNEL)
    return streamer


def stop(streamer: Streamer) -> None:
    streamer.stop()
    streamer.join(WAIT)


class TestStreamer:
    def test_wait_for_value(self, client: Client, streamer: Streamer) -> None:
        """Should return True once the value satisfies the predicate."""
        streamer.start()
        client.stream.push(sy.Frame({CHANNEL: sy.Series([1.0, 2.0])}))
        assert streamer.wait_for(CHANNEL, lambda v: v == 2.0, WAIT)
        assert streamer.read(CHANNEL) == 2.0
        stop(streamer)
        assert client.stream.closed

    def test_wait_for_raises_read_error(
        self, client: Client, streamer: Streamer
    ) -> None:
        """Should raise the error that stopped the stream from a waiting call."""
        streamer.start()
        client.stream.push(ValueError("decode failed"))
        with pytest.raises(RuntimeError, match="decode failed") as info:
            streamer.wait_for(CHANNEL, lambda v: v == 1.0, WAIT)
        assert isinstance(info.value.__cause__, ValueError)
        stop(streamer)
        assert client.stream.closed
        with pytest.raises(RuntimeError, match="decode failed"):
            streamer.read(CHANNEL)
        with pytest.raises(RuntimeError, match="decode failed"):
            streamer.check()

    def test_raises_open_error(self, client: Client, streamer: Streamer) -> None:
        """Should raise an error from opening the stream."""
        client.open_error = ConnectionError("refused")
        streamer.start()
        with pytest.raises(RuntimeError, match="refused"):
            streamer.wait_for(CHANNEL, lambda v: v == 1.0, WAIT)

    def test_raises_close_error(self, client: Client, streamer: Streamer) -> None:
        """Should raise an error from closing the stream after a stop."""
        client.stream.close_error = ConnectionError("reset")
        streamer.start()
        stop(streamer)
        with pytest.raises(RuntimeError, match="reset"):
            streamer.check()

    def test_stop_without_error(self, client: Client, streamer: Streamer) -> None:
        """Should not report an error after a clean stop."""
        streamer.start()
        stop(streamer)
        streamer.check()
