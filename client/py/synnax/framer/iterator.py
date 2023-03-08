#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from enum import Enum

from freighter import EOF, ExceptionPayload, Payload, Stream, StreamClient

from synnax.exceptions import UnexpectedError
from synnax.telem import TimeRange, TimeSpan, TimeStamp
from synnax.framer.payload import BinaryFrame, NumpyFrame
from synnax.channel.retrieve import ChannelRetriever

AUTO_SPAN = TimeSpan(-1)


class _Command(int, Enum):
    OPEN = 0
    NEXT = 1
    PREV = 2
    SEEK_FIRST = 3
    SEEK_LAST = 4
    SEEK_LE = 5
    SEEK_GE = 6
    VALID = 7
    ERROR = 8


class _ResponseVariant(int, Enum):
    NONE = 0
    ACK = 1
    DATA = 2


class _Request(Payload):
    command: _Command
    span: TimeSpan | None = None
    range: TimeRange | None = None
    stamp: TimeStamp | None = None
    keys: list[str] | None = None


class _Response(Payload):
    variant: _ResponseVariant
    command: _Command
    ack: bool
    error: ExceptionPayload
    frame: BinaryFrame


class CoreIterator:
    """Used to iterate over a databases telemetry in time-order. It should not be
    instantiated directly, and should instead be instantiated using the segment Client.

    Using an iterator is ideal when querying/processing large ranges of data, but is
    relatively complex and difficult to use. If you're looking to retrieve telemetry
    between two timestamps, see the segment Client read method instead.
    """

    _ENDPOINT = "/frame/iterate"

    aggregate: bool
    open: bool
    tr: TimeRange
    keys: list[str]
    _client: StreamClient
    _stream: Stream[_Request, _Response]
    _value: BinaryFrame

    def __init__(
        self,
        client: StreamClient,
        tr: TimeRange,
        keys: list[str],
        aggregate: bool = False,
    ) -> None:
        self._client = client
        self.aggregate = aggregate
        self.keys = keys
        self.tr = tr
        self._open()

    def _open(self):
        """Opens the iterator, configuring it to iterate over the telemetry in the
        channels with the given keys within the provided time range.

        :param keys: The keys of the channels to iterate over.
        :param tr: The time range to iterate over.
        """
        self._stream = self._client.stream(self._ENDPOINT, _Request, _Response)
        self._exec(command=_Command.OPEN, range=self.tr, keys=self.keys)
        self._value = BinaryFrame()

    def next(self, span: TimeSpan) -> bool:
        """Reads the next time span of telemetry for each channel in the iterator.

        :param span: The span of time to read. A negative span is equivalent to calling
        prev with the absolute value of the span. If AUTO_SPAN is provided, the iterator
        will automatically determine the span of time to read. This is useful for iterating
        quickly over an entire time range.

        :returns: False if a segment satisfying the request can't be found for a
        particular channel or the iterator has accumulated an error.
        """
        return self._exec(command=_Command.NEXT, span=span)

    def prev(self, span: TimeSpan) -> bool:
        """Reads the previous time span of telemetry for each channel in the iterator.

        :param span: The span of time to read. A negative span is equivalent to calling
        next with the absolute value of the span. If AUTO_SPAN is provided, the iterator
        will automatically determine the span of time to read. This is useful for iterating
        quickly over an entire time range.

        :returns: False if a segment satisfying the request can't be found for a particular
        channel or the iterator has accumulated an error.
        """
        return self._exec(command=_Command.NEXT, span=span)

    def seek_first(self) -> bool:
        """Seeks the iterator to the first segment in the time range, but does not read
        it. Also invalidates the iterator. The iterator will not be considered valid
        until a call to first, last, next, prev, prev_span, next_span, or next_range.

        :returns: False if the iterator is not pointing to a valid segment for a particular
        channel or has accumulated an error.
        """
        return self._exec(command=_Command.SEEK_FIRST)

    def seek_last(self) -> bool:
        """Seeks the iterator to the last segment in the time range, but does not read it.
        Also invalidates the iterator. The iterator will not be considered valid
        until a call to first, last, next, prev, prev_span, next_span, or next_range.

        :returns: False if the iterator is not pointing to a valid segment for a particular
        channel or has accumulated an error.
        """
        return self._exec(command=_Command.SEEK_LAST)

    def seek_lt(self, stamp: TimeStamp) -> bool:
        """Seeks the iterator to the first segment whose start is less than or equal to
        the provided timestamp. Also invalidates the iterator. The iterator will not be
        considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.

        :returns: False if the iterator is not pointing to a valid segment for a particular
        channel or has accumulated an error.
        """
        return self._exec(command=_Command.SEEK_LE, stamp=stamp)

    def seek_ge(self, stamp: TimeStamp) -> bool:
        """Seeks the iterator to the first segment whose start is greater than or equal to
        the provided timestamp. Also invalidates the iterator. The iterator will not be
        considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.

        :returns: False if the iterator is not pointing to a valid segment for a particular
        channel or has accumulated an error.
        """
        return self._exec(command=_Command.SEEK_GE, stamp=stamp)

    def valid(self) -> bool:
        """Returns true if the iterator value contains a valid segment, and False otherwise.
        valid most commonly returns false when the iterator is exhausted or has accumulated
        an error.
        """
        return self._exec(command=_Command.VALID)

    def close(self):
        """Close closes the iterator. An iterator MUST be closed after use, and this method
        should probably be placed in a 'finally' block. If the iterator is not closed, it make
        leak resources and threads.
        """
        exc = self._stream.close_send()
        if exc is not None:
            raise exc
        _, exc = self._stream.receive()
        if exc is None:
            raise UnexpectedError(
                """Unexpected missing close acknowledgement from server.
                Please report this issue to the Synnax team."""
            )
        elif not isinstance(exc, EOF):
            raise exc

    def __iter__(self):
        if not self.seek_first():
            raise StopIteration
        return self

    def __next__(self):
        if not self.next(AUTO_SPAN):
            raise StopIteration
        return self.value

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.close()

    @property
    def value(self) -> BinaryFrame:
        return self._value.compact()

    def _exec(self, **kwargs) -> bool:
        exc = self._stream.send(_Request(**kwargs))
        if exc is not None:
            raise exc
        if not self.aggregate:
            self._value = BinaryFrame()
        while True:
            r, exc = self._stream.receive()
            if exc is not None:
                raise exc
            assert r is not None
            if r.variant == _ResponseVariant.ACK:
                return r.ack
            self._value.append_frame(r.frame)


class NumpyIterator(CoreIterator):
    """Used to iterate over a databases telemetry in time-order. It should not be
    instantiated directly, and should instead be instantiated using the segment Client.

    Using an iterator is ideal when querying/processing large ranges of data, but is
    relatively complex and difficult to use. If you're looking to retrieve telemetry
    between two timestamps, see the segment Client read method instead.
    """

    _channels: ChannelRetriever

    def __init__(
        self,
        transport: StreamClient,
        channels: ChannelRetriever,
        tr: TimeRange,
        *keys_or_names: str | list[str],
        aggregate: bool = False,
    ):
        self._channels = channels
        channels = self._channels.retrieve(*keys_or_names, node_id=None)
        super().__init__(transport,  tr, [ch.key for ch in channels], aggregate)

    @property
    def value(self) -> NumpyFrame:
        """
        :returns: The current iterator value as a dictionary whose keys are channels
        and values are segments containing telemetry at the current iterator position.
        """
        v = super().value
        v.keys = self._value_keys(v.keys)
        return NumpyFrame.from_binary(v)

    def _value_keys(self, keys: list[str]) -> list[str]:
        # We can safely ignore the none case here because we've already
        # checked that all channels can be retrieved.
        channels = self._channels.retrieve(keys)
        return [ch.name if ch.key != key else key for key, ch in zip(keys, channels)]
