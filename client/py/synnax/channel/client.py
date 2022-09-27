from numpy import ndarray

from synnax.segment.client import SegmentClient as SegmentClient
from synnax.telem import (
    DATA_TYPE_UNKNOWN,
    Rate,
    UnparsedDataType,
    UnparsedRate,
    UnparsedTimeStamp,
)

from .create import ChannelCreator
from .payload import ChannelPayload
from .retrieve import ChannelRetriever


class Channel(ChannelPayload):
    """Represents a Channel in a Synnax database. It should not be instantiated directly,
    and should be created or retrieved using the Synnax Client.
    """

    segment_client: SegmentClient | None = None

    class Config:
        arbitrary_types_allowed = True

    def __init__(self, pld: ChannelPayload, segment_client: SegmentClient):
        super().__init__(**pld.dict())
        self.segment_client = segment_client

    def _payload(self) -> ChannelPayload:
        return ChannelPayload(
            data_type=self.data_type,
            density=self.density,
            rate=self.rate,
            name=self.name,
            node_id=self.node_id,
            key=self.key,
        )

    def read(self, start: UnparsedTimeStamp, end: UnparsedTimeStamp) -> ndarray:
        """Reads telemetry from the channel between the two timestamps.

        :param start: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :returns: A numpy array containing the retrieved telemetry from the database.
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        return self.segment_client.read(self.key, start, end)

    def write(self, start: UnparsedTimeStamp, data: ndarray):
        """Writes telemetry to the channel starting at the given timestamp.

        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        self.segment_client.write(self.key, start, data)


class ChannelClient:
    """The core client class for executing channel operations against a Synnax cluster."""

    _segment_client: SegmentClient
    _retriever: ChannelRetriever
    _creator: ChannelCreator

    def __init__(
        self,
        segment_client: SegmentClient,
        retriever: ChannelRetriever,
        creator: ChannelCreator,
    ):
        self._segment_client = segment_client
        self._retriever = retriever
        self._creator = creator

    def create_n(
        self,
        name: str = "",
        rate: UnparsedRate = Rate(0),
        data_type: UnparsedDataType = DATA_TYPE_UNKNOWN,
        node_id: int = 0,
        count: int = 1,
    ) -> list[Channel]:
        """Creates N channels using the given parameters as a template.

        :param name: The name of the channel to create.
        :param rate: The sample rate of the channel in Hz.
        :param data_type: The data type of the channel. Can be any type in
        UnparsedDataType, such as np.float64 or np.int64,
        :param node_id: The node that holds the lease on the channel. If you don't know
        what this is, don't worry about it.
        :param count: The number of channels to create.
        :returns: A list of created channels.
        """
        return self._sugar(
            *self._creator.create_n(
                ChannelPayload(
                    name=name,
                    node_id=node_id,
                    rate=rate,
                    data_type=data_type,
                ),
                count,
            )
        )

    def create(
        self,
        rate: UnparsedRate,
        data_type: UnparsedDataType,
        name: str = "",
        node_id: int = 0,
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
        return self._sugar(self._creator.create(name, node_id, rate, data_type))[0]

    def retrieve(self, keys: list[str]) -> list[Channel]:
        """Retrieves channels with the given keys.

        :param keys: The list of keys to retrieve channels for.
        :raises QueryError: If any of the channels can't be found.
        :returns: A list of retrieved Channels.
        """
        return self._sugar(*self._retriever.retrieve(keys))

    def retrieve_by_name(self, names: list[str]) -> list[Channel]:
        """Retrieves channels with the given names.

        :param names: The list of names to retrieve channels for.
        :returns: A list of retrieved channels matching the given name.
        """
        return self._sugar(*self._retriever.retrieve_by_name(names))

    def retrieve_by_node_id(self, node_id: int) -> list[Channel]:
        """Retrieves channels whose lease node is the given node_id.

        :param node_id: The node id to retrieve the channels for.
        :returns: A list of retrieved channels matching the given node id.
        """
        return self._sugar(*self._retriever.retrieve_by_node_id(node_id))

    def _sugar(self, *channels: ChannelPayload) -> list[Channel]:
        return [Channel(c, self._segment_client) for c in channels]
