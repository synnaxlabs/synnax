from __future__ import annotations
from dataclasses import dataclass
from typing import Optional, TypeAlias

from freighter import (
    UnaryClient,
    Endpoint,
    JSONEncoderDecoder
)
from freighter.http import (
    GETClient,
    POSTClient,
)

from .channel import Channel


@dataclass
class _RetrieveRequest:
    keys: Optional[list[str]] = None
    node_id: Optional[int] = None
    names: Optional[list[str]] = None


@dataclass
class _Response:
    channels: list[Channel]

    def load(self, data: dict):
        self.channels = [Channel.parse(c) for c in data["channels"]]


def _response_factory() -> _Response:
    return _Response(channels=[])


@dataclass
class _CreateRequest:
    channel: Channel
    count: int


_RETRIEVE_ENDPOINT = "/channel/retrieve"
_CREATE_ENDPOINT = "/channel/create"

ChannelRetrieveTransport: TypeAlias = UnaryClient[_RetrieveRequest, _Response,]
ChannelCreateTransport: TypeAlias = UnaryClient[_CreateRequest, _Response]


class Client:
    retrieve_transport: ChannelRetrieveTransport
    create_transport: ChannelCreateTransport

    def __init__(self,
                 retrieve_transport: ChannelRetrieveTransport,
                 create_transport: ChannelCreateTransport,
                 ):
        self.retrieve_transport = retrieve_transport
        self.create_transport = create_transport

    def retrieve(self, keys: list[str]) -> list[Channel]:
        return self._retrieve(_RetrieveRequest(keys=keys))

    def retrieve_by_name(self, names: list[str]) -> list[Channel]:
        return self._retrieve(_RetrieveRequest(names=names))

    def retrieve_by_node_id(self, node_id: int) -> list[Channel]:
        return self._retrieve(_RetrieveRequest(node_id=node_id))

    def create(self, channel: Channel, count: int = 1) -> list[Channel]:
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


def new_http_client(endpoint: Endpoint) -> Client:
    return Client(
        retrieve_transport=GETClient(
            endpoint=endpoint,
            response_factory=_response_factory,
            encoder_decoder=JSONEncoderDecoder(),
        ),
        create_transport=POSTClient(
            endpoint=endpoint,
            response_factory=_response_factory,
            encoder_decoder=JSONEncoderDecoder(),
        )
    )
