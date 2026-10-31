#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal, overload
from uuid import UUID

from pydantic import BaseModel

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax import util
from synnax.access.policy import builder_gen
from synnax.access.policy.builder_gen import Condition
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

    @overload
    def retrieve(self, key: UUID, /) -> Policy: ...

    @overload
    def retrieve(self, keys: list[UUID], /) -> list[Policy]: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[True],
        keys: list[UUID] | None = None,
        subjects: list[ID] | None = None,
        internal: bool | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> Policy: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[False] = False,
        keys: list[UUID] | None = None,
        subjects: list[ID] | None = None,
        internal: bool | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> list[Policy]: ...

    def retrieve(
        self,
        *args: UUID | list[UUID] | Condition,
        one: bool = False,
        keys: list[UUID] | None = None,
        subjects: list[ID] | None = None,
        internal: bool | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> Policy | list[Policy]:
        """Retrieves the policies that match every condition and filter. Returns one
        policy for a key or one, and a list otherwise.

        :param args: Conditions from the policy builder, such as
            ``policy.internal.eq(False)``, or one key or list of keys.
        :param one: Returns the one policy that matches, instead of a list.
        :param subjects: Keeps the policies of the roles that any of these subjects
            hold.
        :raises NotFoundError: If no policy matches a key or one.
        :raises MultipleFoundError: If one is True and more than one policy matches.
        :raises TypeError: If args are neither policy conditions nor a shortcut.
        """
        conditions, query = util.shortcut(
            "Policy", args, Condition, UUID, builder_gen.key
        )
        flat = Where(
            key=None if keys is None else filter.Set[UUID](in_=keys),
            subjects=subjects,
            internal=None if internal is None else filter.Bool(eq=internal),
        )
        where = filter.merge(filter.join(Where, conditions), flat)
        res = self._client.send(
            "/access/policy/retrieve",
            RetrieveRequest(
                where=where,
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            RetrieveResponse,
        )
        if query is None and not one:
            return res.policies
        return util.one("Policy", where if query is None else query, res.policies)

    def delete(self, keys: UUID | list[UUID]) -> None:
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/access/policy/delete", req, Empty)
