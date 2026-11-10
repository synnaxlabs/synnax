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

from freighter import Empty, UnaryClient
from synnax.group import actions_gen
from synnax.group.types_gen import (
    Cursor,
    Group,
    Key,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.resource import ID, CrudeID, Type
from synnax.transport import send_batch
from synnax.util import request_where, returns_one, sole
from x import filter
from x.params import deprecate_positional


class _CreateReq(BaseModel):
    key: Key | None = None
    name: str
    scope: Type | None = None
    parent: ID | None = None


class _CreateRes(BaseModel):
    group: Group


class _DeleteReq(BaseModel):
    keys: list[Key]


class _DispatchReq(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class Client:
    _client: UnaryClient

    def __init__(self, client: UnaryClient):
        self._client = client

    @deprecate_positional("parent", "name", "key")
    def create(
        self,
        name: str,
        *,
        scope: Type | None = None,
        parent: CrudeID | None = None,
        key: Key | None = None,
    ) -> Group:
        """Creates a group.

        :param name: The name of the group.
        :param scope: The type of resource the group holds. Required when the group
            has no parent.
        :param parent: The group, project, or rack to nest the group under. The group
            takes its scope from the parent.
        :param key: The key to give the group. The Core makes one when it is None.
        :returns: The created group.
        """
        return self._client.send(
            "/group/create",
            _CreateReq(
                key=key or None,
                name=name,
                scope=scope,
                parent=ID(parent) if parent is not None else None,
            ),
            _CreateRes,
        ).group

    @overload
    def retrieve(self, key: Key) -> Group: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        parents: ID | list[ID] = ...,
        nested: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Group: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        parents: ID | list[ID] = ...,
        nested: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Group: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        parents: ID | list[ID] = ...,
        nested: bool = ...,
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
    ) -> list[Group]: ...

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
    ) -> Group | list[Group]:
        """Retrieves groups. With no arguments, retrieves every group.

        Each filter field takes a value, a list of values, or an operator, and a group
        must match every field. A plain ``key``, by position or by keyword, returns one
        group.

        :param key: Keeps groups with this key.
        :param parents: Keeps groups that sit in any of these resources.
        :param nested: Keeps only groups that sit in a parent when True, and only top
            groups when False.
        :param and_: Keeps groups that match every filter in the list.
        :param or_: Keeps groups that match any filter in the list.
        :param not_: Keeps groups that do not match the filter.
        :param where: A filter tree that the groups must also match.
        :param keys: Keeps groups with one of these keys.
        :param search_term: A fuzzy search term matched against group names.
        :param limit: The maximum number of groups to retrieve.
        :param order: The order of the groups. Defaults to key order.
        :param after: Retrieves the groups past this position in the order: the key of
            the last group of the page before, and its value of the order field.
        :param one: Returns one group for any filter.
        :returns: One group for a plain ``key`` or when ``one`` is set, otherwise a list
            of groups. The list leaves out keys that do not exist.
        :raises NotFoundError: If one group is wanted and none matches.
        :raises MultipleFoundError: If one group is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one group is wanted
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
        res = self._client.send("/group/retrieve", req, RetrieveResponse)
        return sole("group", req, res.groups) if single else res.groups

    def rename(self, key: Key, name: str) -> Empty:
        action = actions_gen.set(actions_gen.SetPayload(path=["name"], value=name))
        return self._client.send(
            "/group/dispatch", _DispatchReq(key=key, actions=[action]), Empty
        )

    def delete(self, keys: list[Key]) -> Empty:
        return self._client.send("/group/delete", _DeleteReq(keys=keys), Empty)

    def set_parent(self, keys: list[Key], parent: CrudeID | None = None) -> Empty:
        """Moves groups under a parent, or to the top level when the parent is None.

        :param keys: The keys of the groups to move.
        :param parent: The group, project, or rack to move the groups under.
        """
        action = actions_gen.set_parent(
            actions_gen.SetParentPayload(
                parent=ID(parent) if parent is not None else None
            )
        )
        send_batch(
            self._client,
            "/group/dispatch",
            [_DispatchReq(key=k, actions=[action]) for k in keys],
        )
        return Empty()
