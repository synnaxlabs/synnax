#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import Instrumentation, trace
from freighter import Empty, UnaryClient, send_required
from pydantic import BaseModel

from synnax.channel.payload import (
    Key,
    Params,
    Payload,
    normalize_params,
)
from synnax.channel.retrieve import CacheRetriever


class _CreateRequest(BaseModel):
    channels: list[Payload]


_Response = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[Key] | tuple[Key] | None = None
    names: list[str] | tuple[str] | None = None


class _RenameRequest(BaseModel):
    keys: list[Key] | tuple[Key]
    names: list[str] | tuple[str]


class Writer:
    _client: UnaryClient
    _cache: CacheRetriever | None
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation,
        cache: CacheRetriever | None,
    ):
        self._client = client
        self.instrumentation = instrumentation
        self._cache = cache

    @trace("debug")
    def create(
        self,
        channels: list[Payload],
    ) -> list[Payload]:
        req = _CreateRequest(channels=channels)
        res = send_required(self._client, "/channel/create", req, _Response)
        if self._cache is not None:
            self._cache.set(res.channels)
        return res.channels

    @trace("debug")
    def delete(self, channels: Params) -> None:
        normal = normalize_params(channels)
        req = _DeleteRequest(**{normal.variant: normal.channels})
        send_required(self._client, "/channel/delete", req, Empty)
        if self._cache is not None:
            self._cache.delete(normal.channels)

    @trace("debug")
    def rename(
        self, keys: list[Key] | tuple[Key], names: list[str] | tuple[str]
    ) -> None:
        req = _RenameRequest(keys=keys, names=names)
        send_required(self._client, "/channel/rename", req, Empty)
        if self._cache is not None:
            self._cache.rename(keys, names)
