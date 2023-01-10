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

from numpy import ndarray
from rich.pretty import Pretty

from synnax.exceptions import ValidationError
from synnax.framer import FramerClient
from synnax.telem import (
    DATA_TYPE_UNKNOWN,
    Rate,
    UnparsedDataType,
    UnparsedDensity,
    UnparsedRate,
    UnparsedTimeStamp,
)

from .create import ChannelCreator
from .payload import ChannelPayload
from .retrieve import ChannelRetriever


class Channel(ChannelPayload):
    """Represents a Channel in a Synnax database."""

    segment_client: FramerClient | None = None

    class Config:
        arbitrary_types_allowed = True

    def __init__(
        self,
        data_type: UnparsedDataType,
        rate: UnparsedRate = 0,
        name: str = "",
        node_id: int = 0,
        key: str = "",
        density: UnparsedDensity = 0,
        is_index: bool = False,
        index: str = "",
        segment_client: FramerClient = None,
    ):
        super().__init__(
            data_type=data_type,
            rate=rate,
            name=name,
            node_id=node_id,
            key=key,
            density=density,
            is_index=is_index,
            index=index,
        )
        self.segment_client = segment_client

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

    def read(self, start: UnparsedTimeStamp, end: UnparsedTimeStamp) -> ndarray:
        """Reads telemetry from the channel between the two timestamps.

        :param start: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :returns: A numpy array containing the retrieved telemetry from the database.
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        return self.segment_client.read(self.key, start, end).data

    def write(self, start: UnparsedTimeStamp, data: ndarray):
        """Writes telemetry to the channel starting at the given timestamp.

        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        self.segment_client.write(self.key, start, data)

    def _assert_created(self):
        if not self.segment_client:
            raise ValidationError(
                "cannot read from a channel that has not been created"
            )

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

    _segment_client: FramerClient
    _retriever: ChannelRetriever
    _creator: ChannelCreator

    def __init__(
        self,
        segment_client: FramerClient,
        retriever: ChannelRetriever,
        creator: ChannelCreator,
    ):
        self._segment_client = segment_client
        self._retriever = retriever
        self._creator = creator

    def create_many(self, channels: list[Channel]) -> list[Channel]:
        """Creates all channels in the given list."""
        return self._sugar(*self._creator.create_many([c._payload() for c in channels]))

    def create(
        self,
        name: str = "",
        node_id: int = 0,
        rate: UnparsedRate = Rate(0),
        data_type: UnparsedDataType = DATA_TYPE_UNKNOWN,
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
        return self._sugar(
            self._creator.create(
                name=name,
                node_id=node_id,
                rate=rate,
                data_type=data_type,
                index=index,
                is_index=is_index,
            )
        )[0]

    def retrieve(self, key: str = None, name: str = None) -> Channel:
        """Retrieves channels with the given keys.

        :param keys: The list of keys to retrieve channels for.
        :raises QueryError: If any of the channels can't be found.
        :returns: A list of retrieved Channels.
        """
        return self._sugar(self._retriever.retrieve(key, name))[0]

    def filter(
        self,
        keys: list[str] = None,
        names: list[str] = None,
        node_id: int = None,
    ) -> list[Channel]:
        """Filters channels using the given parameters.

        :param kwargs: The parameters to filter channels by.
        :returns: A list of channels that match the given parameters.
        """
        return self._sugar(*self._retriever.filter(keys, names, node_id))

    def _sugar(self, *channels: ChannelPayload) -> list[Channel]:
        return [
            Channel(**c.dict(), segment_client=self._segment_client) for c in channels
        ]
