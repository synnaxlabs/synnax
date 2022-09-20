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
    segment_client: SegmentClient

    def __init__(self, pld: ChannelPayload, segment_client: SegmentClient):
        super().__init__(pld.dict())
        self.segment_client = segment_client

    def payload(self) -> ChannelPayload:
        return ChannelPayload(
            data_type=self.data_type,
            density=self.density,
            rate=self.rate,
            name=self.name,
            node_id=self.node_id,
            key=self.key,
        )

    def read(self, start: UnparsedTimeStamp, end: UnparsedTimeStamp) -> ndarray:
        return self.segment_client.read(self.key, start, end)

    def write(self, start: UnparsedTimeStamp, data: ndarray):
        self.segment_client.write(self.key, start, data)


class ChannelClient:
    segment_client: SegmentClient
    retriever: ChannelRetriever
    creator: ChannelCreator

    def __init__(
        self,
        segment_client: SegmentClient,
        retriever: ChannelRetriever,
        creator: ChannelCreator,
    ):
        self.segment_client = segment_client
        self.retriever = retriever
        self.creator = creator

    def create_n(self, channel: Channel, count: int = 1) -> list[Channel]:
        return self.sugar(*self.creator.create_n(channel, count))

    def create(
        self,
        name: str = "",
        node_id: int = 0,
        rate: UnparsedRate = Rate(0),
        data_type: UnparsedDataType = DATA_TYPE_UNKNOWN,
    ) -> Channel:
        return self.sugar(self.creator.create(name, node_id, rate, data_type))[0]

    def retrieve(self, keys: list[str]) -> list[Channel]:
        return self.sugar(*self.retriever.retrieve(keys))

    def retrieve_by_name(self, names: list[str]) -> list[Channel]:
        return self.sugar(*self.retriever.retrieve_by_name(names))

    def retrieve_by_node_id(self, node_id: int) -> list[Channel]:
        return self.sugar(*self.retriever.retrieve_by_node_id(node_id))

    def sugar(self, *channels: ChannelPayload) -> list[Channel]:
        return [Channel(c, self.segment_client) for c in channels]
