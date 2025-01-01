#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

from alamos import NOOP, Instrumentation, trace
from freighter import Payload, UnaryClient

from synnax.ranger.payload import RangePayload, RangeKey, RangeName
from synnax.util.normalize import normalize


class _Request(Payload):
    keys: list[uuid.UUID | str] | None = None
    names: list[str] | None = None
    term: str | None = None


class _Response(Payload):
    ranges: list[RangePayload] | None


class RangeRetriever:
    __ENDPOINT = "/range/retrieve"
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
        key: RangeKey | None = None,
        name: RangeName | None = None,
        names: list[RangeName] | None = None,
        keys: list[RangeKey] | None = None,
    ) -> list[RangePayload]:
        if key is not None:
            keys = normalize(key)
        if name is not None:
            names = normalize(name)
        return self.__execute(_Request(keys=keys, names=names))

    @trace("debug")
    def search(self, term: str) -> list[RangePayload]:
        return self.__execute(_Request(term=term))

    def __execute(self, req: _Request) -> list[RangePayload]:
        res, exc = self.__client.send(self.__ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        if res.ranges is None:
            return list()
        return res.ranges
