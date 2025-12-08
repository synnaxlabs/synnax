#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from enum import Enum
from typing import Literal, TypeAlias, overload
from uuid import uuid4

from freighter import (
    EOF,
    ExceptionPayload,
    Payload,
    Stream,
    WebsocketClient,
    decode_exception,
)
from freighter.websocket import Message

from synnax.channel.payload import (
    ChannelKey,
    ChannelKeys,
    ChannelName,
    ChannelNames,
    ChannelPayload,
)
from synnax.framer.adapter import WriteFrameAdapter
from synnax.framer.codec import (
    HIGH_PERF_SPECIAL_CHAR,
    LOW_PERF_SPECIAL_CHAR,
    WSFramerCodec,
)
from synnax.framer.frame import CrudeFrame, FramePayload
from synnax.telem import CrudeSeries, CrudeTimeStamp, TimeSpan, TimeStamp
from synnax.telem.control import Authority, CrudeAuthority, Subject
from synnax.util.normalize import normalize


class WriterCommand(int, Enum):
    OPEN = 0
    WRITE = 1
    COMMIT = 2
    SET_AUTHORITY = 3


class WriterMode(int, Enum):
    PERSIST_STREAM = 1
    PERSIST = 2
    STREAM = 3


CrudeWriterMode: TypeAlias = (
    WriterMode
    | Literal["persist_stream"]
    | Literal["persist"]
    | Literal["stream"]
    | int
)


class WriterConfig(Payload):
    authorities: list[int] = Authority.ABSOLUTE
    control_subject: Subject = Subject(name="", key=str(uuid4()))
    start: TimeStamp | None = None
    keys: ChannelKeys
    mode: WriterMode = WriterMode.PERSIST_STREAM
    err_on_unauthorized: bool = False
    enable_auto_commit: bool = True
    auto_index_persist_interval: TimeSpan = 1 * TimeSpan.SECOND


class WriterRequest(Payload):
    config: WriterConfig | None = None
    command: WriterCommand
    frame: FramePayload | None = None


class WriterResponse(Payload):
    command: WriterCommand
    end: TimeStamp | None
    err: ExceptionPayload


class WSWriterCodec(WSFramerCodec):
    def encode(self, pld: Message[WriterRequest]) -> bytes:
        if pld.type == "close" or pld.payload.command != WriterCommand.WRITE:
            data = self.lower_perf_codec.encode(pld)
            return bytes([LOW_PERF_SPECIAL_CHAR]) + data
        data = self.codec.encode(pld.payload.frame, 1)
        data = bytearray(data)
        data[0] = HIGH_PERF_SPECIAL_CHAR
        return bytes(data)

    def decode(self, data: bytes, pld_t: Message[WriterResponse]) -> object:
        if data[0] == LOW_PERF_SPECIAL_CHAR:
            return self.lower_perf_codec.decode(data[1:], pld_t)
        frame = self.codec.decode(data, 1)
        msg = Message[WriterRequest](type="data")
        msg.payload = Payload(command=WriterCommand.WRITE, frame=frame)
        return msg


def parse_writer_mode(mode: CrudeWriterMode) -> WriterMode:
    if mode == "persist_stream":
        return WriterMode.PERSIST_STREAM
    if mode == "persist":
        return WriterMode.PERSIST
    if mode == "stream":
        return WriterMode.STREAM
    if isinstance(mode, WriterMode):
        return mode
    if isinstance(mode, int):
        try:
            return WriterMode(mode)
        except:
            ...
    raise ValueError(f"invalid writer mode {mode}")


ALWAYS_INDEX_PERSIST_ON_AUTO_COMMIT: TimeSpan = TimeSpan(-1)


class WriterClosed(BaseException): ...


class Writer:
    """A writer is used to write telemetry to a set of channels in time order. It
    should not be constructed directly, and should instead be created using the Synnax
    client's open_writer method.

    The writer is a streaming protocol that is heavily optimized for performance. This
    comes at the cost of increased complexity, and should only be used directly when
    writing large volumes of data (such as recording telemetry from a sensor or
    ingesting data from a file). Simpler methods (such as client.write()) should be
    used in most cases.

    For a detailed guide on writing data to Synnax, see
    https://docs.synnaxlabs.com/reference/concepts/writes. A rough summary of the write
    process is detailed below:

    1. The writer is opened with a starting timestamp and a list of channel keys (or
    names). The writer will fail to open if the starting timestamp overlaps with any
    existing telemetry for any of the channels specified. If the writer is opened
    successfully, the caller is then free to write frames to the writer.

    2. To write a frame, the caller can use the write method and follow the validation
    rules described in the method's documentation. This process is asynchronous,
    meaning that write will return before the frame has been written to the cluster. This
    also means that the writer can accumulate an error after write is called. If the
    writer accumulates an error, subsequent calls to `write` will repeatedly throw that
    error. If write throws an error, the writer can no longer be used, and it must
    be closed and re-opened to continue writing.

    3. To commit the written frames to the database, the caller can call the commit
    method. Unlike write, commit is synchronous, meaning that it will not return until
    all frames have been committed to the database. If commit fails or the writer
    has accumulated an error through previous calls to commit or write, then commit
    will throw that error. If commit throws an error, the writer can no longer be used,
    and it must be closed and re-opened to continue writing. Commit can be called
    several times throughout a writer's lifetime, and will commit all frames written
    since the last commit.

    4. A writer MUST be closed after use in order to prevent resource leaks. Close
    should typically be called in a 'finally' block. If the writer has accumulated an
    error, close will raise the accumulated error.
    """

    _stream: Stream[WriterRequest, WriterResponse]
    _adapter: WriteFrameAdapter
    _close_exc: Exception | None = None

    start: CrudeTimeStamp

    def __init__(
        self,
        start: CrudeTimeStamp,
        client: WebsocketClient,
        adapter: WriteFrameAdapter,
        name: str = "",
        authorities: list[Authority] | Authority = Authority.ABSOLUTE,
        mode: CrudeWriterMode = WriterMode.PERSIST_STREAM,
        err_on_unauthorized: bool = False,
        enable_auto_commit: bool = True,
        auto_index_persist_interval: TimeSpan = 1 * TimeSpan.SECOND,
        use_experimental_codec: bool = True,
    ) -> None:
        self.start = start
        self._adapter = adapter
        if use_experimental_codec:
            client = client.with_codec(WSWriterCodec(adapter.codec))
        self._stream = client.stream("/frame/write", WriterRequest, WriterResponse)
        config = WriterConfig(
            control_subject=Subject(name=name, key=str(uuid4())),
            keys=self._adapter.keys,
            start=TimeStamp(self.start),
            authorities=normalize(authorities),
            mode=parse_writer_mode(mode),
            err_on_unauthorized=err_on_unauthorized,
            enable_auto_commit=enable_auto_commit,
            auto_index_persist_interval=auto_index_persist_interval,
        )
        exc = self._stream.send(
            WriterRequest(command=WriterCommand.OPEN, config=config)
        )
        if exc is not None:
            raise exc
        _, exc = self._stream.receive()
        if exc is not None:
            raise exc

    @overload
    def write(
        self, channels_or_data: ChannelKey | ChannelName, series: CrudeSeries
    ): ...

    @overload
    def write(
        self, channels_or_data: ChannelKeys | ChannelNames, series: list[CrudeSeries]
    ): ...

    @overload
    def write(
        self,
        channels_or_data: CrudeFrame,
    ): ...

    def write(
        self,
        channels_or_data: (
            ChannelName | ChannelKey | ChannelKeys | ChannelNames | CrudeFrame
        ),
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ) -> None:
        """Writes the given data to the database. The formats are listed below. Before
        we get into them, here are some important terms to know.

            1. Channel ID -> the key or name of the channel(s) you're writing to.
            2. Series or CrudeSeries -> the data for that channel, which can be
            represented as a synnax Series type, a numpy array, or a simple Python
            list. You can also provide a single numeric (or, in the case of variable
            length types, a string or JSON) value and Synnax will convert it into a
            Series for you.

        Here are the formats you can use to write data to a Synnax cluster:

            1. Channel ID and a single series: Writes the series for the given channel.
            2. A list of channel ids and their corresponding series: Assumes a
            one to one mapping of ids to series i.e. the channel id at index i
            corresponds to the series at index i.
            3. A Synnax Frame (see the Frame documentation for more).
            4. A dictionary of channel ids to series i.e. write the series for the
            given channel id.
            5. A pandas DataFrame where the columns are the channel ids and the rows
            are the series to write.
            6. A dictionary of channel ids to a single
            numeric value. Synnax will convert this into a series for you.

        There are a few important rules to keep in mind when writing data to a Synnax
        cluster:

            1. Have exactly one array for each key in the list of keys provided to the
            writer's open method.
            2. Have equal length arrays for each key.
            3. When writing to an index (i.e. TimeStamp) channel, the values must be
            monotonically increasing.

        :raises: Raises if the writer has accumulated an error. Once write throws,
        all subsequent calls to write and/or commit will also fail. At this point,
        the writer must be closed and re-opened in order to continue writing.
        """
        if self._close_exc is not None:
            raise self._close_exc
        frame = self._adapter.adapt(channels_or_data, series)
        try:
            self._exec(
                WriterRequest(command=WriterCommand.WRITE, frame=frame.to_payload()),
                timeout=0,
            )
        except TimeoutError:
            ...

    @overload
    def set_authority(self, value: CrudeAuthority) -> None:
        """Sets the authority level for all channels the writer is writing to.

        :param value: The authority level to set (0-255). Use Authority.ABSOLUTE (255)
            for exclusive control or Authority.DEFAULT (1) for standard priority.
        """
        ...

    @overload
    def set_authority(
        self,
        value: ChannelKey | ChannelName,
        authority: CrudeAuthority,
    ) -> None:
        """Sets the authority level for a single channel.

        :param value: The key or name of the channel to set authority for.
        :param authority: The authority level to set (0-255).
        """
        ...

    @overload
    def set_authority(
        self,
        value: dict[ChannelKey | ChannelName | ChannelPayload, CrudeAuthority],
    ) -> None:
        """Sets the authority level for multiple channels.

        :param value: A dictionary mapping channel keys/names to their authority levels.
        """
        ...

    def set_authority(
        self,
        value: (
            dict[ChannelKey | ChannelName | ChannelPayload, CrudeAuthority]
            | ChannelKey
            | ChannelName
            | CrudeAuthority
        ),
        authority: CrudeAuthority | None = None,
    ) -> None:
        """Sets the control authority for channels the writer is writing to. Authority
        determines which writer takes precedence when multiple writers attempt to write
        to the same channel. Higher authority values take priority over lower ones.

        This method has three call signatures:

        1. Set authority for all channels:
            writer.set_authority(Authority.ABSOLUTE)

        2. Set authority for a single channel by key or name:
            writer.set_authority("my_channel", Authority.ABSOLUTE)
            writer.set_authority(channel_key, 200)

        3. Set authority for multiple channels via dictionary:
            writer.set_authority({
                "channel_a": Authority.ABSOLUTE,
                "channel_b": 100,
            })

        Authority levels range from 0 to 255:
            - Authority.ABSOLUTE (255): Highest priority. No other writer can take
              control while this writer is active.
            - Authority.DEFAULT (1): Standard priority for normal operations.
            - Custom values (0-255): Higher values take precedence over lower ones.

        :param value: Either an authority level (int) to apply to all channels, a
            channel key/name to set authority for a single channel, or a dictionary
            mapping channels to their authority levels.
        :param authority: The authority level when setting for a single channel.
            Required when value is a channel key or name, ignored otherwise.
        :raises ValueError: If a single channel is specified without an authority value.
        :raises: Any accumulated error from previous writer operations.
        """
        if self._close_exc is not None:
            raise self._close_exc
        if isinstance(value, int) and authority is None:
            cfg = WriterConfig(keys=[], authorities=[value])
        else:
            if isinstance(value, (ChannelKey, ChannelName)):
                if authority is None:
                    raise ValueError(
                        "authority must be provided when setting a single channel"
                    )
                value = {value: authority}
            value = self._adapter.adapt_dict_keys(value)
            cfg = WriterConfig(
                keys=list(value.keys()),
                authorities=list(value.values()),
            )
        self._exec(WriterRequest(command=WriterCommand.SET_AUTHORITY, config=cfg))

    def commit(self) -> TimeStamp | None:
        """Commits the written frames to the database. Commit is synchronous, meaning
        that it will not return until all frames have been committed to the database.

        :returns: The end timestamp of the committed region, or None if no data was
            committed.
        :raises: An exception if the commit fails or any calls to previous writer
            methods have thrown an exception. Once commit fails, the writer must be
            closed and re-opened to continue use.
        """
        if self._close_exc is not None:
            raise self._close_exc
        res = self._exec(WriterRequest(command=WriterCommand.COMMIT))
        return res.end

    def close(self):
        """Closes the writer, raising any accumulated error encountered during
        operation. A writer MUST be closed after use, and this method should probably
        be placed in a 'finally' block.
        """
        return self._close(None)

    def _close(self, exc: Exception | None) -> None:
        if self._close_exc is not None:
            if isinstance(self._close_exc, WriterClosed):
                return
            raise self._close_exc
        self._close_exc = exc
        self._stream.close_send()
        while True:
            if self._close_exc is not None:
                if isinstance(self._close_exc, WriterClosed):
                    return
                raise self._close_exc
            res, exc = self._stream.receive()
            if exc is not None:
                self._close_exc = WriterClosed() if isinstance(exc, EOF) else exc
            else:
                self._close_exc = decode_exception(res.err)

    def _exec(
        self, req: WriterRequest, timeout: int | None = None
    ) -> WriterResponse | None:
        exc = self._stream.send(req)
        if exc is not None:
            return self._close(exc)
        while True:
            res, exc = self._stream.receive(timeout)
            if exc is not None:
                return self._close(exc)
            exc = decode_exception(res.err)
            if exc is not None:
                return self._close(exc)
            if res.command == req.command:
                return res

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.close()
