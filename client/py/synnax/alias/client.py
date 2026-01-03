#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid
from typing import overload

from freighter import UnaryClient, send_required

from synnax.alias.payload import (
    DeleteRequest,
    EmptyResponse,
    ListRequest,
    ListResponse,
    ResolveRequest,
    ResolveResponse,
    RetrieveRequest,
    RetrieveResponse,
    SetRequest,
)
from synnax.channel import ChannelKey
from synnax.util.normalize import normalize


class Aliaser:
    """Channel alias operations scoped to a range."""

    _client: UnaryClient
    _rng: uuid.UUID
    _cache: dict[str, ChannelKey]

    def __init__(self, rng: uuid.UUID, client: UnaryClient) -> None:
        """Create a new Aliaser instance.

        :param rng: The range key to scope the alias operations to.
        :param client: The unary client for making requests.
        """
        self._client = client
        self._rng = rng
        self._cache = {}

    @overload
    def resolve(self, alias: str) -> ChannelKey: ...

    @overload
    def resolve(self, aliases: list[str]) -> dict[str, ChannelKey]: ...

    def resolve(self, aliases: str | list[str]) -> dict[str, ChannelKey] | ChannelKey:
        """Resolve one or more aliases to channel keys.

        :param aliases: A single alias or list of aliases to resolve.
        :returns: The channel key if a single alias was provided,
            otherwise a dict of aliases to channel keys.
        """
        to_fetch = list()
        aliases_list = normalize(aliases)
        is_single = isinstance(aliases, str)

        results: dict[str, ChannelKey] = {}
        for alias in aliases_list:
            key = self._cache.get(alias, None)
            if key is not None:
                results[alias] = key
            else:
                to_fetch.append(alias)

        if len(to_fetch) == 0:
            return results

        req = ResolveRequest(range=self._rng, aliases=to_fetch)
        res = send_required(self._client, "/range/alias/resolve", req, ResolveResponse)

        for alias, key in res.aliases.items():
            self._cache[alias] = key

        if is_single:
            return res.aliases[aliases]
        return {**results, **res.aliases}

    def set(self, aliases: dict[ChannelKey, str]) -> None:
        """Set aliases for channels.

        :param aliases: A dict mapping channel keys to their aliases.
        """
        req = SetRequest(range=self._rng, aliases=aliases)
        send_required(self._client, "/range/alias/set", req, EmptyResponse)

    def list_(self) -> dict[ChannelKey, str]:
        """List all aliases for the range.

        :returns: A dict mapping channel keys to aliases.
        """
        req = ListRequest(range=self._rng)
        res = send_required(self._client, "/range/alias/list", req, ListResponse)
        return res.aliases

    @overload
    def retrieve(self, channel: ChannelKey) -> str: ...

    @overload
    def retrieve(self, channels: list[ChannelKey]) -> dict[ChannelKey, str]: ...

    def retrieve(self, channels: ChannelKey) -> str | dict[ChannelKey, str]:
        """Retrieve aliases for one or more channels.

        :param channels: A single channel key or list of channel keys.
        :returns: The alias if a single channel was provided,
            otherwise a dict of channel keys to aliases.
        """
        is_single = isinstance(channels, int)
        req = RetrieveRequest(range=self._rng, channels=normalize(channels))
        res = send_required(
            self._client, "/range/alias/retrieve", req, RetrieveResponse
        )
        if is_single:
            return res.aliases[str(channels)]
        return res.aliases

    def delete(self, channels: ChannelKey | list[ChannelKey]) -> None:
        """Delete aliases for one or more channels.

        :param channels: A single channel key or list of channel keys to delete aliases for.
        """
        req = DeleteRequest(range=self._rng, channels=normalize(channels))
        send_required(self._client, "/range/alias/delete", req, EmptyResponse)


class Client:
    """Client for accessing the Alias API."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        """Create a new Alias client.

        :param client: The unary client for making requests.
        """
        self._client = client

    def get(self, range_key: uuid.UUID) -> Aliaser:
        """Get an Aliaser instance scoped to a specific range.

        :param range_key: The range key to scope the alias operations to.
        :returns: An Aliaser instance for the specified range.
        """
        return Aliaser(range_key, self._client)
