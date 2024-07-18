#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
import uuid

from alamos import Instrumentation, trace, NOOP
from freighter import Payload, UnaryClient
from synnax.access.payload import Policy


class _CreateRequest(Payload):
    policies: list[Policy]


_CreateResponse = _CreateRequest


class _DeleteRequest(Payload):
    keys: list[uuid.UUID]


class _DeleteResponse(Payload):
    ...


_POLICY_CREATE_ENDPOINT = "/access/policy/create"
_POLICY_DELETE_ENDPOINT = "/access/policy/delete"


class PolicyWriter:
    __client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ):
        self.__client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def create(
        self,
        policies: list[Policy],
    ) -> list[Policy]:
        req = _CreateRequest(policies=policies)
        res, exc = self.__client.send(_POLICY_CREATE_ENDPOINT, req, _CreateResponse)
        if exc is not None:
            raise exc
        return res.policies

    @trace("debug")
    def delete(self, keys: list[uuid.UUID]) -> None:
        req = _DeleteRequest(keys=keys)
        res, exc = self.__client.send(_POLICY_DELETE_ENDPOINT, req, _DeleteResponse)
        if exc is not None:
            raise exc
        return res
