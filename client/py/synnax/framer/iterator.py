#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from enum import Enum

from alamos import NOOP, Instrumentation, trace
from freighter import EOF, ExceptionPayload, Payload, Stream, StreamClient

from synnax.channel.payload import ChannelKeys
from synnax.exceptions import UnexpectedError
from synnax.framer.adapter import ReadFrameAdapter
from synnax.framer.frame import Frame, FramePayload
from synnax.telem import TimeRange, TimeSpan, TimeStamp

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
    bounds: TimeRange | None = None
    stamp: TimeStamp | None = None
    keys: ChannelKeys | None = None


class _Response(Payload):
    variant: _ResponseVariant
    command: _Command
    ack: bool
    error: str | None
    frame: FramePayload


class Iterator:
    """Used to iterate over a databases telemetry in time-order. It should not be
    instantiated directly, and should instead be instantiated using the segment Client.

    Using an iterator is ideal when querying/processing large ranges of data, but is
    relatively complex and difficult to use. If you're looking to retrieve telemetry
    between two timestamps, see the segment Client read method instead.
    """

    __ENDPOINT = "/frame/iterate"
    __stream: Stream[_Request, _Response]
    __adapter: ReadFrameAdapter

    open: bool
    tr: TimeRange
    instrumentation: Instrumentation
    value: Frame

    def __init__(
        self,
        tr: TimeRange,
        client: StreamClient,
        adapter: ReadFrameAdapter,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self.tr = tr
        self.instrumentation = instrumentation
        self.__adapter = adapter
        self.__stream = client.stream(self.__ENDPOINT, _Request, _Response)
        self.__open()

    @trace("debug", "open")
    def __open(self):
        """Opens the iterator, configuring it to iterate over the telemetry in the
        channels with the given keys within the provided time range.

        :param keys: The keys of the channels to iterate over.
        :param tr: The time range to iterate over.
        """
        self._exec(command=_Command.OPEN, bounds=self.tr, keys=self.__adapter.keys)
        self.value = Frame()

    @trace("debug")
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

    @trace("debug")
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

    @trace("debug")
    def seek_first(self) -> bool:
        """Seeks the iterator to the first segment in the time range, but does not read
        it. Also invalidates the iterator. The iterator will not be considered valid
        until a call to first, last, next, prev, prev_span, next_span, or next_range.

        :returns: False if the iterator is not pointing to a valid segment for a particular
        channel or has accumulated an error.
        """
        return self._exec(command=_Command.SEEK_FIRST)

    @trace("debug")
    def seek_last(self) -> bool:
        """Seeks the iterator to the last segment in the time range, but does not read it.
        Also invalidates the iterator. The iterator will not be considered valid
        until a call to first, last, next, prev, prev_span, next_span, or next_range.

        :returns: False if the iterator is not pointing to a valid segment for a particular
        channel or has accumulated an error.
        """
        return self._exec(command=_Command.SEEK_LAST)

    @trace("debug")
    def seek_lt(self, stamp: TimeStamp) -> bool:
        """Seeks the iterator to the first segment whose start is less than or equal to
        the provided timestamp. Also invalidates the iterator. The iterator will not be
        considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.

        :returns: False if the iterator is not pointing to a valid segment for a particular
        channel or has accumulated an error.
        """
        return self._exec(command=_Command.SEEK_LE, stamp=stamp)

    @trace("debug")
    def seek_ge(self, stamp: TimeStamp) -> bool:
        """Seeks the iterator to the first segment whose start is greater than or equal to
        the provided timestamp. Also invalidates the iterator. The iterator will not be
        considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.

        :returns: False if the iterator is not pointing to a valid segment for a particular
        channel or has accumulated an error.
        """
        return self._exec(command=_Command.SEEK_GE, stamp=stamp)

    @trace("debug")
    def valid(self) -> bool:
        """Returns true if the iterator value contains a valid segment, and False otherwise.
        valid most commonly returns false when the iterator is exhausted or has accumulated
        an error.
        """
        return self._exec(command=_Command.VALID)

    @trace("debug")
    def close(self):
        """Close closes the iterator. An iterator MUST be closed after use, and this method
        should probably be placed in a 'finally' block. If the iterator is not closed, it make
        leak resources and threads.
        """
        exc = self.__stream.close_send()
        if exc is not None:
            raise exc
        r, exc = self.__stream.receive()
        if exc is None:
            raise UnexpectedError(
                """Unexpected missing close acknowledgement from server.
                Please report this issue to the Synnax team."""
            )
        elif not isinstance(exc, EOF):
            raise exc

    def __iter__(self):
        self.seek_first()
        return self

    def __next__(self):
        if not self.next(AUTO_SPAN):
            raise StopIteration
        return self.value

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.close()

    def _exec(self, **kwargs) -> bool:
        exc = self.__stream.send(_Request(**kwargs))
        if exc is not None:
            raise exc
        self.value = Frame()
        while True:
            r, exc = self.__stream.receive()
            if exc is not None:
                raise exc
            assert r is not None
            if r.variant == _ResponseVariant.ACK:
                return r.ack
            fr = Frame(channels=r.frame.keys, series=r.frame.series)
            self.value.append(self.__adapter.adapt(fr))
