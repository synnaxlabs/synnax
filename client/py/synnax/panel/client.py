#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal, overload

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax import ontology
from synnax.exceptions import NotFoundError
from synnax.panel.types_gen import ONTOLOGY_TYPE, Key, Panel
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


class _RetrieveRequest(BaseModel):
    keys: list[Key] | None = None
    search_term: str | None = None
    limit: int | None = None
    offset: int | None = None


class _RetrieveResponse(BaseModel):
    panels: list[Panel] = Field(default_factory=list)


class Client:
    """Client for creating, retrieving, renaming, and deleting panels on a Synnax
    Core. A panel is a tree of visualization tabs owned by a parent resource, such as
    a project.
    """

    _client: UnaryClient
    _ontology: ontology.Client

    def __init__(self, client: UnaryClient, ontology: ontology.Client) -> None:
        self._client = client
        self._ontology = ontology

    @overload
    def create(self, *, name: str, parent: ontology.ID) -> Panel: ...

    @overload
    def create(self, panels: Panel) -> Panel: ...

    @overload
    def create(self, panels: list[Panel]) -> list[Panel]: ...

    def create(
        self,
        panels: Panel | list[Panel] | None = None,
        *,
        name: str = "",
        parent: ontology.ID | None = None,
    ) -> Panel | list[Panel]:
        """Create one or more panels.

        :param panels: A single panel or a list of them. When omitted, a panel is
            created from the ``name`` and ``parent`` keyword arguments.
        :param name: The name of the panel to create when ``panels`` is omitted.
        :param parent: The ontology ID of the resource that owns the panel, such as a
            project, when ``panels`` is omitted.
        :returns: The created panel, or a list of created panels when ``panels`` is a
            list.
        :raises ValidationError: If a panel has no parent.
        """
        is_single = not isinstance(panels, list)
        if panels is None:
            panels = [Panel(name=name, parent=parent)]
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
        parent: ontology.ID | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> list[Panel]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        parent: ontology.ID | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> Panel | list[Panel]:
        """Retrieve one or more panels.

        :param key: The key of a single panel to retrieve. When provided, a single
            panel is returned and NotFoundError is raised if it does not exist.
        :param keys: The keys of the panels to retrieve.
        :param parent: The ontology ID of a resource, such as a project. Only the
            panels that are children of this resource are returned. Replaces ``keys``.
        :param search_term: A fuzzy search term matched against panel names.
        :param limit: The maximum number of panels to return.
        :param offset: The number of panels to skip before returning results.
        :returns: The matching panel when ``key`` is provided, otherwise a list of
            matching panels.
        :raises NotFoundError: If ``key`` is provided and no panel exists for it.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        if parent is not None:
            keys = []
            for child in self._ontology.retrieve_children(parent):
                if child.type != ONTOLOGY_TYPE.type:
                    continue
                if child.key is None:
                    raise ValueError(f"panel child of {parent} has no key")
                keys.append(Key(child.key))
            if len(keys) == 0:
                return []
        res = self._client.send(
            "/panel/retrieve",
            _RetrieveRequest(
                keys=keys,
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            _RetrieveResponse,
        )
        panels = res.panels
        if is_single:
            if len(panels) == 0:
                raise NotFoundError("Panel not found")
            return panels[0]
        return panels
