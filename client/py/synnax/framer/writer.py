#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from enum import Enum
from uuid import uuid4
from warnings import warn

from freighter import EOF, Payload, Stream, StreamClient, decode_exception
from numpy import can_cast as np_can_cast
from pandas import DataFrame
from pandas import concat as pd_concat

from synnax import io
from synnax.channel.payload import ChannelKey, ChannelKeys, ChannelName, ChannelNames
from synnax.exceptions import Field, ValidationError
from synnax.framer.adapter import WriteFrameAdapter
from synnax.framer.frame import Frame, FramePayload
from synnax.telem import CrudeSeries, CrudeTimeStamp, DataType, TimeSpan, TimeStamp
from synnax.telem.control import Authority, Subject
from synnax.util.normalize import normalize


class _Command(int, Enum):
    OPEN = 0
    WRITE = 1
    COMMIT = 2
    ERROR = 3
    SET_AUTHORITY = 4
    SET_MODE = 5


class WriterMode(int, Enum):
    PERSIST_STREAM = 1
    PERSIST_ONLY = 2
    STREAM_ONLY = 3


class _Config(Payload):
    authorities: list[int]
    control_subject: Subject
    start: TimeStamp | None = None
    keys: ChannelKeys
    mode: WriterMode
    enable_auto_commit: bool
    auto_index_persist_interval: TimeSpan


class _Request(Payload):
    config: _Config | None = None
    command: _Command
    frame: FramePayload | None = None


class _Response(Payload):
    command: _Command
    ack: bool
    error: str | None
    end: TimeStamp | None


ALWAYS_INDEX_PERSIST_ON_AUTO_COMMIT: TimeSpan = TimeSpan(-1)


class Writer:
    """CoreWriter is used to write a range of telemetry to a set of channels in time
    order. It should not be instantiated directly, and should instead be created using
    the Synnax client.

    The writer is a streaming protocol that is heavily optimized for performance. This
    comes at the cost of increased complexity, and should only be used directly when
    writing large volumes of data (such as recording telemetry from a sensor or
    ingesting data from a file). Simpler methods (such as the frame writer's write
    method)
    should be used in most cases.

    The protocol is as follows:

    1. The writer is opened with a starting timestamp and a list of channel keys. The
    writer will fail to open if the starting timestamp overlaps with any existing
    telemetry for any of the channels specified. If the writer is opened successfully,
    the caller is then free to write frames to the writer.

    2. To writer a frame, the caller can use the write method and follow the validation
    rules described in its method's documentation. This process is asynchronous, meaning
    that write will return before the frame has been written to the cluster. This also
    means that the writer can accumulate an error after write is called. If the writer
    accumulates an error, all subsequent write and commit calls will return False. The
    caller can check for errors by calling the error method, which returns the
    accumulated error and resets the writer for future use. The caller can also check
    for errors by closing the writer, which will raise any accumulated error.

    3. To commit the written frames to the database, the caller can call the commit
    method. Unlike write, commit is synchronous, meaning that it will not return until
    all frames have been committed to the database. If the writer has accumulated an
    error, commit will return False. After the caller acknowledges the error, they can
    attempt to commit again. Commit can be called several times throughout a writer's
    lifetime, and will commit all frames written since the last commit.

    4. A writer MUST be closed after use in order to prevent resource leaks. Close
    should typically be called in a 'finally' block. If the writer has accumulated an
    error, close will raise the accumulated error.
    """

    __ENDPOINT = "/frame/write"
    __stream: Stream[_Request, _Response]
    __adapter: WriteFrameAdapter
    __suppress_warnings: bool = False
    __strict: bool = False
    __mode: WriterMode

    start: CrudeTimeStamp

    def __init__(
        self,
        start: CrudeTimeStamp,
        client: StreamClient,
        adapter: WriteFrameAdapter,
        name: str = "",
        authorities: list[Authority] | Authority = Authority.ABSOLUTE,
        suppress_warnings: bool = False,
        strict: bool = False,
        mode: WriterMode = WriterMode.PERSIST_STREAM,
        enable_auto_commit: bool = False,
        auto_index_persist_interval: TimeSpan = 1 * TimeSpan.SECOND,
    ) -> None:
        self.start = start
        self.__adapter = adapter
        self.__suppress_warnings = suppress_warnings
        self.__strict = strict
        self.__mode = mode
        self.__stream = client.stream(self.__ENDPOINT, _Request, _Response)
        self.__open(name, authorities, enable_auto_commit, auto_index_persist_interval)

    def __open(
        self,
        name: str,
        authorities: list[Authority],
        enable_auto_commit: bool,
        auto_index_persist_interval: TimeSpan,
    ) -> None:
        config = _Config(
            control_subject=Subject(name=name, key=str(uuid4())),
            keys=self.__adapter.keys,
            start=TimeStamp(self.start),
            authorities=normalize(authorities),
            mode=self.__mode,
            enable_auto_commit=enable_auto_commit,
            auto_index_persist_interval=auto_index_persist_interval,
        )
        self.__stream.send(_Request(command=_Command.OPEN, config=config))
        _, exc = self.__stream.receive()
        if exc is not None:
            raise exc

    def write(
        self,
        channels_or_data: ChannelName
        | ChannelKey
        | ChannelKeys
        | ChannelNames
        | Frame
        | dict[ChannelKey | ChannelName, CrudeSeries]
        | DataFrame,
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ) -> bool:
        """Writes the given frame to the database. The provided frame must:

        :param channels_or_data: The data to write to the database. The frame must:

            1. Have exactly one array for each key in the list of keys provided to the
            writer's open method.
            2. Have equal length arrays for each key.
            3. When writing to an index (i.e. TimeStamp) channel, the values must be
            monotonically increasing.

        :returns: False if the writer has accumulated an error. If this is the case,
        the caller should acknowledge the error by calling the error method or closing
        the writer.
        """
        if self.__stream.received():
            return False

        frame = self.__adapter.adapt(channels_or_data, series)
        self.__check_keys(frame)
        self.__prep_data_types(frame)

        err = self.__stream.send(
            _Request(command=_Command.WRITE, frame=frame.to_payload())
        )
        if err is not None:
            raise err
        return True

    def set_authority(self, value: dict[ChannelKey, Authority]) -> bool:
        err = self.__stream.send(
            _Request(
                command=_Command.SET_AUTHORITY,
                config=_Config(
                    keys=list(value.keys()),
                    authorities=list(value.values()),
                ),
            )
        )
        if err is not None:
            raise err
        while True:
            res, err = self.__stream.receive()
            if err is not None:
                raise err
            if res.command == _Command.SET_AUTHORITY:
                return res.ack

    def set_mode(self, value: WriterMode) -> bool:
        err = self.__stream.send(
            _Request(
                command=_Command.SET_MODE,
                config=_Config(mode=value),
            )
        )
        if err is not None:
            raise err
        while True:
            res, err = self.__stream.receive()
            if err is not None:
                raise err
            if res.command == _Command.SET_MODE:
                return res.ack

    def commit(self) -> tuple[TimeStamp, bool]:
        """Commits the written frames to the database. Commit is synchronous, meaning
        that it will not return until all frames have been committed to the database.

        :returns: False if the commit failed due to an error. In this case, the caller
        should acknowledge the error by calling the error method or closing the writer.
        After the error is acknowledged, the caller can attempt to commit again.
        """
        if self.__stream.received():
            return TimeStamp.ZERO, False
        err = self.__stream.send(_Request(command=_Command.COMMIT))
        if err is not None:
            raise err

        while True:
            res, err = self.__stream.receive()
            if err is not None:
                raise err
            if res.command == _Command.COMMIT:
                return res.end, res.ack

    def error(self) -> Exception | None:
        """
        :returns: The exception that the writer has accumulated, if any. If the writer
        has not accumulated an error, this method will return None. This method will
        clear the writer's error state, allowing the writer to be used again.
        """
        self.__stream.send(_Request(command=_Command.ERROR))

        while True:
            res, err = self.__stream.receive()
            if err is not None:
                raise err
            assert res is not None
            if res.command == _Command.ERROR:
                return decode_exception(res.error)

    def close(self):
        """Closes the writer, raising any accumulated error encountered during
        operation. A writer MUST be closed after use, and this method should probably
        be placed in a 'finally' block.
        """
        self.__stream.close_send()
        while True:
            res, err = self.__stream.receive()
            if err is None:
                assert res is not None
                err = decode_exception(res.error)

            if err is not None:
                if isinstance(err, EOF):
                    return
                raise err

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.close()

    def __check_keys(self, frame: Frame):
        missing = set(self.__adapter.keys) - set(frame.channels)
        extra = set(frame.channels) - set(self.__adapter.keys)
        if missing and extra:
            raise ValidationError(
                Field(
                    "keys",
                    f"frame is missing keys {missing} and has extra keys {extra}",
                )
            )
        elif extra:
            raise ValidationError(Field("keys", f"frame has extra keys {extra}"))

    def __prep_data_types(self, frame: Frame):
        for i, (label, series) in enumerate(frame.items()):
            ch = self.__adapter.retriever.retrieve(label)[0]  # type: ignore
            if series.data_type != ch.data_type:
                if (
                    not np_can_cast(series.data_type.np, ch.data_type.np)
                    or self.__strict
                ):
                    raise ValidationError(
                        Field(
                            str(label),
                            f"""label {label} has type {series.data_type} but channel {ch.key}
                                            expects type {ch.data_type}""",
                        )
                    )
                elif not self.__suppress_warnings and not (
                    ch.data_type == DataType.TIMESTAMP
                    and series.data_type == DataType.INT64
                ):
                    warn(
                        f"""Series for channel {ch.name} has type {series.data_type} but channel
                        expects type {ch.data_type}. We can safely convert between the two,
                        but this can cause performance degradations and is not recommended.
                        To suppress this warning, set suppress_warnings=True when constructing
                        the writer. To raise an error instead, set strict=True when constructing
                        the writer."""
                    )
                frame.series[i] = series.astype(ch.data_type)


class BufferedWriter(io.DataFrameWriter):
    """BufferedWriter extends the Writer class by buffering
    writes to the underlying stream. This can improve performance by reducing the
    number of round trips to the server.
    """

    size_threshold: int
    time_threshold: TimeSpan
    last_flush: TimeStamp
    _wrapped: Writer
    _buf: DataFrame

    def __init__(
        self,
        wrapped: Writer,
        size_threshold: int = int(1e6),
        time_threshold: TimeSpan = TimeSpan.MAX,
    ) -> None:
        self._wrapped = wrapped
        self._buf = DataFrame()
        self.last_flush = TimeStamp.now()
        self.size_threshold = size_threshold
        self.time_threshold = time_threshold

    def _(self) -> io.DataFrameWriter:
        return self

    def write(self, frame: DataFrame):
        self._buf = pd_concat([self._buf, frame], ignore_index=True)
        if self._exceeds_any:
            self.__flush()

    def close(self):
        self.__flush()
        self._wrapped.close()

    @property
    def _exceeds_any(self) -> bool:
        return (
            len(self._buf) * len(self._buf.columns) >= self.size_threshold
            or TimeSpan.since(self.last_flush) >= self.time_threshold
        )

    def __flush(self):
        self._wrapped.write(self._buf)
        self._wrapped.commit()
        self.last_flush = TimeStamp.now()
        self._buf = DataFrame()
