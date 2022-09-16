from __future__ import annotations
from dataclasses import dataclass
from typing import Optional, TypeAlias

import freighter
from .entity import Channel
from .. import telem
from ..transport import Transport


@dataclass
class _RetrieveRequest:
    keys: Optional[list[str]] = None
    node_id: int = None
    names: Optional[list[str]] = None


@dataclass
class _Response:
    channels: list[Channel]

    def load(self, data: dict):
        self.channels = [Channel(**c) for c in data["channels"]]


def _response_factory() -> _Response:
    return _Response(channels=[])


@dataclass
class _CreateRequest:
    channel: Channel
    count: int


_RETRIEVE_ENDPOINT = "/channel/retrieve"
_CREATE_ENDPOINT = "/channel/create"

ChannelRetrieveTransport: TypeAlias = freighter.UnaryClient[
    _RetrieveRequest,
    _Response,
]
ChannelCreateTransport: TypeAlias = freighter.UnaryClient[_CreateRequest, _Response]


class Client:
    retrieve_transport: ChannelRetrieveTransport
    create_transport: ChannelCreateTransport

    def __init__(self, transport: Transport):
        self.retrieve_transport = transport.http.get(
            _RetrieveRequest,
            _response_factory,
        )
        self.create_transport = transport.http.post(
            _CreateRequest,
            _response_factory,
        )

    def retrieve(self, keys: list[str]) -> list[Channel]:
        return self._retrieve(_RetrieveRequest(keys=keys))

    def retrieve_by_name(self, names: list[str]) -> list[Channel]:
        return self._retrieve(_RetrieveRequest(names=names))

    def retrieve_by_node_id(self, node_id: int) -> list[Channel]:
        return self._retrieve(_RetrieveRequest(node_id=node_id))

    def create(
            self,
            name: str = "",
            node_id: int = 0,
            rate: telem.UnparsedRate = telem.Rate(0),
            data_type: telem.UnparsedDataType = telem.DATA_TYPE_UNKNOWN,
    ) -> Channel:
        return self.create_n(Channel(name, node_id, rate, data_type), 1)[0]

    def create_n(self, channel: Channel, count: int = 1) -> list[Channel]:
        req = _CreateRequest(channel=channel, count=count)
        res, exc = self.create_transport.send(_CREATE_ENDPOINT, req)
        if exc is not None:
            raise exc
        assert res is not None
        return res.channels

    def _retrieve(self, req: _RetrieveRequest) -> list[Channel]:
        res, exc = self.retrieve_transport.send(_RETRIEVE_ENDPOINT, req)
        if exc is not None:
            raise exc
        assert res is not None
        return res.channels
