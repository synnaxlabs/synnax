from dataclasses import dataclass
from typing import Optional

import freighter

from delta.channel import Channel


@dataclass
class ChannelRetrieveRequest:
    keys: Optional[list[str]] = None
    node: Optional[int] = None
    names: Optional[list[str]] = None


@dataclass
class ChannelResponse:
    channels: list[Channel]


@dataclass
class ChannelCreateRequest:
    channel: Channel
    count: int


class ChannelClient:
    retrieve_transport: freighter.UnaryClient[ChannelRetrieveRequest, ChannelResponse]
    create_transport: freighter.UnaryClient[ChannelCreateRequest, ChannelResponse]

    def __init__(self, retrieve_transport: freighter.UnaryClient):
        self.retrieve_transport = retrieve_transport

    async def retrieve(self, keys: list[str]) -> list[Channel]:
        request = ChannelRetrieveRequest(keys=keys)
        response = ChannelResponse(channels=[])
        exc = await self.retrieve_transport.send("/channel/retrieve", request, response)
        if exc is not None:
            raise exc
        return response.channels

    async def retrieve_by_name(self, names: list[str]) -> list[Channel]:
        request = ChannelRetrieveRequest(names=names)
        response = ChannelResponse(channels=[])
        exc = await self.retrieve_transport.send("/channel/retrieve", request, response)
        if exc is not None:
            raise exc
        return response.channels

    async def retrieve_by_node(self, node: int) -> list[Channel]:
        request = ChannelRetrieveRequest(node=node)
        response = ChannelResponse(channels=[])
        exc = await self.retrieve_transport.send("/channel/retrieve", request, response)
        if exc is not None:
            raise exc
        return response.channels

    async def create(self, channel: Channel, count: int = 1):
        request = ChannelCreateRequest(channel=channel, count=count)
        response = ChannelResponse(channels=[])
        exc = await self.create_transport.send("/channel/create", request, response)
        if exc is not None:
            raise exc
        return response.channels
