from dataclasses import dataclass
from typing import Optional

import freighter

from delta import telem
from delta.channel import Channel


@dataclass
class ChannelRetrieveRequest:
    keys: Optional[list[str]] = None
    node: Optional[int] = None
    names: Optional[list[str]] = None


@dataclass
class ChannelResponse:
    channels: list[Channel]

    def decode(self, data: dict):
        return ChannelResponse(
            channels=[Channel(**c) for c in data["channels"]]
        )


@dataclass
class ChannelCreateRequest:
    channel: Channel
    count: int


ChannelRetrieveTransport = freighter.UnaryClient[ChannelRetrieveRequest, ChannelResponse]
ChannelCreateTransport = freighter.UnaryClient[ChannelCreateRequest, ChannelResponse]


class ChannelClient:
    retrieve_transport: ChannelRetrieveTransport
    create_transport: ChannelCreateTransport

    def __init__(self,
                 retrieve_transport: ChannelRetrieveTransport,
                 create_transport: ChannelCreateTransport,
                 ):
        self.retrieve_transport = retrieve_transport
        self.create_transport = create_transport

    def retrieve(self, keys: list[str]) -> list[Channel]:
        request = ChannelRetrieveRequest(keys=keys)
        res, exc = self.retrieve_transport.send("/channel/retrieve", request)
        if exc is not None:
            raise exc
        return res.channels

    async def retrieve_by_name(self, names: list[str]) -> list[Channel]:
        request = ChannelRetrieveRequest(names=names)
        res, exc = await self.retrieve_transport.send("/channel/retrieve", request)
        if exc is not None:
            raise exc
        return res.channels

    async def retrieve_by_node(self, node: int) -> list[Channel]:
        res, exc = await self.retrieve_transport.send("/channel/retrieve", request)
        if exc is not None:
            raise exc
        return res.channels

    async def create(self, channel: Channel, count: int = 1):
        req = ChannelCreateRequest(channel=channel, count=count)
        res, exc = await self.create_transport.send("/channel/create", req)
        if exc is not None:
            raise exc
        return res.channels
