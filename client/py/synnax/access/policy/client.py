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

from pydantic import BaseModel

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.access.policy.payload import Policy
from synnax.access.policy.types_gen import RetrieveRequest, RetrieveResponse, Where
from synnax.ontology.payload import ID
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    policies: list[Policy]


_CreateResponse = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[UUID]


_ONTOLOGY_TYPE = ID(type="policy")


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
        where: Where | None = None,
        keys: list[UUID] | None = None,
        subjects: list[ID] | None = None,
        internal: bool | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> list[Policy]:
        """Retrieves the policies that match every given filter.

        :param where: Keeps policies that match this filter tree.
        :param subjects: Keeps the policies of the roles that any of these subjects
            hold.
        """
        flat = Where(
            key=None if keys is None else filter.Set[UUID](in_=keys),
            subjects=subjects,
            internal=None if internal is None else filter.Bool(eq=internal),
        )
        res = self._client.send(
            "/access/policy/retrieve",
            RetrieveRequest(
                where=filter.merge(where, flat),
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            RetrieveResponse,
        )
        return res.policies

    def delete(self, keys: UUID | list[UUID]) -> None:
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/access/policy/delete", req, Empty)
