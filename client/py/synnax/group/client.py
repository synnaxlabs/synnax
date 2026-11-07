#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload
from uuid import UUID

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax.group.types_gen import (
    Group,
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.resource import ID, CrudeID, Type
from synnax.util import request_where, returns_one, sole
from x import filter


class _CreateReq(BaseModel):
    key: UUID | None = None
    name: str
    scope: Type | None = None
    parent: ID | None = None


class _CreateRes(BaseModel):
    group: Group


class _RenameReq(BaseModel):
    key: UUID
    name: str


class _DeleteReq(BaseModel):
    keys: list[UUID]


class _SetParentReq(BaseModel):
    keys: list[UUID]
    parent: ID | None = None


class Client:
    _client: UnaryClient

    def __init__(self, client: UnaryClient):
        self._client = client

    def create(
        self,
        name: str,
        *,
        scope: Type | None = None,
        parent: CrudeID | None = None,
        key: str | None = None,
    ) -> Group:
        """Creates a group, or updates the group with the given key.

        :param name: The name of the group.
        :param scope: The type of resource the group holds. Required when the group
            has no parent.
        :param parent: The group, project, or rack to nest the group under. The group
            takes its scope from the parent.
        :param key: The key of the group to update.
        :returns: The created group.
        """
        return self._client.send(
            "/group/create",
            _CreateReq(
                key=UUID(key) if key else None,
                name=name,
                scope=scope,
                parent=ID(parent) if parent is not None else None,
            ),
            _CreateRes,
        ).group

    @overload
    def retrieve(self, key: UUID) -> Group: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
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
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[UUID] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Group: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[UUID] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        offset: int = ...,
        one: Literal[False] = ...,
    ) -> list[Group]: ...

    def retrieve(
        self,
        key: filter.SetInput[Key] | None = None,
        *,
        where: Where | None = None,
        keys: list[UUID] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        one: bool = False,
        **fields: Any,
    ) -> Group | list[Group]:
        """Retrieves groups. With no arguments, retrieves every group.

        Each filter field takes a value, a list of values, or an operator, and a group
        must match every field. A plain ``key``, by position or by keyword, returns one
        group.

        :param key: Keeps groups with this key.
        :param and_: Keeps groups that match every filter in the list.
        :param or_: Keeps groups that match any filter in the list.
        :param not_: Keeps groups that do not match the filter.
        :param where: A filter tree that the groups must also match.
        :param keys: Keeps groups with one of these keys.
        :param search_term: A fuzzy search term matched against group names.
        :param limit: The maximum number of groups to retrieve.
        :param offset: The number of groups to skip.
        :param one: Returns one group for any filter.
        :returns: One group for a plain ``key`` or when ``one`` is set, otherwise a list
            of groups. The list leaves out keys that do not exist.
        :raises NotFoundError: If one group is wanted and none matches.
        :raises MultipleFoundError: If one group is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one group is wanted
            beside a limit or an offset.
        """
        if key is not None:
            fields["key"] = key
        single = returns_one(fields, one, limit, offset)
        flat = Where(key=None if keys is None else filter.Set[UUID](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            offset=offset,
        )
        res = self._client.send("/group/retrieve", req, RetrieveResponse)
        return sole("Group", fields, res.groups) if single else res.groups

    def rename(self, key: UUID, name: str) -> Empty:
        return self._client.send("/group/rename", _RenameReq(key=key, name=name), Empty)

    def delete(self, keys: list[UUID]) -> Empty:
        return self._client.send("/group/delete", _DeleteReq(keys=keys), Empty)

    def set_parent(self, keys: list[UUID], parent: CrudeID | None = None) -> Empty:
        """Moves groups under a parent, or to the top level when the parent is None.

        :param keys: The keys of the groups to move.
        :param parent: The group, project, or rack to move the groups under.
        """
        return self._client.send(
            "/group/set-parent",
            _SetParentReq(keys=keys, parent=ID(parent) if parent is not None else None),
            Empty,
        )
