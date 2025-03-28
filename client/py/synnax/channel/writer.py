#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import Instrumentation, trace
from freighter import Empty, Payload, UnaryClient, send_required

from synnax.channel.payload import (
    ChannelKeys,
    ChannelNames,
    ChannelParams,
    ChannelPayload,
    normalize_channel_params,
)
from synnax.channel.retrieve import CacheChannelRetriever


class _CreateRequest(Payload):
    channels: list[ChannelPayload]


_Response = _CreateRequest


class _DeleteRequest(Payload):
    keys: ChannelKeys | None = None
    names: ChannelNames | None = None


_CREATE_ENDPOINT = "/channel/create"
_DELETE_ENDPOINT = "/channel/delete"
_RENAME_ENDPOINT = "/channel/rename"


class _RenameRequest(Payload):
    keys: ChannelKeys
    names: ChannelNames


class ChannelWriter:
    _client: UnaryClient
    _cache: CacheChannelRetriever | None
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation,
        cache: CacheChannelRetriever | None,
    ):
        self._client = client
        self.instrumentation = instrumentation
        self._cache = cache

    @trace("debug")
    def create(
        self,
        channels: list[ChannelPayload],
    ) -> list[ChannelPayload]:
        req = _CreateRequest(channels=channels)
        res = send_required(self._client, _CREATE_ENDPOINT, req, _Response)
        if self._cache is not None:
            self._cache.set(res.channels)
        return res.channels

    @trace("debug")
    def delete(self, channels: ChannelParams) -> None:
        normal = normalize_channel_params(channels)
        req = _DeleteRequest(**{normal.variant: normal.channels})
        send_required(self._client, _DELETE_ENDPOINT, req, Empty)
        if self._cache is not None:
            self._cache.delete(normal.channels)

    @trace("debug")
    def rename(self, keys: ChannelKeys, names: ChannelNames) -> None:
        req = _RenameRequest(keys=keys, names=names)
        send_required(self._client, _RENAME_ENDPOINT, req, Empty)
        if self._cache is not None:
            self._cache.rename(keys, names)
