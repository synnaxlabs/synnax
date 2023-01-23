#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from enum import Enum

from freighter import (
    EOF,
    ExceptionPayload,
    Payload,
    Stream,
    StreamClient,
    decode_exception,
)
from pandas import DataFrame

from synnax.channel.payload import ChannelPayload
from synnax.channel.registry import ChannelRegistry
from synnax.exceptions import Field, GeneralError, ValidationError
from synnax.telem import TimeStamp, UnparsedTimeStamp

from .payload import BinaryFrame, pandas_to_frame


class _Command(int, Enum):
    NONE = 0
    WRITE = 1
    COMMIT = 2
    ERROR = 3


class _Config(Payload):
    keys: list[str]
    start: TimeStamp


class _Request(Payload):
    command: _Command
    config: _Config = None
    frame: BinaryFrame = None


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
    client: StreamClient
    stream: Stream[_Request, _Response] | None
    keys: list[str]

    def __init__(self, client: StreamClient) -> None:
        self.client = client

    def open(self, start: UnparsedTimeStamp, keys: list[str]):
        """Opens the writer to write a range of telemetry starting at the given time.

        :param start: The starting timestamp of the new range to write to. If start
        overlaps with existing telemetry, the writer will fail to open.
        :param keys: A list of keys representing the channels the writer will write to.
        All frames written to the writer must have exactly one array for each key in
        this list.
        """
        self.keys = keys
        self.stream = self.client.stream(self._ENDPOINT, _Request, _Response)
        self.stream.send(
            _Request(
                command=_Command.NONE, config=_Config(keys=keys, start=TimeStamp(start))
            )
        )
        res, exc = self.stream.receive()
        if exc is not None:
            raise exc

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
        self._assert_open()
        if self.stream.received():
            return False

        self._check_keys(frame)
        err = self.stream.send(_Request(command=_Command.WRITE, frame=frame))
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
        if self.stream.received():
            return False
        err = self.stream.send(_Request(command=_Command.COMMIT))
        if err is not None:
            raise err

        while True:
            res, err = self.stream.receive()
            if err is not None:
                raise err
            if res.command == _Command.COMMIT:
                return res.ack

    def error(self) -> Exception:
        """
        :returns: The exception that the writer has accumulated, if any. If the writer
        has not accumulated an error, this method will return None. This method will
        clear the writer's error state, allowing the writer to be used again.
        """
        self._assert_open()
        self.stream.send(_Request(command=_Command.ERROR, open_keys=[], segments=[]))

        while True:
            res, err = self.stream.receive()
            if err is not None:
                raise err
            if res.command == _Command.ERROR:
                return decode_exception(res.error)

    def close(self):
        """Closes the writer, raising any accumulated error encountered during operation.
        A writer MUST be closed after use, and this method should probably be placed in
        a 'finally' block.
        """
        self._assert_open()
        self.stream.close_send()
        res, err = self.stream.receive()
        if err is None:
            err = decode_exception(res.error)
        if not isinstance(err, EOF):
            raise err

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
        if self.stream is None:
            raise GeneralError(
                "Writer is not open. Please open before calling write() or close()."
            )


class DataFrameWriter(FrameWriter):
    """DataFrameWriter extends the FrameWriter protocol by allowing the caller to Write
    pandas DataFrames.
    """

    registry: ChannelRegistry
    channels: list[ChannelPayload]

    def __init__(
        self,
        client: StreamClient,
        registry: ChannelRegistry,
    ) -> None:
        super().__init__(client)
        self.registry = registry
        self.channels = []

    def open(self, start: UnparsedTimeStamp, keys: list[str]):
        """Opens the writer, acquiring an exclusive lock on the given
        channels for the duration of the writer's lifetime. open must be called before
        any other writer methods.
        """
        super(DataFrameWriter, self).open(start, keys)
        self.channels = self.registry.get_n(keys)

    def write(self, frame: DataFrame):
        super(DataFrameWriter, self).write(
            pandas_to_frame(self.channels, frame).to_binary(),
        )
