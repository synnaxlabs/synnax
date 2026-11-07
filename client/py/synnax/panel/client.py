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
from synnax import project as project_
from synnax import ranger
from synnax.panel.types_gen import (
    Key,
    Panel,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    panels: list[Panel]


class _CreateResponse(BaseModel):
    panels: list[Panel]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _RenamePayload(BaseModel):
    name: str


class _RenameAction(BaseModel):
    type: Literal["rename"] = "rename"
    rename: _RenamePayload


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[_RenameAction]


class Client:
    """Client for creating, retrieving, renaming, and deleting panels on a Synnax
    Core. A panel is a tree of visualization tabs owned by a project.
    """

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, *, name: str, project: project_.Key) -> Panel: ...

    @overload
    def create(self, panels: Panel) -> Panel: ...

    @overload
    def create(self, panels: list[Panel]) -> list[Panel]: ...

    def create(
        self,
        panels: Panel | list[Panel] | None = None,
        *,
        name: str = "",
        project: project_.Key | None = None,
    ) -> Panel | list[Panel]:
        """Create one or more panels.

        :param panels: A single panel or a list of them. When omitted, a panel is
            created from the ``name`` and ``project`` keyword arguments.
        :param name: The name of the panel to create when ``panels`` is omitted.
        :param project: The key of the project that owns the panel when ``panels`` is
            omitted.
        :returns: The created panel, or a list of created panels when ``panels`` is a
            list.
        :raises ValidationError: If a panel has no project.
        """
        is_single = not isinstance(panels, list)
        if panels is None:
            panels = [Panel(name=name, project=project)]
        req = _CreateRequest(panels=normalize(panels))
        res = self._client.send("/panel/create", req, _CreateResponse)
        return res.panels[0] if is_single else res.panels

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more panels by key.

        :param keys: A single panel key or a list of keys to delete.
        """
        self._client.send("/panel/delete", _DeleteRequest(keys=normalize(keys)), Empty)

    def rename(self, key: Key, name: str) -> None:
        """Rename a panel.

        :param key: The key of the panel to rename.
        :param name: The new name for the panel.
        """
        req = _DispatchRequest(
            key=key, actions=[_RenameAction(rename=_RenamePayload(name=name))]
        )
        self._client.send("/panel/dispatch", req, Empty)

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        projects: project_.Key | list[project_.Key] = ...,
        ranges: ranger.Key | list[ranger.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Panel: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        projects: project_.Key | list[project_.Key] = ...,
        ranges: ranger.Key | list[ranger.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Panel: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        projects: project_.Key | list[project_.Key] = ...,
        ranges: ranger.Key | list[ranger.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        offset: int = ...,
        one: Literal[False] = ...,
    ) -> list[Panel]: ...

    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        where: Where | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        one: bool = False,
        **fields: Any,
    ) -> Panel | list[Panel]:
        """Retrieves panels. With no arguments, retrieves every panel.

        Each filter field takes a value, a list of values, or an operator, and a panel
        must match every field. A plain ``key`` returns one panel.

        :param key: Keeps panels with this key.
        :param and_: Keeps panels that match every filter in the list.
        :param or_: Keeps panels that match any filter in the list.
        :param not_: Keeps panels that do not match the filter.
        :param where: A filter tree that the panels must also match.
        :param keys: Keeps panels with one of these keys.
        :param projects: Keeps the panels owned by any of these projects.
        :param ranges: Keeps the panels owned by any of these ranges.
        :param search_term: A fuzzy search term matched against panel names.
        :param limit: The maximum number of panels to retrieve.
        :param offset: The number of panels to skip.
        :param one: Returns one panel for any filter.
        :returns: One panel for a plain ``key`` or when ``one`` is set, otherwise a list
            of panels. The list leaves out keys that do not exist.
        :raises NotFoundError: If one panel is wanted and none matches.
        :raises MultipleFoundError: If one panel is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one panel is wanted
            beside a limit or an offset.
        """
        single = returns_one(fields, one, limit, offset)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            offset=offset,
        )
        res = self._client.send("/panel/retrieve", req, RetrieveResponse)
        return sole("Panel", fields, res.panels) if single else res.panels
