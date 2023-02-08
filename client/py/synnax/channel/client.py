#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload, Literal
from numpy import ndarray
from pydantic import PrivateAttr

from synnax.exceptions import ValidationError, QueryError
from synnax.framer import FrameClient
from synnax.telem import (
    Rate,
    Density,
    TimeRange,
    UnparsedDataType,
    UnparsedDensity,
    UnparsedRate,
    UnparsedTimeStamp,
    DataType,
)

from synnax.channel.create import ChannelCreator
from synnax.channel.payload import ChannelPayload
from synnax.channel.retrieve import ChannelRetriever


class Channel(ChannelPayload):
    """Represents a Channel in a Synnax database."""

    __frame_client: FrameClient | None = PrivateAttr(None)

    class Config:
        arbitrary_types_allowed = True

    def __init__(
        self,
        data_type: UnparsedDataType,
        rate: UnparsedRate = 0,
        name: str = "",
        node_id: int = 0,
        key: str = "",
        is_index: bool = False,
        index: str = "",
        density: UnparsedDensity = 0,
        frame_client: FrameClient | None = None,
    ):
        super().__init__(
            data_type=DataType(data_type),
            rate=Rate(rate),
            name=name,
            node_id=node_id,
            key=key,
            density=Density(density),
            is_index=is_index,
            index=index,
        )
        self.__frame_client = frame_client

    def _payload(self) -> ChannelPayload:
        return ChannelPayload(
            data_type=self.data_type,
            density=self.density,
            rate=self.rate,
            name=self.name,
            node_id=self.node_id,
            key=self.key,
            index=self.index,
            is_index=self.is_index,
        )

    def read(
        self, start: UnparsedTimeStamp, end: UnparsedTimeStamp
    ) -> tuple[ndarray, TimeRange]:
        """Reads telemetry from the channel between the two timestamps.

        :param start: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :returns: A numpy array containing the retrieved telemetry from the database.
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        return self._frame_client.read(start, end, key=self.key)

    def write(self, start: UnparsedTimeStamp, data: ndarray):
        """Writes telemetry to the channel starting at the given timestamp.

        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        self._frame_client.write(start, data, key=self.key)

    @property
    def _frame_client(self) -> FrameClient:
        if self.__frame_client is None:
            raise ValidationError(
                "Cannot read from a channel that has not been created."
            )
        return self.__frame_client

    def __hash__(self):
        return hash(self.key)

    def __eq__(self, other):
        return self.key == other.key

    def __str__(self):
        base = f"{self.name} ({self.data_type})"
        if self.rate != 0:
            base += f" @ {self.rate}Hz"
        return base


class ChannelClient:
    """The core client class for executing channel operations against a Synnax cluster."""

    _frame_client: FrameClient
    _retriever: ChannelRetriever
    _creator: ChannelCreator

    def __init__(
        self,
        frame_client: FrameClient,
        retriever: ChannelRetriever,
        creator: ChannelCreator,
    ):
        self._frame_client = frame_client
        self._retriever = retriever
        self._creator = creator

    @overload
    def create(
        self,
        channels: None = None,
        data_type: UnparsedDataType = DataType.UNKNOWN,
        name: str = "",
        node_id: int = 0,
        rate: UnparsedRate = Rate(0),
        index: str = "",
        is_index: bool = False,
    ) -> Channel:
        """Creates a channel using the given template.

        :param name: The name of the channel to create.
        :param rate: The sample rate of the channel in Hz.
        :param data_type: The data type of the channel. Can be any type in
        UnparsedDataType, such as np.float64 or np.int64,
        :param node_id: The node that holds the lease on the channel. If you don't know
        what this is, don't worry about it.
        :returns: The created channel.
        """
        ...

    @overload
    def create(self, channels: Channel) -> Channel:
        ...

    @overload
    def create(self, channels: list[Channel]) -> list[Channel]:
        ...

    def create(
        self,
        channels: Channel | list[Channel] | None = None,
        data_type: UnparsedDataType = DataType.UNKNOWN,
        name: str = "",
        node_id: int = 0,
        rate: UnparsedRate = Rate(0),
        index: str = "",
        is_index: bool = False,
    ) -> Channel | list[Channel]:
        if channels is None:
            _channels = [
                ChannelPayload(
                    name=name,
                    node_id=node_id,
                    rate=Rate(rate),
                    data_type=DataType(data_type),
                    index=index,
                    is_index=is_index,
                )
            ]
        elif isinstance(channels, Channel):
            _channels = [channels._payload()]
        else:
            _channels = [c._payload() for c in channels]
        created = self._sugar(self._creator.create(_channels))
        return created if isinstance(channels, list) else created[0]

    @overload
    def retrieve(self, key: str | None = None, name: str | None = None) -> Channel:
        ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
        include_not_found: Literal[False] = False,
    ) -> list[Channel]:
        ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
        include_not_found: Literal[True] = True,
    ) -> tuple[list[Channel], list[str]]:
        ...

    def retrieve(
        self,
        key: str | None = None,
        name: str | None = None,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
        include_not_found: bool = False,
    ) -> Channel | list[Channel] | tuple[list[Channel], list[str]]:
        res = self._retriever.retrieve(
            key=key,
            name=name,
            keys=keys, 
            names=names, 
            node_id=node_id, 
            include_not_found=include_not_found,
        )
        if res is None:
            if key is not None:
                raise QueryError(f"Channel with key {key} not found.")
            if name is not None:
                raise QueryError(f"Channel with name {name} not found.")
            raise QueryError("No channels found.")
        if isinstance(res, ChannelPayload):
            return self._sugar([res])[0]
        if isinstance(res, tuple):
            return self._sugar(res[0]), res[1]
        return self._sugar(res)

    def _sugar(self, channels: list[ChannelPayload]) -> list[Channel]:
        return [Channel(**c.dict(), frame_client=self._frame_client) for c in channels]
