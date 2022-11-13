from __future__ import annotations
import typing

from freighter import HTTPClientFactory, Payload, UnaryClient

from .payload import ChannelPayload
from ..exceptions import ValidationError, QueryError


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

    def get(self, key: str = None, name: str = None) -> ChannelPayload:
        req = _Request()
        if key is None and name is None:
            raise ValidationError("Must specify a key or name")
        if key is not None:
            req.keys = [key]
        if name is not None:
            req.names = [name]
        res = self._execute(req)
        if len(res) == 0:
            raise QueryError("channel not found")
        return res[0]

    def filter(
        self,
        keys: list[str] = None,
        names: list[str] = None,
        node_id: int = None,
    ) -> list[ChannelPayload]:
        return self._execute(_Request(keys=keys, names=names))

    def _execute(self, req: _Request) -> list[ChannelPayload]:
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        return res.channels
