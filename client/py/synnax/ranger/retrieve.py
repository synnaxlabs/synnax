#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

from alamos import NOOP, Instrumentation, trace
from freighter import UnaryClient
from pydantic import BaseModel

from synnax.ranger.payload import Key, Payload
from synnax.util.normalize import normalize


class _Request(BaseModel):
    keys: list[uuid.UUID | str] | None = None
    names: list[str] | None = None
    search_term: str | None = None


class _Response(BaseModel):
    ranges: list[Payload] | None


class Retriever:
    __client: UnaryClient
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self.__client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def retrieve(
        self,
        key: Key | None = None,
        name: str | None = None,
        names: list[str] | tuple[str] | None = None,
        keys: list[Key] | tuple[Key] | None = None,
    ) -> list[Payload]:
        if key is not None:
            keys = normalize(key)
        if name is not None:
            names = normalize(name)
        return self.__execute(_Request(keys=keys, names=names))

    @trace("debug")
    def search(self, term: str) -> list[Payload]:
        return self.__execute(_Request(search_term=term))

    def __execute(self, req: _Request) -> list[Payload]:
        res, exc = self.__client.send("/range/retrieve", req, _Response)
        if exc is not None:
            raise exc
        if res is None or res.ranges is None:
            return list()
        return res.ranges
