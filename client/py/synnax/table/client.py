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
from synnax import group as group_, project as project_
from synnax.table.types_gen import (
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Table,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    tables: list[Table]


class _CreateResponse(BaseModel):
    tables: list[Table]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _SetPayload(BaseModel):
    path: list[str]
    value: Any


class _SetAction(BaseModel):
    type: Literal["set"] = "set"
    set: _SetPayload


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[_SetAction]


class Client:
    """Client for creating, retrieving, renaming, and deleting tables in a Core."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, *, name: str, project: project_.Key) -> Table: ...

    @overload
    def create(self, tables: Table) -> Table: ...

    @overload
    def create(self, tables: list[Table]) -> list[Table]: ...

    def create(
        self,
        tables: Table | list[Table] | None = None,
        *,
        name: str = "",
        project: project_.Key | None = None,
    ) -> Table | list[Table]:
        """Create one or more tables.

        :param tables: A single table or a list of them. Each one belongs to a project.
            When omitted, a table is created from the ``name`` and ``project`` keyword
            arguments.
        :param name: The name of the table to create when ``tables`` is omitted.
        :param project: The key of the project that owns the table when ``tables`` is
            omitted.
        :returns: The created table, or a list of created tables when ``tables`` is a
            list.
        :raises ValidationError: If a table does not belong to a project.
        """
        is_single = not isinstance(tables, list)
        if tables is None:
            tables = Table(name=name, project=project)
        req = _CreateRequest(tables=normalize(tables))
        res = self._client.send("/table/create", req, _CreateResponse)
        return res.tables[0] if is_single else res.tables

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        projects: project_.Key | list[project_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        one: bool = ...,
    ) -> Table: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        projects: project_.Key | list[project_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        one: Literal[True],
    ) -> Table: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        projects: project_.Key | list[project_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        one: Literal[False] = ...,
    ) -> list[Table]: ...

    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        where: Where | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Table | list[Table]:
        """Retrieves tables. With no arguments, retrieves every table.

        Each filter field takes a value, a list of values, or an operator, and a table
        must match every field. A plain ``key`` returns one table.

        :param key: Keeps tables with this key.
        :param and_: Keeps tables that match every filter in the list.
        :param or_: Keeps tables that match any filter in the list.
        :param not_: Keeps tables that do not match the filter.
        :param where: A filter tree that the tables must also match.
        :param keys: Keeps tables with one of these keys.
        :param projects: Keeps the tables owned by any of these projects.
        :param groups: Keeps tables in any of these groups.
        :param grouped: Keeps only tables in a group when True, and only tables in no
            group when False.
        :param one: Returns one table for any filter.
        :returns: One table for a plain ``key`` or when ``one`` is set, otherwise a list
            of tables. The list leaves out keys that do not exist.
        :raises NotFoundError: If one table is wanted and none matches.
        :raises MultipleFoundError: If one table is wanted and more than one matches.
        :raises ValidationError: If a filter field is None.
        """
        single = returns_one(fields, one)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields)
        )
        res = self._client.send("/table/retrieve", req, RetrieveResponse)
        return sole("table", req, res.tables) if single else res.tables

    def rename(self, key: Key, name: str) -> None:
        """Rename a table.

        :param key: The key of the table to rename.
        :param name: The new name for the table.
        """
        action = _SetAction(set=_SetPayload(path=["name"], value=name))
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/table/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more tables by key.

        :param keys: A single table key or a list of keys to delete.
        """
        self._client.send("/table/delete", _DeleteRequest(keys=normalize(keys)), Empty)
