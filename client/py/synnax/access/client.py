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

from alamos import Instrumentation, NOOP, trace
from freighter import Payload, UnaryClient, send_required, Empty
from synnax.access.payload import Policy
from synnax.ontology.payload import ID
from synnax.util.normalize import normalize


class _RetrieveRequest(Payload):
    subject: ID


class _RetrieveResponse(Payload):
    policies: list[Policy] | None


class _CreateRequest(Payload):
    policies: list[Policy]


_CreateResponse = _CreateRequest


class _DeleteRequest(Payload):
    keys: list[uuid.UUID]


ONTOLOGY_TYPE = ID(type="policy")

_CREATE_ENDPOINT = "/access/policy/create"
_RETRIEVE_ENDPOINT = "/access/policy/retrieve"
_DELETE_ENDPOINT = "/access/policy/delete"


class PolicyClient:
    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ):
        self._client = client
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        *,
        subjects: list[ID] = None,
        objects: list[ID] = None,
        actions: list[str] = None,
    ) -> Policy:
        ...

    @overload
    def create(
        self,
        policies: Policy,
    ) -> Policy:
        ...

    @overload
    def create(
        self,
        policies: list[Policy],
    ) -> list[Policy]:
        ...

    @trace("debug")
    def create(
        self,
        policies: Policy | list[Policy] | None = None,
        *,
        subjects: list[ID] = None,
        objects: list[ID] = None,
        actions: list[str] = None,
    ) -> Policy | list[Policy]:
        is_single = not isinstance(policies, list)
        if policies is None:
            policies = Policy(
                subjects=subjects,
                objects=objects,
                actions=actions,
            )
        req = _CreateRequest(policies=normalize(policies))
        res = send_required(self._client, _CREATE_ENDPOINT, req, _CreateResponse)
        return res.policies[0] if is_single else res.policies

    @trace("debug")
    def retrieve(self, subject: ID) -> list[Policy]:
        return send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(subject=subject),
            _RetrieveResponse,
        ).policies

    @trace("debug")
    def delete(self, keys: uuid.UUID | list[uuid.UUID]) -> None:
        req = _DeleteRequest(keys=normalize(keys))
        send_required(self._client, _DELETE_ENDPOINT, req, Empty)
