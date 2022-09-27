from freighter import (
    EOF,
    ExceptionPayload,
    Payload,
    Stream,
    StreamClient,
    decode_exception,
)
from numpy import ndarray

from synnax.channel.registry import ChannelRegistry
from synnax.exceptions import UnexpectedError, ValidationError, Field, GeneralError
from synnax.telem import Size, TimeStamp, UnparsedTimeStamp

from .encoder import NumpyEncoderDecoder
from .payload import SegmentPayload
from .splitter import Splitter
from .sugared import NumpySegment, SugaredBinarySegment
from .validate import ContiguityValidator, ScalarTypeValidator, Validator


class _Request(Payload):
    open_keys: list[str]
    segments: list[SegmentPayload]


class _Response(Payload):
    ack: bool
    error: ExceptionPayload


class CoreWriter:
    """Used to write telemetry to a set of channels in time-order. It should not be
    instantiated directly, and should instead be created using the segment client.

    Using a writer is ideal when writing large volumes of data (such as recording telemetry
    from a sensor), but it is relatively complex and challenging to use. If you're looking
    to write a contiguous block of telemetry, see the segment Client write method instead.
    """

    _ENDPOINT = "/segment/write"
    client: StreamClient
    stream: Stream[_Request, _Response] | None
    keys: list[str]

    def __init__(self, client: StreamClient) -> None:
        self.client = client

    def open(self, keys: list[str]):
        """Opens the writer, acquiring an exclusive lock on the given
        channels for the duration of the writer's lifetime. open must be called before
        any other writer methods.

        :param keys: A list of keys representing the channels the writer will write to.
        """
        self.keys = keys
        self.stream = self.client.stream(self._ENDPOINT, _Request, _Response)
        self.stream.send(_Request(open_keys=keys, segments=[]))
        res, exc = self.stream.receive()
        if exc is not None:
            raise exc
        assert res is not None
        if not res.ack:
            raise UnexpectedError(
                "Writer failed to positively acknowledge open request. This is a bug"
                + "please report it."
            )

    def write(self, segments: list[SegmentPayload]) -> bool:
        """Validates and writes the given segments to the database. The provided segments
        must:

            1. Be in time order (on a per-channel basis).
            2. Have channel keys in the set of keys provided to open.
            3. Have non-zero length data with the correct data type for the given channel.

        :param segments: A list of segments to write to the database.
        :returns: False if the writer has accumulated an error. In this case,
        the caller should stop executing requests and close the writer.
        """
        self._assert_open()
        if self.stream.received():
            return False

        self._check_keys(segments)
        err = self.stream.send(_Request(open_keys=[], segments=segments))
        if err is not None:
            raise err
        return True

    def close(self):
        """Closes the writer, raising any accumulated error encountered during operation.
        A writer MUST be closed after use, and this method should probably be placed in
        a 'finally' block. If the writer is not closed, the database will not release
        the exclusive lock on the channels, preventing any other callers from writing to
        them. It also might leak resources and threads.
        """
        self._assert_open()
        self.stream.close_send()
        res, err = self.stream.receive()
        if err is None:
            err = decode_exception(res.error)
        if not isinstance(err, EOF):
            raise err

    def _check_keys(self, segments: list[SegmentPayload]):
        for segment in segments:
            if segment.channel_key not in self.keys:
                raise ValidationError(
                    Field(
                        "key",
                        f"key {segment.key} is not in the list of keys for this writer.",
                    )
                )

    def _assert_open(self):
        if self.stream is None:
            raise GeneralError(
                "Writer is not open. Please open before calling write() or close()."
            )


class NumpyWriter:
    """Used to write telemetry to a set of channels in time-order. It should not be
    instantiated directly, and should instead be created using the segment client.

    Using a writer is ideal when writing large volumes of data (such as recording telemetry
    from a sensor), but it is relatively complex and challenging to use. If you're looking
    to write a contiguous block of telemetry, see the segment Client write method instead.
    """

    core: CoreWriter
    validators: list[Validator]
    encoder: NumpyEncoderDecoder
    splitter: Splitter
    channels: ChannelRegistry

    def __init__(
        self,
        core: CoreWriter,
        channels: ChannelRegistry,
    ) -> None:
        self.core = core
        self.validators = [
            ScalarTypeValidator(),
            ContiguityValidator(dict(), allow_no_high_water_mark=True),
        ]
        self.encoder = NumpyEncoderDecoder()
        self.splitter = Splitter(threshold=Size(4e6))
        self.channels = channels

    def open(self, keys: list[str]):
        """Opens the writer, acquiring an exclusive lock on the given
        channels for the duration of the writer's lifetime. open must be called before
        any other writer methods.
        """
        self.core.open(keys)

    def write(self, to: str, start: UnparsedTimeStamp, data: ndarray) -> bool:
        """Writes the given telemetry to the database.

        :param to: The key of the channel to write to. This key must be present
        in the list of keys the writer was opened with.
        :param start: The start timestamp of the first sample in data. This must be
        equal to the end of the previous segment written to the channel (unless
        this is the first segment written to the channel).
        :param data: The telemetry to write. This must be a 1D numpy array with
        the same data type as the channel.
        :returns: False if the writer has accumulated an error. In this case,
        the caller should stop executing requests and close the writer.
        """
        ch = self.channels.get(to)
        seg = NumpySegment(ch, TimeStamp(start), data)
        for val in self.validators:
            val.validate(seg)
        encoded = SugaredBinarySegment.sugar(ch, self.encoder.encode(seg))
        split = self.splitter.split(encoded)
        return self.core.write([seg.payload() for seg in split])

    def close(self):
        """Closes the writer, raising any accumulated error encountered during operation.
        A writer MUST be closed after use, and this method should probably be placed in
        a 'finally' block. If the writer is not closed, the database will not release
        the exclusive lock on the channels, preventing any other callers from writing to
        them. It also might leak resources and threads.
        """
        self.core.close()
