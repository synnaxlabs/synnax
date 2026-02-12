#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid
from typing import overload

from freighter import UnaryClient
from pydantic import BaseModel

import synnax.channel.payload as channel
from synnax.util.normalize import normalize


class _ResolveRequest(BaseModel):
    range: uuid.UUID
    aliases: list[str]


class _ResolveResponse(BaseModel):
    aliases: dict[str, channel.Key]


class _SetRequest(BaseModel):
    range: uuid.UUID
    aliases: dict[channel.Key, str]


class _EmptyResponse(BaseModel): ...


class Client:
    __client: UnaryClient
    __cache: dict[str, channel.Key]

    def __init__(self, rng: uuid.UUID, client: UnaryClient) -> None:
        self.__client = client
        self.__rng = rng
        self.__cache = {}

    @overload
    def resolve(self, aliases: str) -> channel.Key: ...

    @overload
    def resolve(self, aliases: list[str]) -> dict[str, channel.Key]: ...

    def resolve(self, aliases: str | list[str]) -> dict[str, channel.Key] | channel.Key:
        is_single = isinstance(aliases, str)
        normalized_aliases = normalize(aliases)
        to_fetch: list[str] = list()

        results: dict[str, channel.Key] = {}
        for alias in normalized_aliases:
            key = self.__cache.get(alias, None)
            if key is not None:
                results[alias] = key
            else:
                to_fetch.append(alias)

        if len(to_fetch) == 0:
            if is_single:
                return results[normalized_aliases[0]]
            return results

        req = _ResolveRequest(range=self.__rng, aliases=to_fetch)
        res, exc = self.__client.send("/range/alias/resolve", req, _ResolveResponse)
        if exc is not None:
            raise exc
        if res is None:
            if is_single:
                raise KeyError(f"Alias not found: {aliases}")
            return results

        for alias, key in res.aliases.items():
            self.__cache[alias] = key

        if is_single:
            return res.aliases[normalized_aliases[0]]
        return {**results, **res.aliases}

    def set(self, aliases: dict[channel.Key, str]) -> None:
        req = _SetRequest(range=self.__rng, aliases=aliases)
        res, exc = self.__client.send("/range/alias/set", req, _EmptyResponse)
        if exc is not None:
            raise exc
