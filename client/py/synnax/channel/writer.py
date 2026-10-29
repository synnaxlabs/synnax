#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel

from alamos import Instrumentation, trace
from freighter import Empty, UnaryClient
from synnax.channel.payload import (
    Key,
    NormalizedNameResult,
    Params,
    Payload,
    normalize_params,
)


class _CreateRequest(BaseModel):
    channels: list[Payload]


class _Response(BaseModel):
    channels: list[Payload]


class _DeleteRequest(BaseModel):
    keys: list[Key] | tuple[Key] | None = None
    names: list[str] | tuple[str] | None = None


class _RenameRequest(BaseModel):
    keys: list[Key] | tuple[Key]
    names: list[str] | tuple[str]


class Writer:
    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        *,
        client: UnaryClient,
        instrumentation: Instrumentation,
    ):
        self._client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def create(
        self,
        channels: list[Payload],
    ) -> list[Payload]:
        req = _CreateRequest(channels=channels)
        return self._client.send("/channel/create", req, _Response).channels

    @trace("debug")
    def delete(self, channels: Params) -> None:
        normal = normalize_params(channels)
        if isinstance(normal, NormalizedNameResult):
            req = _DeleteRequest(names=normal.channels)
        else:
            req = _DeleteRequest(keys=normal.channels)
        self._client.send("/channel/delete", req, Empty)

    @trace("debug")
    def rename(
        self, keys: list[Key] | tuple[Key], names: list[str] | tuple[str]
    ) -> None:
        req = _RenameRequest(keys=keys, names=names)
        self._client.send("/channel/rename", req, Empty)
