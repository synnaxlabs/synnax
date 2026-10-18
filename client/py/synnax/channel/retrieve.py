#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from pydantic import BaseModel, Field

from alamos import NOOP, Instrumentation, trace
from freighter import UnaryClient
from synnax.channel.payload import (
    Key,
    NormalizedNameResult,
    Params,
    Payload,
    normalize_params,
)
from synnax.exceptions import NotFoundError


class _Request(BaseModel):
    names: list[str] | None = None
    keys: list[int] | None = None


class _Response(BaseModel):
    channels: list[Payload] = Field(default_factory=list)
    not_found: list[str] | None = Field(default_factory=list)


class Retriever:
    """Retrieves channel payloads from the Core."""

    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def retrieve(self, channels: Params) -> list[Payload]:
        normal = normalize_params(channels)
        if len(normal.channels) == 0:
            return list()
        if isinstance(normal, NormalizedNameResult):
            req = _Request(names=normal.channels)
        else:
            req = _Request(keys=normal.channels)
        return self._exec_retrieve(req)

    @trace("debug")
    def retrieve_one(self, param: Key | str) -> Payload:
        req = _Request()
        if isinstance(param, Key):
            req.keys = [param]
        else:
            req.names = [param]
        channels = self._exec_retrieve(req)
        if len(channels) == 0:
            raise NotFoundError(f"Could not find channel matching {param}")
        return channels[0]

    def _exec_retrieve(self, req: _Request) -> list[Payload]:
        return self._client.send("/channel/retrieve", req, _Response).channels


def retrieve_required(r: Retriever, channels: Params) -> list[Payload]:
    normal = normalize_params(channels)
    results = r.retrieve(channels)
    found: set[Key | str]
    if isinstance(normal, NormalizedNameResult):
        found = {c.name for c in results}
    else:
        found = {c.key for c in results}
    not_found: list[Key | str] = [p for p in normal.channels if p not in found]
    if len(not_found) > 0:
        raise NotFoundError(f"Could not find channels: {not_found}")
    return results
