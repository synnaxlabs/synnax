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
from pydantic import BaseModel

from synnax.ranger.kv.payload import Pair
from synnax.util.normalize import normalize


class _GetRequest(BaseModel):
    range: uuid.UUID
    keys: list[str]


class _GetResponse(BaseModel):
    pairs: list[Pair]


class _SetRequest(BaseModel):
    range: uuid.UUID
    pairs: list[Pair]


class _DeleteRequest(BaseModel):
    range: uuid.UUID
    keys: list[str]


class _EmptyResponse(BaseModel): ...


class Client:
    _client: UnaryClient
    _rng_key: uuid.UUID

    def __init__(self, rng: uuid.UUID, client: UnaryClient) -> None:
        self._client = client
        self._rng_key = rng

    @overload
    def get(self, keys: str) -> str: ...

    @overload
    def get(self, keys: list[str]) -> dict[str, str]: ...

    def get(self, keys: str | list[str]) -> dict[str, str] | str:
        req = _GetRequest(range=self._rng_key, keys=normalize(keys))
        res = send_required(self._client, "/range/kv/get", req, _GetResponse)
        if isinstance(keys, str):
            return res.pairs[0].value
        return {pair.key: pair.value for pair in res.pairs}

    @overload
    def set(self, key: str, value: Any): ...

    @overload
    def set(self, key: dict[str, Any]): ...

    def set(self, key: str | dict[str, Any], value: Any = None) -> None:
        pairs = list()
        if isinstance(key, str):
            pairs.append(Pair(range=self._rng_key, key=key, value=value))
        else:
            for k, v in key.items():
                pairs.append(Pair(range=self._rng_key, key=k, value=v))
        req = _SetRequest(range=self._rng_key, pairs=pairs)
        send_required(self._client, "/range/kv/set", req, _EmptyResponse)

    def delete(self, keys: str | list[str]) -> None:
        req = _DeleteRequest(range=self._rng_key, keys=normalize(keys))
        send_required(self._client, "/range/kv/delete", req, _EmptyResponse)

    def __getitem__(self, key: str) -> str:
        return self.get(key)

    def __setitem__(self, key: str, value: Any) -> None:
        self.set(key, value)

    def __delitem__(self, key: str) -> None:
        self.delete(key)
