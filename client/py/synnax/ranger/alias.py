#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

from freighter import Payload, UnaryClient

from synnax.channel import ChannelKey


class _ResolveRequest(Payload):
    range: uuid.UUID
    aliases: list[str]


class _ResolveResponse(Payload):
    aliases: dict[str, ChannelKey]


class _SetRequest(Payload):
    range: uuid.UUID
    aliases: dict[ChannelKey, str]


class _EmptyResponse(Payload):
    ...


class Aliaser:
    __SET_ENDPOINT = "/range/alias/set"
    __RESOLVE_ENDPOINT = "/range/alias/resolve"
    __client: UnaryClient
    __cache: dict[str, ChannelKey]

    def __init__(self, rng: uuid.UUID, client: UnaryClient) -> None:
        self.__client = client
        self.__rng = rng
        self.__cache = {}

    def resolve(self, alias: str) -> ChannelKey:
        ...

    def resolve(self, aliases: list[str]) -> dict[str, ChannelKey]:
        ...

    def resolve(self, aliases: str | list[str]) -> dict[str, ChannelKey] | ChannelKey:
        to_fetch = []
        is_single = isinstance(aliases, str)
        if is_single:
            cached = self.__cache.get(aliases, None)
            if cached is not None:
                return cached
            to_fetch = [aliases]
        else:
            cached = {}
            for alias in aliases:
                if alias in self.__cache:
                    cached[alias] = self.__cache[alias]
                else:
                    to_fetch.append(alias)
            if len(to_fetch) == 0:
                return cached
        req = _ResolveRequest(range=self.__rng, aliases=to_fetch)
        res, exc = self.__client.send(self.__RESOLVE_ENDPOINT, req, _ResolveResponse)
        if exc is not None:
            raise exc

        for alias, key in res.aliases.items():
            self.__cache[alias] = key

        if is_single:
            return res.aliases[aliases]
        return {**cached, **res.aliases}

    def set(self, aliases: dict[ChannelKey, str]) -> None:
        req = _SetRequest(range=self.__rng, aliases=aliases)
        res, exc = self.__client.send(self.__SET_ENDPOINT, req, _EmptyResponse)
        if exc is not None:
            raise exc
