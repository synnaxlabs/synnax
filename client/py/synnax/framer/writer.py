#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from enum import Enum
from warnings import warn

from pandas import DataFrame, concat as pd_concat
from numpy import can_cast as np_can_cast, ndarray
from freighter import (
    EOF,
    ExceptionPayload,
    Payload,
    Stream,
    StreamClient,
    decode_exception,
)

from synnax import io
from synnax.channel.payload import ChannelPayload
from synnax.channel.retrieve import ChannelRetriever
from synnax.exceptions import Field, GeneralError, ValidationError
from synnax.telem import TimeSpan, TimeStamp, UnparsedTimeStamp, NumpyArray
from synnax.framer.payload import BinaryFrame, NumpyFrame
from synnax.util.flatten import flatten


class _Command(int, Enum):
    OPEN = 0
    WRITE = 1
    COMMIT = 2
    ERROR = 3


class _Config(Payload):
    keys: list[str]
    start: TimeStamp


class _Request(Payload):
    command: _Command
    config: _Config | None = None
    frame: BinaryFrame | None = None


class _Response(Payload):
    command: _Command
    ack: bool
    error: ExceptionPayload


class FrameWriter:
    """CoreWriter is used to write a range of telemetry to a set of channels in time
    order. It should not be instantiated directly, and should instead be created using
    the frame client.

    The writer is a streaming protocol that is heavily optimized for performance. This
    comes at the cost of increased complexity, and should only be used directly when
    writing large volumes of data (such as recording telemetry from a sensor or ingesting
    data from a file). Simpler methods (such as the frame client's write method) should
    be used in most cases.

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

    _ENDPOINT = "/frame/write"

    __stream: Stream[_Request, _Response] | None

    client: StreamClient
    keys: list[str]
    start: UnparsedTimeStamp

    def __init__(self, client: StreamClient, start: UnparsedTimeStamp, *keys: str  | list[str]) -> None:
        self.client = client
        self.start = start
        self.keys = flatten(*keys)
        self._open()

    def _open(self):
        """Opens the writer to write a range of telemetry starting at the given time.

        :param start: The starting timestamp of the new range to write to. If start
        overlaps with existing telemetry, the writer will fail to open.
        :param keys: A list of keys representing the channels the writer will write to.
        All frames written to the writer must have exactly one array for each key in
        this list.
        """
        self.__stream = self.client.stream(self._ENDPOINT, _Request, _Response)
        self._stream.send(
            _Request(
                command=_Command.OPEN,
                config=_Config(keys=self.keys, start=TimeStamp(self.start))
            )
        )
        _, exc = self._stream.receive()
        if exc is not None:
            raise exc

    @property
    def _stream(self) -> Stream[_Request, _Response]:
        self._assert_open()
        assert self.__stream is not None
        return self.__stream

    def write(self, frame: BinaryFrame) -> bool:
        """Writes the given frame to the database. The provided frame must:

        :param frame: The frame to write to the database. The frame must:

            1. Have exactly one array for each key in the list of keys provided to the
            writer's open method.
            2. Have equal length arrays for each key.
            3. When writing to an index (i.e. TimeStamp) channel, the values must be
            monotonically increasing.

        :returns: False if the writer has accumulated an error. If this is the case,
        the caller should acknowledge the error by calling the error method or closing
        the writer.
        """
        if self._stream.received():
            return False

        self._check_keys(frame)
        err = self._stream.send(_Request(command=_Command.WRITE, frame=frame))
        if err is not None:
            raise err
        return True

    def commit(self) -> bool:
        """Commits the written frames to the database. Commit is synchronous, meaning
        that it will not return until all frames have been committed to the database.

        :returns: False if the commit failed due to an error. In this case, the caller
        should acknowledge the error by calling the error method or closing the writer.
        After the error is acknowledged, the caller can attempt to commit again.
        """
        self._assert_open()
        if self._stream.received():
            return False
        err = self._stream.send(_Request(command=_Command.COMMIT))
        if err is not None:
            raise err

        while True:
            res, err = self._stream.receive()
            if err is not None:
                raise err
            assert res is not None
            if res.command == _Command.COMMIT:
                return res.ack

    def error(self) -> Exception | None:
        """
        :returns: The exception that the writer has accumulated, if any. If the writer
        has not accumulated an error, this method will return None. This method will
        clear the writer's error state, allowing the writer to be used again.
        """
        self._assert_open()
        self._stream.send(_Request(command=_Command.ERROR))

        while True:
            res, err = self._stream.receive()
            if err is not None:
                raise err
            assert res is not None
            if res.command == _Command.ERROR:
                return decode_exception(res.error)

    def close(self):
        """Closes the writer, raising any accumulated error encountered during operation.
        A writer MUST be closed after use, and this method should probably be placed in
        a 'finally' block.
        """
        self._stream.close_send()
        res, err = self._stream.receive()
        if err is None:
            assert res is not None
            err = decode_exception(res.error)
        if err is not None and not isinstance(err, EOF):
            raise err

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.close()

    def _check_keys(self, frame: BinaryFrame):
        missing = set(self.keys) - set(frame.keys)
        extra = set(frame.keys) - set(self.keys)
        if missing and extra:
            raise ValidationError(
                Field(
                    "keys",
                    f"frame is missing keys {missing} and has extra keys {extra}",
                )
            )
        elif missing:
            raise ValidationError(Field("keys", f"frame is missing keys {missing}"))
        elif extra:
            raise ValidationError(Field("keys", f"frame has extra keys {extra}"))

    def _assert_open(self):
        if self.__stream is None:
            raise GeneralError(
                "Writer is not open. Please open before calling write() or close()."
            )


class DataFrameWriter(FrameWriter, io.DataFrameWriter):
    """DataFrameWriter extends the FrameWriter protocol by allowing the caller to Write
    pandas DataFrames.
    """

    _channel_retriever: ChannelRetriever
    _strict: bool
    _suppress_warnings: bool
    _skip_invalid: bool
    _channels: list[ChannelPayload]

    def __init__(
        self,
        client: StreamClient,
        channels: ChannelRetriever,
        start: UnparsedTimeStamp,
        *keys_or_names: str | list[str],
        strict: bool = False,
        suppress_warnings: bool = False,
    ) -> None:
        self._channel_retriever = channels
        flat = flatten(*keys_or_names)
        self._channels = self._channel_retriever.retrieve(flat)
        super().__init__(client, start, [ch.key for ch in self._channels])
        self._strict = strict
        self._suppress_warnings = suppress_warnings

    def write(self, frame: DataFrame):
        super(DataFrameWriter, self).write(self._convert(frame))

    def _convert(self, df: DataFrame) -> BinaryFrame:
        np_fr = NumpyFrame()
        for ch in self._channels:
            col, arr = self._retrieve(ch, df)
            np_data = self._prep_arr(arr, ch, col)
            np_fr.append(ch.key, NumpyArray(data=np_data, data_type=ch.data_type))
        return np_fr.to_binary()

    def _retrieve(self, ch: ChannelPayload, df: DataFrame) -> tuple[str, ndarray]:
        v = df.get(ch.key, None)
        if v is None:
            v = df.get(ch.name, None)
            if v is None:
                raise ValidationError(
                    Field(
                        ch.name,
                        f"frame is missing {self._mode.value} entry for channel {ch.key}: {ch.name}",
                    )
                )
        return v, v.to_numpy()

    def _prep_arr(self, arr: ndarray, ch: ChannelPayload, col: str):
        ch_dt = ch.data_type.np
        if arr.dtype != ch_dt:
            if not np_can_cast(arr.dtype, ch_dt):
                raise ValidationError(
                    Field(
                        col,
                        f"""column {col} has type {arr.dtype} but channel {ch.key}
                        expects type {ch_dt}""",
                    )
                )
            elif not self._suppress_warnings:
                warn(
                    f"""column {col} has type {arr.dtype} but channel {ch.key} expects
                    type {ch_dt}. We can safely convert between the two, but this can
                    cause performance degradations and is not recommended. To suppress
                    this warning, set suppress_warnings=True when constructing the
                    writer. To raise an error instead, set strict=True when constructing
                    the writer."""
                )
        return arr.astype(ch_dt)


class BufferedDataFrameWriter(io.DataFrameWriter):
    """BufferedDataFrameWriter extends the DataFrameWriter protocol by buffering
    writes to the underlying stream. This can improve performance by reducing the
    number of round trips to the server.
    """

    size_threshold: int
    time_threshold: TimeSpan
    last_flush: TimeStamp
    _wrapped: DataFrameWriter
    _buf: DataFrame

    def __init__(
        self,
        wrapped: DataFrameWriter,
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
            self._flush()

    def close(self):
        self._flush()
        self._wrapped.close()

    @property
    def _exceeds_any(self) -> bool:
        return (
            len(self._buf) * len(self._buf.columns) >= self.size_threshold
            or TimeStamp.since(self.last_flush) >= self.time_threshold
        )

    def _flush(self):
        self._wrapped.write(self._buf)
        self._wrapped.commit()
        self.last_flush = TimeStamp.now()
        self._buf = DataFrame()
