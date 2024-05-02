#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import NOOP, Instrumentation, trace
from freighter import Payload, UnaryClient

from synnax.ranger.payload import RangeKey, RangePayload


class _CreateRequest(Payload):
    ranges: list[RangePayload]


_CreateResponse = _CreateRequest


class _DeleteRequest(Payload):
    keys: list[RangeKey]


class _DeleteResponse(Payload):
    ...


_CREATE_ENDPOINT = "/range/create"
_DELETE_ENDPOINT = "/range/delete"


class RangeWriter:
    __client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self.__client = client
        self.instrumentation = instrumentation

    @trace("debug", "range.create")
    def create(self, ranges: list[RangePayload]) -> list[RangePayload]:
        req = _CreateRequest(ranges=ranges)
        res, exc = self.__client.send(_CREATE_ENDPOINT, req, _CreateResponse)
        if exc is not None:
            raise exc
        return res.ranges

    @trace("debug", "range.delete")
    def delete(self, keys: list[RangeKey]):
        req = _DeleteRequest(keys=keys)
        res, exc = self.__client.send(_DELETE_ENDPOINT, req, _DeleteResponse)
        if exc is not None:
            raise exc
