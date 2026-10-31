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

from freighter import Empty, UnaryClient
from synnax import util
from synnax.group import builder_gen
from synnax.group.builder_gen import Condition
from synnax.group.types_gen import Group, RetrieveRequest, RetrieveResponse, Where
from synnax.ontology.payload import ID, CrudeID
from x import filter


class _CreateReq(BaseModel):
    parent: ID
    key: UUID | None = None
    name: str


class _CreateRes(BaseModel):
    group: Group


class _RenameReq(BaseModel):
    key: UUID
    name: str


class _DeleteReq(BaseModel):
    keys: list[UUID]


class Client:
    _client: UnaryClient

    def __init__(self, client: UnaryClient):
        self._client = client

    def create(self, parent: CrudeID, name: str, *, key: str | None = None) -> Group:
        return self._client.send(
            "/ontology/create-group",
            _CreateReq(parent=ID(parent), key=UUID(key) if key else None, name=name),
            _CreateRes,
        ).group

    @overload
    def retrieve(self, key: UUID, /) -> Group: ...

    @overload
    def retrieve(self, keys: list[UUID], /) -> list[Group]: ...

    @overload
    def retrieve(self, *, key: UUID) -> Group: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[True],
        keys: list[UUID] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> Group: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[False] = False,
        keys: list[UUID] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> list[Group]: ...

    def retrieve(
        self,
        *args: UUID | list[UUID] | Condition,
        key: UUID | None = None,
        one: bool = False,
        keys: list[UUID] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> Group | list[Group]:
        """Retrieves groups that match every condition and filter. Returns one group for
        a key or one, and a list otherwise.

        :param args: Conditions from the group builder, such as ``group.key.in_(keys)``,
            or one key or list of keys.
        :param one: Returns the one group that matches, instead of a list.
        :raises NotFoundError: If no group matches a key or one.
        :raises MultipleFoundError: If one is True and more than one group matches.
        :raises TypeError: If args are neither group conditions nor a shortcut.
        """
        conditions, query = util.shortcut(
            "Group", args, Condition, UUID, builder_gen.key
        )
        if key is not None:
            keys = [key]
            query = key
        flat = Where(key=None if keys is None else filter.Set[UUID](in_=keys))
        where = filter.merge(filter.join(Where, conditions), flat)
        res = self._client.send(
            "/ontology/retrieve-group",
            RetrieveRequest(
                where=where,
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            RetrieveResponse,
        )
        if query is None and not one:
            return res.groups
        return util.one("Group", where if query is None else query, res.groups)

    def rename(self, key: UUID, name: str) -> Empty:
        return self._client.send(
            "/ontology/rename-group", _RenameReq(key=key, name=name), Empty
        )

    def delete(self, keys: list[UUID]) -> Empty:
        return self._client.send("/ontology/delete-group", _DeleteReq(keys=keys), Empty)
