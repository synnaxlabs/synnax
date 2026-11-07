#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal, overload

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
)
from synnax.util import one
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
    def retrieve(self, *, key: Key) -> Panel: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        where: Where | None = None,
        projects: list[project_.Key] | None = None,
        ranges: list[ranger.Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> list[Panel]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        where: Where | None = None,
        projects: list[project_.Key] | None = None,
        ranges: list[ranger.Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> Panel | list[Panel]:
        """Retrieve one or more panels.

        :param key: The key of a single panel to retrieve. When provided, a single
            panel is returned and NotFoundError is raised if it does not exist.
        :param keys: The keys of the panels to retrieve.
        :param where: A filter that the panels must also match.
        :param projects: The keys of projects. Only the panels these projects own are
            returned.
        :param ranges: The keys of ranges. Only the panels these ranges own are
            returned.
        :param search_term: A fuzzy search term matched against panel names.
        :param limit: The maximum number of panels to return.
        :param offset: The number of panels to skip before returning results.
        :returns: The matching panel when ``key`` is provided, otherwise a list of
            matching panels. The list leaves out keys that do not exist.
        :raises NotFoundError: If ``key`` is provided and no panel exists for it.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            projects=projects,
            ranges=ranges,
        )
        res = self._client.send(
            "/panel/retrieve",
            RetrieveRequest(
                where=filter.merge(where, flat),
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            RetrieveResponse,
        )
        panels = res.panels
        if is_single:
            return one("Panel", key, panels)
        return panels
