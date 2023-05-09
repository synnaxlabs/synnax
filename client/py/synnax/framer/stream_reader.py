#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import Payload, ExceptionPayload, Stream, StreamClient, EOF

from synnax.channel.payload import Keys, KeysOrNames
from synnax.channel.retrieve import ChannelRetriever
from synnax.exceptions import GeneralError, UnexpectedError
from synnax.framer.payload import BinaryFrame, NumpyFrame
from synnax.telem import TimeStamp, UnparsedTimeStamp
from synnax.util.flatten import flatten


class _Request(Payload):
    start: TimeStamp
    keys: Keys


class _Response(Payload):
    frame: BinaryFrame
    error: ExceptionPayload | None


class FrameStreamReader:
    _ENDPOINT = "/frame/read"
    __stream: Stream[_Request, _Response] | None

    keys: Keys
    start: UnparsedTimeStamp

    def __init__(
        self,
        client: StreamClient,
        start: UnparsedTimeStamp,
        *keys: Keys,
    ) -> None:
        self.start = start
        self.keys = flatten(*keys)
        self._open(client)

    def _open(self, client: StreamClient):
        self.__stream = client.stream(self._ENDPOINT, _Request, _Response)
        self._stream.send(_Request(keys=self.keys, start=self.start))

    def read(self) -> BinaryFrame:
        res, err = self._stream.receive()
        if err is not None:
            raise err
        return res.frame

    def close(self):
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
        return self

    def __enter__(self):
        return self

    def __next__(self):
        return self.read()

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

    @property
    def _stream(self) -> Stream[_Request, _Response]:
        self._assert_open()
        assert self.__stream is not None
        return self.__stream

    def _assert_open(self):
        if self.__stream is None:
            raise GeneralError(
                "StreamReader is not open. Please open before calling read() or close()"
            )


class NumpyStreamReader(FrameStreamReader):
    """Used to iterate over a databases telemetry in time-order. It should not be
    instantiated directly, and should instead be instantiated using the segment Client.

    Using an iterator is ideal when querying/processing large ranges of data, but is
    relatively complex and difficult to use. If you're looking to retrieve telemetry
    between two timestamps, see the segment Client read method instead.
    """

    _channels: ChannelRetriever
    _keys_or_names: list[str]

    def __init__(
        self,
        transport: StreamClient,
        channels: ChannelRetriever,
        start: TimeStamp,
        *keys_or_names: KeysOrNames,
    ):
        self._channels = channels
        self._keys_or_names = flatten(*keys_or_names)
        channels_, not_found = self._channels.retrieve(
            flatten(*keys_or_names),
            include_not_found=True,
        )
        if len(not_found) > 0:
            raise ValueError(f"Unable to find channels {not_found}")

        super().__init__(transport, start, [ch.key for ch in channels_])

    def read(self) -> NumpyFrame:
        """
        :returns: The current iterator value as a dictionary whose keys are channels
        and values are segments containing telemetry at the current iterator position.
        """
        v = super().read()
        v.keys = self._value_keys(v.keys)
        return NumpyFrame.from_binary(v)

    def _value_keys(self, keys: Keys) -> KeysOrNames:
        # We can safely ignore the none case here because we've already
        # checked that all channels can be retrieved.
        channels = self._channels.retrieve(keys)
        keys_or_names = []
        for ch in channels:
            v = [k for k in self._keys_or_names if k == ch.key or k == ch.name]
            if len(v) == 0:
                raise ValueError(f"Unexpected channel key {ch.key}")
            keys_or_names.append(v[0])
        return keys_or_names
