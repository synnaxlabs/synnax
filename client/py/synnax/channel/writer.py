#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import Instrumentation, trace
from freighter import Payload, UnaryClient

from synnax.channel.payload import (
    ChannelKeys,
    ChannelNames,
    ChannelPayload,
    ChannelParams,
    normalize_channel_params,
)
from synnax.channel.retrieve import CacheChannelRetriever


class _CreateRequest(Payload):
    channels: list[ChannelPayload]


_Response = _CreateRequest


class _DeleteRequest(Payload):
    keys: ChannelKeys | None
    names: ChannelNames | None


class _DeleteResponse(Payload):
    ...


_CHANNEL_CREATE_ENDPOINT = "/channel/create"
_CHANNEL_DELETE_ENDPOINT = "/channel/delete"
_CHANNEL_RENAME_ENDPOINT = "/channel/rename"


class _RenameRequest(Payload):
    keys: ChannelKeys
    names: ChannelNames


class _RenameResponse(Payload):
    ...


class ChannelWriter:
    __ENDPOINT = "/channel/create"
    __client: UnaryClient
    __cache: CacheChannelRetriever
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation,
        cache: CacheChannelRetriever,
    ):
        self.__client = client
        self.instrumentation = instrumentation
        self.__cache = cache

    @trace("debug")
    def create(
        self,
        channels: list[ChannelPayload],
    ) -> list[ChannelPayload]:
        req = _CreateRequest(channels=channels)
        res, exc = self.__client.send(self.__ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        self.__cache.set(res.channels)
        return res.channels

    @trace("debug")
    def delete(self, channels: ChannelParams) -> None:
        normal = normalize_channel_params(channels)
        req = _DeleteRequest(**{normal.variant: normal.params})
        res, exc = self.__client.send(_CHANNEL_DELETE_ENDPOINT, req, _DeleteResponse)
        if exc is not None:
            raise exc
        self.__cache.delete(normal.params)
        return res

    @trace("debug")
    def rename(self, keys: ChannelKeys, names: ChannelNames) -> None:
        req = _RenameRequest(keys=keys, names=names)
        res, exc = self.__client.send(_CHANNEL_RENAME_ENDPOINT, req, _RenameResponse)
        if exc is not None:
            raise exc
        self.__cache.rename(keys, names)
        return res
