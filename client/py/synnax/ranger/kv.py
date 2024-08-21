#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

from typing import overload
from freighter import Payload, UnaryClient, send_required

from synnax.util.normalize import normalize


class KVPair(Payload):
    range: uuid.UUID
    key: str
    value: str


class _GetRequest(Payload):
    range: uuid.UUID
    keys: list[str]


class _GetResponse(Payload):
    pairs: list[KVPair]


class _SetRequest(Payload):
    range: uuid.UUID
    pairs: list[KVPair]


class _DeleteRequest(Payload):
    range: uuid.UUID
    keys: list[str]


class _EmptyResponse(Payload):
    ...


_SET_ENDPOINT = "/range/kv/set"
_GET_ENDPOINT = "/range/kv/get"
_DELETE_ENDPOINT = "/range/kv/delete"


class KV:
    _client: UnaryClient
    _rng_key: uuid.UUID

    def __init__(self, rng: uuid.UUID, client: UnaryClient) -> None:
        self._client = client
        self._rng_key = rng

    @overload
    def get(self, keys: str) -> str:
        ...

    def get(self, keys: str | list[str]) -> dict[str, str] | str:
        req = _GetRequest(range=self._rng_key, keys=normalize(keys))
        res = send_required(self._client, _GET_ENDPOINT, req, _GetResponse)
        if isinstance(keys, str):
            return res.pairs[0].value
        return {pair.key: pair.value for pair in res.pairs}

    @overload
    def set(self, key: str, value: str):
        ...

    @overload
    def set(self, key: dict[str, str]):
        ...

    def set(self, key: str | dict[str, str], value: str | None = None) -> None:
        pairs = list()
        if isinstance(key, str):
            pairs.append(KVPair(range=self._rng_key, key=key, value=value))
        else:
            for k, v in key.items():
                pairs.append(KVPair(range=self._rng_key, key=k, value=v))
        req = _SetRequest(range=self._rng_key, pairs=pairs)
        send_required(self._client, _SET_ENDPOINT, req, _EmptyResponse)

    def delete(self, keys: str | list[str]) -> None:
        req = _DeleteRequest(range=self._rng_key, keys=normalize(keys))
        send_required(self._client, _DELETE_ENDPOINT, req, _EmptyResponse)

    def __getitem__(self, key: str) -> str:
        return self.get(key)

    def __setitem__(self, key: str, value: str) -> None:
        self.set(key, value)

    def __delitem__(self, key: str) -> None:
        self.delete(key)
