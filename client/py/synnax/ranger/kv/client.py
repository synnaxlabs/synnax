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

from freighter import UnaryClient, send_required

from synnax.ranger.kv.payload import (
    DeleteRequest,
    EmptyResponse,
    GetRequest,
    GetResponse,
    KVPair,
    SetRequest,
)
from synnax.util.normalize import normalize


class KV:
    _client: UnaryClient
    _rng_key: uuid.UUID

    def __init__(self, rng: uuid.UUID, client: UnaryClient) -> None:
        self._client = client
        self._rng_key = rng

    @overload
    def get(self, keys: str) -> str: ...

    def get(self, keys: str | list[str]) -> dict[str, str] | str:
        req = GetRequest(range=self._rng_key, keys=normalize(keys))
        res = send_required(self._client, "/range/kv/get", req, GetResponse)
        if isinstance(keys, str):
            return res.pairs[0].value
        return {pair.key: pair.value for pair in res.pairs}

    @overload
    def set(self, key: str, value: any): ...

    @overload
    def set(self, key: dict[str, any]): ...

    def set(self, key: str | dict[str, any], value: any = None) -> None:
        pairs = list()
        if isinstance(key, str):
            pairs.append(KVPair(range=self._rng_key, key=key, value=value))
        else:
            for k, v in key.items():
                pairs.append(KVPair(range=self._rng_key, key=k, value=v))
        req = SetRequest(range=self._rng_key, pairs=pairs)
        send_required(self._client, "/range/kv/set", req, EmptyResponse)

    def delete(self, keys: str | list[str]) -> None:
        req = DeleteRequest(range=self._rng_key, keys=normalize(keys))
        send_required(self._client, "/range/kv/delete", req, EmptyResponse)

    def __getitem__(self, key: str) -> str:
        return self.get(key)

    def __setitem__(self, key: str, value: str) -> None:
        self.set(key, value)

    def __delitem__(self, key: str) -> None:
        self.delete(key)
