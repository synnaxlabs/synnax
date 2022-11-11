from enum import Enum

from freighter import EOF, ExceptionPayload, Payload, Stream, StreamClient

from synnax.channel.registry import ChannelRegistry
from synnax.telem import TimeRange, TimeSpan, TimeStamp

from .encoder import NumpyEncoderDecoder
from .payload import SegmentPayload
from .sugared import NumpySegment

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
    ack: bool
    command: _Command | None = None
    error: ExceptionPayload | None = None
    segments: list[SegmentPayload] | None = None


class CoreIterator:
    """Used to iterate over a databases telemetry in time-order. It should not be
    instantiated directly, and should instead be instantiated using the segment Client.

    Using an iterator is ideal when querying/processing large ranges of data, but is
    relatively complex and difficult to use. If you're looking to retrieve telemetry
    between two timestamps, see the segment Client read method instead.
    """

    _ENDPOINT = "/segment/iterate"

    client: StreamClient
    stream: Stream[_Request, _Response]
    values: list[SegmentPayload]
    aggregate: bool

    def __init__(self, client: StreamClient, aggregate: bool = False) -> None:
        self.client = client
        self.aggregate = aggregate

    def open(self, keys: list[str], tr: TimeRange):
        """Opens the iterator, configuring it to iterate over the telemetry in the
        channels with the given keys within the provided time range.

        :param keys: The keys of the channels to iterate over.
        :param tr: The time range to iterate over.
        """
        self.stream = self.client.stream(self._ENDPOINT, _Request, _Response)
        self._exec(command=_Command.OPEN, range=tr, keys=keys)
        self.values = []

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
        exc = self.stream.close_send()
        if exc is not None:
            raise exc
        pld, exc = self.stream.receive()
        if not isinstance(exc, EOF):
            raise exc

    def _exec(self, **kwargs) -> bool:
        exc = self.stream.send(_Request(**kwargs))
        if exc is not None:
            raise exc
        if not self.aggregate:
            self.values = []
        while True:
            r, exc = self.stream.receive()
            if exc is not None:
                raise exc
            if r.variant == _ResponseVariant.ACK:
                return r.ack
            self.values += r.segments


class NumpyIterator(CoreIterator):
    """Used to iterate over a databases telemetry in time-order. It should not be
    instantiated directly, and should instead be instantiated using the segment Client.

    Using an iterator is ideal when querying/processing large ranges of data, but is
    relatively complex and difficult to use. If you're looking to retrieve telemetry
    between two timestamps, see the segment Client read method instead.
    """

    decoder: NumpyEncoderDecoder
    channels: ChannelRegistry

    def __init__(
        self,
        transport: StreamClient,
        channels: ChannelRegistry,
        aggregate: bool = False,
    ):
        super().__init__(transport, aggregate)
        self.decoder = NumpyEncoderDecoder()
        self.channels = channels

    def open(self, keys: list[str], tr: TimeRange) -> None:
        super().open(keys, tr)

    @property
    def value(self) -> dict[str, NumpySegment]:
        """
        :returns: The current iterator value as a dictionary whose keys are channels
        and values are segments containing telemetry at the current iterator position.
        """
        decoded = []
        self.values.sort(key=lambda v: v.start)
        res = dict()
        for i, seg in enumerate(self.values):
            decoded.append(self.decoder.decode(self.channels.get(seg.channel_key), seg))
        for i, dec in enumerate(decoded):
            if dec.channel.key in res:
                res[dec.channel.key].extend(dec)
            else:
                res[dec.channel.key] = dec
        return res
