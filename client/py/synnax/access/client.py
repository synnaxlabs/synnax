#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import uuid
from typing import overload

from alamos import Instrumentation, NOOP, trace
from freighter import Payload, UnaryClient
from synnax.access.payload import Policy
from synnax.exceptions import NotFoundError
from synnax.ontology.id import OntologyID


class _RetrieveRequest(Payload):
    subject: OntologyID


class _RetrieveResponse(Payload):
    policies: list[Policy] | None


class _CreateRequest(Payload):
    policies: list[Policy]


_CreateResponse = _CreateRequest


class _DeleteRequest(Payload):
    keys: list[uuid.UUID]


class _DeleteResponse(Payload):
    ...


policy_ontology_type = OntologyID(type="policy")


class PolicyClient:
    __CREATE_ENDPOINT = "/access/policy/create"
    __RETRIEVE_ENDPOINT = "/access/policy/retrieve"
    __DELETE_ENDPOINT = "/access/policy/delete"
    __client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ):
        self.__client = client
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        *,
        subjects: list[OntologyID] = None,
        objects: list[OntologyID] = None,
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
        subjects: list[OntologyID] = None,
        objects: list[OntologyID] = None,
        actions: list[str] = None,
    ) -> Policy | list[Policy]:
        if policies is None:
            _policies = [
                Policy(
                    subjects=subjects,
                    objects=objects,
                    actions=actions,
                )
            ]
        elif isinstance(policies, Policy):
            _policies = [policies]
        else:
            _policies = policies

        req = _CreateRequest(policies=_policies)
        res, exc = self.__client.send(self.__CREATE_ENDPOINT, req, _CreateResponse)
        if exc is not None:
            raise exc

        return res.policies[0] if len(res.policies) == 1 else res.policies

    @trace("debug")
    def retrieve(self, subject: OntologyID) -> list[Policy]:
        res, exc = self.__client.send(
            self.__RETRIEVE_ENDPOINT,
            _RetrieveRequest(subject=subject),
            _RetrieveResponse,
        )
        if exc is not None:
            raise exc
        return res.policies

    @trace("debug")
    def delete(self, keys: uuid.UUID | list[uuid.UUID]) -> None:
        res, exc = self.__client.send(
            self.__DELETE_ENDPOINT,
            _DeleteRequest(keys=[keys] if isinstance(keys, uuid.UUID) else keys),
            _DeleteResponse,
        )
        if exc is not None:
            raise exc
        return res
