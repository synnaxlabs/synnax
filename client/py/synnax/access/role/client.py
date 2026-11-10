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
from synnax.access.role.types_gen import (
    Cursor,
    Key,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Role,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize
from x.params import deprecate_positional


class _CreateRequest(BaseModel):
    roles: list[Role]


_CreateResponse = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[Key]


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
        roles: Role,
    ) -> Role: ...

    @overload
    def create(
        self,
        roles: list[Role],
    ) -> list[Role]: ...

    def create(
        self,
        roles: Role | list[Role],
    ) -> Role | list[Role]:
        is_single = not isinstance(roles, list)
        req = _CreateRequest(roles=normalize(roles))
        res = self._client.send("/access/role/create", req, _CreateResponse)
        return res.roles[0] if is_single else res.roles

    @overload
    def retrieve(self, key: Key) -> Role: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        internal: filter.BoolInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Role: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        internal: filter.BoolInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Role: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        internal: filter.BoolInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Role: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        internal: filter.BoolInput = ...,
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
    ) -> list[Role]: ...

    @deprecate_positional("keys", "search_term", "limit")
    def retrieve(
        self,
        key: filter.SetInput[Key] | None = None,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Role | list[Role]:
        """Retrieves roles. With no arguments, retrieves every role.

        Each filter field takes a value, a list of values, or an operator, and a role
        must match every field. A plain ``key``, by position or by keyword, or a plain
        ``name`` returns one role.

        :param key: Keeps roles with this key.
        :param name: Keeps roles with this name.
        :param internal: Keeps roles with this internal flag.
        :param and_: Keeps roles that match every filter in the list.
        :param or_: Keeps roles that match any filter in the list.
        :param not_: Keeps roles that do not match the filter.
        :param where: A filter tree that the roles must also match.
        :param keys: Keeps roles with one of these keys.
        :param search_term: A fuzzy search term matched against role names.
        :param limit: The maximum number of roles to retrieve.
        :param order: The order of the roles. Defaults to key order.
        :param after: Retrieves the roles past this position in the order: the key of
            the last role of the page before, and its value of the order field.
        :param one: Returns one role for any filter.
        :returns: One role for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of roles. The list leaves out keys that do not exist.
        :raises NotFoundError: If one role is wanted and none matches.
        :raises MultipleFoundError: If one role is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one role is wanted
            beside a limit or a cursor.
        """
        if key is not None:
            fields["key"] = key
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
        )
        res = self._client.send("/access/role/retrieve", req, RetrieveResponse)
        return sole("role", req, res.roles) if single else res.roles

    def delete(self, keys: Key | list[Key]) -> None:
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/access/role/delete", req, Empty)
