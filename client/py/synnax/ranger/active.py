#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
from alamos import NOOP, Instrumentation
from freighter import Payload, UnaryClient

from synnax.exceptions import QueryError
from synnax.ranger.payload import RangeKey, RangePayload


class _SetActiveRequest(Payload):
    range: RangeKey


class _SetActiveResponse(Payload):
    pass


class _ClearActiveRequest(Payload):
    pass


class _RetrieveActiveRequest(Payload):
    pass


class _RetrieveActiveResponse(Payload):
    range: RangePayload | None = None


_SET_ENDPOINT = "/range/set-active"
_CLEAR_ENDPOINT = "/range/clear-active"
_RETRIEVE_ENDPOINT = "/range/retrieve-active"


class Active:
    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self.__client = client
        self.instrumentation = instrumentation

    def set(self, key: RangeKey) -> None:
        self.__client.send(
            _SET_ENDPOINT, _SetActiveRequest(range=key), _SetActiveResponse
        )

    def clear(self) -> None:
        _, exc = self.__client.send(
            _CLEAR_ENDPOINT, _ClearActiveRequest(), _SetActiveResponse
        )
        if exc is not None:
            raise exc

    def retrieve(self) -> RangePayload | None:
        res, exc = self.__client.send(
            _RETRIEVE_ENDPOINT, _RetrieveActiveRequest(), _RetrieveActiveResponse
        )
        if isinstance(exc, QueryError):
            return None
        if exc is not None:
            raise exc
        return res.range
