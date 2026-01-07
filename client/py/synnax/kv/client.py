#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid
from typing import Any, overload

from freighter import UnaryClient, send_required

from synnax.kv.payload import (
    DeleteRequest,
    EmptyResponse,
    GetRequest,
    GetResponse,
    Pair,
    SetRequest,
)
from synnax.util.normalize import normalize


class KV:
    """Key-value storage operations scoped to a range."""

    _client: UnaryClient
    _rng_key: uuid.UUID

    def __init__(self, rng: uuid.UUID, client: UnaryClient) -> None:
        """Create a new KV instance.

        :param rng: The range key to scope the KV operations to.
        :param client: The unary client for making requests.
        """
        self._client = client
        self._rng_key = rng

    @overload
    def get(self, keys: str) -> str: ...

    def get(self, keys: str | list[str]) -> dict[str, str] | str:
        """Get one or more values by key.

        :param keys: A single key or list of keys to retrieve.
        :returns: The value if a single key was provided, otherwise a dict of keys to values.
        """
        req = GetRequest(range=self._rng_key, keys=normalize(keys))
        res = send_required(self._client, "/range/kv/get", req, GetResponse)
        if isinstance(keys, str):
            return res.pairs[0].value
        return {pair.key: pair.value for pair in res.pairs}

    @overload
    def set(self, key: str, value: Any): ...

    @overload
    def set(self, key: dict[str, Any]): ...

    def set(self, key: str | dict[str, Any], value: Any = None) -> None:
        """Set one or more key-value pairs.

        :param key: A single key or a dict of keys to values.
        :param value: The value to set (only used if key is a string).
        """
        pairs = list()
        if isinstance(key, str):
            pairs.append(Pair(range=self._rng_key, key=key, value=value))
        else:
            for k, v in key.items():
                pairs.append(Pair(range=self._rng_key, key=k, value=v))
        req = SetRequest(range=self._rng_key, pairs=pairs)
        send_required(self._client, "/range/kv/set", req, EmptyResponse)

    def delete(self, keys: str | list[str]) -> None:
        """Delete one or more keys.

        :param keys: A single key or list of keys to delete.
        """
        req = DeleteRequest(range=self._rng_key, keys=normalize(keys))
        send_required(self._client, "/range/kv/delete", req, EmptyResponse)

    def __getitem__(self, key: str) -> str:
        """Get a value by key using bracket notation."""
        return self.get(key)

    def __setitem__(self, key: str, value: str) -> None:
        """Set a value by key using bracket notation."""
        self.set(key, value)

    def __delitem__(self, key: str) -> None:
        """Delete a key using del notation."""
        self.delete(key)


class Client:
    """Client for accessing the KV API."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        """Create a new KV client.

        :param client: The unary client for making requests.
        """
        self._client = client

    def get(self, range_key: uuid.UUID) -> KV:
        """Get a KV instance scoped to a specific range.

        :param range_key: The range key to scope the KV operations to.
        :returns: A KV instance for the specified range.
        """
        return KV(range_key, self._client)
