from __future__ import annotations

from freighter import HTTPClientFactory, Payload, UnaryClient

from .payload import ChannelPayload


class _Request(Payload):
    keys: list[str] | None = None
    node_id: int | None = None
    names: list[str] | None = None


class _Response(Payload):
    channels: list[ChannelPayload] | None = []


class ChannelRetriever:
    _ENDPOINT = "/channel/retrieve"
    client: UnaryClient

    def __init__(self, client: HTTPClientFactory):
        self.client = client.get_client()

    def retrieve(self, keys: list[str]) -> list[ChannelPayload]:
        return self._retrieve(_Request(keys=keys))

    def retrieve_by_name(self, names: list[str]) -> list[ChannelPayload]:
        return self._retrieve(_Request(names=names))

    def retrieve_by_node_id(self, node_id: int) -> list[ChannelPayload]:
        return self._retrieve(_Request(node_id=node_id))

    def _retrieve(self, req: _Request) -> list[ChannelPayload]:
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        return res.channels
