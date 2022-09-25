from freighter import HTTPClientFactory, Payload, UnaryClient

from synnax.telem import DATA_TYPE_UNKNOWN, Rate, UnparsedDataType, UnparsedRate

from .payload import ChannelPayload


class _Request(Payload):
    channel: ChannelPayload
    count: int


class _Response(Payload):
    channels: list[ChannelPayload]


class ChannelCreator:
    _ENDPOINT = "/channel/create"
    client: UnaryClient

    def __init__(self, client: HTTPClientFactory):
        self.client = client.post_client()

    def create(
        self,
        name: str = "",
        node_id: int = 0,
        rate: UnparsedRate = Rate(0),
        data_type: UnparsedDataType = DATA_TYPE_UNKNOWN,
    ) -> ChannelPayload:
        return self.create_n(
            ChannelPayload(name=name, node_id=node_id, rate=rate, data_type=data_type),
            1,
        )[0]

    def create_n(self, channel: ChannelPayload, count: int = 1) -> list[ChannelPayload]:
        req = _Request(channel=channel, count=count)
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        return res.channels
