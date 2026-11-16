#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload

from pydantic import BaseModel

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.access import role
from synnax.access.policy.payload import Policy
from synnax.access.policy.types_gen import (
    Cursor,
    Key,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.resource import ID
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize
from x.params import deprecate_positional


class _CreateRequest(BaseModel):
    policies: list[Policy]


_CreateResponse = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[Key]


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

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        internal: filter.BoolInput = ...,
        subjects: ID | list[ID] = ...,
        roles: role.Key | list[role.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Policy: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        internal: filter.BoolInput = ...,
        subjects: ID | list[ID] = ...,
        roles: role.Key | list[role.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Policy: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        internal: filter.BoolInput = ...,
        subjects: ID | list[ID] = ...,
        roles: role.Key | list[role.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[Policy]: ...

    @deprecate_positional("keys", "subjects", "internal")
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Policy | list[Policy]:
        """Retrieves policies. With no arguments, retrieves every policy.

        Each filter field takes a value, a list of values, or an operator, and a policy
        must match every field. A plain ``key`` returns one policy.

        :param key: Keeps policies with this key.
        :param internal: Keeps policies with this internal flag.
        :param subjects: Keeps the policies of the roles that any of these subjects
            hold.
        :param roles: Keeps the policies of any of these roles.
        :param and_: Keeps policies that match every filter in the list.
        :param or_: Keeps policies that match any filter in the list.
        :param not_: Keeps policies that do not match the filter.
        :param where: A filter tree that the policies must also match.
        :param keys: Keeps policies with one of these keys.
        :param search_term: A fuzzy search term matched against policy names.
        :param limit: The maximum number of policies to retrieve.
        :param order: The order of the policies. Defaults to key order.
        :param after: Retrieves the policies past this position in the order: the key of
            the last policy of the page before, and its value of the order field.
        :param one: Returns one policy for any filter.
        :returns: One policy for a plain ``key`` or when ``one`` is set, otherwise a
            list of policies. The list leaves out keys that do not exist.
        :raises NotFoundError: If one policy is wanted and none matches.
        :raises MultipleFoundError: If one policy is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one policy is wanted
            beside a limit or a cursor.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
        )
        res = self._client.send("/access/policy/retrieve", req, RetrieveResponse)
        return sole("policy", req, res.policies) if single else res.policies

    def delete(self, keys: Key | list[Key]) -> None:
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/access/policy/delete", req, Empty)
