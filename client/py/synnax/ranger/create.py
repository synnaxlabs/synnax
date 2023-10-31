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

from synnax.ranger.payload import RangePayload


class _Request(Payload):
    ranges: list[RangePayload]


_Response = _Request


class RangeCreator:
    __ENDPOINT = "/range/create"
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
        req = _Request(ranges=ranges)
        res, exc = self.__client.send(self.__ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        return res.ranges
