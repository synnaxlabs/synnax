#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload
from uuid import UUID

from pydantic import BaseModel, Field

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.access.policy.payload import Policy
from synnax.resource import ID
from x.lists import normalize


class _CreateRequest(BaseModel):
    policies: list[Policy]


_CreateResponse = _CreateRequest


class _RetrieveRequest(BaseModel):
    keys: list[UUID] | None
    subjects: list[ID] | None
    roles: list[UUID] | None = None
    internal: bool | None = None


class _RetrieveResponse(BaseModel):
    policies: list[Policy] = Field(default_factory=list)


class _DeleteRequest(BaseModel):
    keys: list[UUID]


_RESOURCE_TYPE = ID(type="policy")


class Client:
    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        *,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ):
        self._client = client
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        policies: Policy,
    ) -> Policy: ...

    @overload
    def create(
        self,
        policies: list[Policy],
    ) -> list[Policy]: ...

    def create(
        self,
        policies: Policy | list[Policy],
    ) -> Policy | list[Policy]:
        is_single = not isinstance(policies, list)
        req = _CreateRequest(policies=normalize(policies))
        res = self._client.send("/access/policy/create", req, _CreateResponse)
        return res.policies[0] if is_single else res.policies

    def retrieve(
        self,
        *,
        keys: list[UUID] | None = None,
        subjects: list[ID] | None = None,
        roles: list[UUID] | None = None,
        internal: bool | None = None,
    ) -> list[Policy]:
        """Retrieves the policies that match every given filter.

        :param keys: Keeps only the policies with these keys.
        :param subjects: Keeps only the policies of the roles these subjects hold.
        :param roles: Keeps only the policies of these roles.
        :param internal: Keeps only built-in policies when true, and others when false.
        """
        res = self._client.send(
            "/access/policy/retrieve",
            _RetrieveRequest(
                keys=keys, subjects=subjects, roles=roles, internal=internal
            ),
            _RetrieveResponse,
        )
        return res.policies

    def delete(self, keys: UUID | list[UUID]) -> None:
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/access/policy/delete", req, Empty)
