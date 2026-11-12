#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from pydantic import BaseModel

import synnax.channel.payload as channel
import synnax.ranger.types_gen as ranger
from freighter import UnaryClient
from synnax.exceptions import NotFoundError
from x.lists import normalize


class _ResolveRequest(BaseModel):
    range: ranger.Key
    aliases: list[str]


class _ResolveResponse(BaseModel):
    aliases: dict[str, channel.Key]


class _SetRequest(BaseModel):
    range: ranger.Key
    aliases: dict[channel.Key, str]


class _EmptyResponse(BaseModel): ...


class Client:
    _client: UnaryClient

    def __init__(self, *, rng: ranger.Key, client: UnaryClient) -> None:
        self._client = client
        self._rng = rng

    @overload
    def resolve(self, aliases: str) -> channel.Key: ...

    @overload
    def resolve(self, aliases: list[str]) -> dict[str, channel.Key]: ...

    def resolve(self, aliases: str | list[str]) -> dict[str, channel.Key] | channel.Key:
        req = _ResolveRequest(range=self._rng, aliases=normalize(aliases))
        res = self._client.send("/range/alias/resolve", req, _ResolveResponse)
        if not isinstance(aliases, str):
            return res.aliases
        if aliases not in res.aliases:
            raise NotFoundError(f"Alias matching {aliases} not found")
        return res.aliases[aliases]

    def set(self, aliases: dict[channel.Key, str]) -> None:
        req = _SetRequest(range=self._rng, aliases=aliases)
        self._client.send("/range/alias/set", req, _EmptyResponse)
