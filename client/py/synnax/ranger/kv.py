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

from synnax.util.normalize import normalize


class _GetRequest(Payload):
    range: uuid.UUID
    keys: list[str]


class _GetResponse(Payload):
    pairs: dict[str, str]


class _SetRequest(Payload):
    range: uuid.UUID
    pairs: dict[str, str]


class _DeleteRequest(Payload):
    range: uuid.UUID
    keys: list[str]


class _EmptyResponse(Payload):
    ...


class KV:
    __SET_ENDPOINT = "/range/kv/set"
    __GET_ENDPOINT = "/range/kv/get"
    __DELETE_ENDPOINT = "/range/kv/delete"

    __client: UnaryClient
    __rng: uuid.UUID

    def __init__(self, rng: uuid.UUID, client: UnaryClient) -> None:
        self.__client = client
        self.__rng = rng

    def get(self, keys: str) -> str:
        ...

    def get(self, keys: str | list[str]) -> dict[str, str]:
        req = _GetRequest(range=self.__rng, keys=normalize(keys))
        res, exc = self.__client.send(self.__GET_ENDPOINT, req, _GetResponse)
        if exc is not None:
            raise exc
        if isinstance(keys, str):
            return res.pairs[keys]
        return res.pairs

    def set(self, key: str, value: str):
        ...

    def set(self, key: dict[str, str]):
        ...

    def set(self, key: str | dict[str, str], value: str | None = None) -> None:
        if isinstance(key, str):
            if value is None:
                raise ValueError("value is required when providing a single key")
            key = {key: value}
        req = _SetRequest(range=self.__rng, pairs=key)
        res, exc = self.__client.send(self.__SET_ENDPOINT, req, _EmptyResponse)
        if exc is not None:
            raise exc

    def delete(self, keys: str | list[str]) -> None:
        req = _DeleteRequest(range=self.__rng, keys=normalize(keys))
        res, exc = self.__client.send(self.__DELETE_ENDPOINT, req, _EmptyResponse)
        if exc is not None:
            raise exc

    # Implement dict-like interface
    def __getitem__(self, key: str) -> str:
        return self.get(key)

    def __setitem__(self, key: str, value: str) -> None:
        self.set(key, value)

    def __delitem__(self, key: str) -> None:
        self.delete(key)
