#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

import pytest

import synnax as sy
from framework.writer import Writer


class _Stream:
    def __init__(self) -> None:
        self.written: list[dict[str, int | float | str]] = []
        self.closed = False
        self.close_failure: Exception | None = None

    def write(self, data: dict[str, int | float | str]) -> None:
        self.written.append(data)

    def close(self) -> None:
        if self.close_failure is not None:
            raise self.close_failure
        self.closed = True


class _Client(sy.Synnax):
    """A client whose first open_writer calls fail, as many as open_failures says."""

    def __init__(self, open_failures: int = 0) -> None:
        self.open_failures = open_failures
        self.streams: list[_Stream] = []

    def open_writer(self, *_: Any, **__: Any) -> Any:
        if self.open_failures > 0:
            self.open_failures -= 1
            raise ConnectionError("Core unreachable")
        stream = _Stream()
        self.streams.append(stream)
        return stream


def _writer(client: _Client) -> Writer:
    return Writer(client)


class TestWriter:
    """Writes values through one persistent stream."""

    def test_should_open_a_writer_again_after_a_failed_open(self) -> None:
        client = _Client(open_failures=1)
        writer = _writer(client)
        with pytest.raises(ConnectionError):
            writer.write("a", 1)
        writer.write("a", 2)
        assert client.streams[-1].written == [{"a": 2}]

    def test_should_open_a_writer_again_after_a_failed_reopen(self) -> None:
        client = _Client()
        writer = _writer(client)
        writer.write("a", 1)
        client.open_failures = 1
        with pytest.raises(ConnectionError):
            writer.write("b", 2)
        writer.write("b", 3)
        assert len(client.streams) == 2
        assert client.streams[-1].written == [{"b": 3}]

    def test_should_open_a_writer_again_after_a_failed_close(self) -> None:
        client = _Client()
        writer = _writer(client)
        writer.write("a", 1)

        client.streams[0].close_failure = ConnectionError("stream dropped")
        with pytest.raises(ConnectionError):
            writer.write("b", 2)
        writer.write("b", 3)
        assert client.streams[-1].written == [{"b": 3}]
