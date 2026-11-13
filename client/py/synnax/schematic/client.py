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
from synnax.schematic import symbol
from synnax.schematic.types_gen import (
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Schematic,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    schematics: list[Schematic]


class _CreateResponse(BaseModel):
    schematics: list[Schematic]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _CopyRequest(BaseModel):
    key: Key
    name: str


class _CopyResponse(BaseModel):
    schematic: Schematic


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
    """Client for creating, retrieving, renaming, copying, and deleting schematics.

    A schematic is a diagram of symbols and the connections between them. Every
    schematic belongs to a project.
    """

    symbols: symbol.Client
    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client
        self.symbols = symbol.Client(client)

    @overload
    def create(self, *, name: str, project: project_.Key) -> Schematic: ...

    @overload
    def create(self, schematics: Schematic) -> Schematic: ...

    @overload
    def create(self, schematics: list[Schematic]) -> list[Schematic]: ...

    def create(
        self,
        schematics: Schematic | list[Schematic] | None = None,
        *,
        name: str = "",
        project: project_.Key | None = None,
    ) -> Schematic | list[Schematic]:
        """Create one or more schematics.

        :param schematics: A single schematic or a list of them. Each one belongs to a
            project. When omitted, a schematic is created from the ``name`` and
            ``project`` keyword arguments.
        :param name: The name of the schematic to create when ``schematics`` is omitted.
        :param project: The key of the project that owns the schematic when
            ``schematics`` is omitted.
        :returns: The created schematic, or a list of created schematics when
            ``schematics`` is a list.
        :raises ValidationError: If a schematic does not belong to a project.
        """
        is_single = not isinstance(schematics, list)
        if schematics is None:
            schematics = Schematic(name=name, project=project)
        req = _CreateRequest(schematics=normalize(schematics))
        res = self._client.send("/schematic/create", req, _CreateResponse)
        return res.schematics[0] if is_single else res.schematics

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
    ) -> Schematic: ...

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
    ) -> Schematic: ...

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
    ) -> list[Schematic]: ...

    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        where: Where | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Schematic | list[Schematic]:
        """Retrieves schematics. With no arguments, retrieves every schematic.

        Each filter field takes a value, a list of values, or an operator, and a
        schematic must match every field. A plain ``key`` returns one schematic.

        :param key: Keeps schematics with this key.
        :param and_: Keeps schematics that match every filter in the list.
        :param or_: Keeps schematics that match any filter in the list.
        :param not_: Keeps schematics that do not match the filter.
        :param where: A filter tree that the schematics must also match.
        :param keys: Keeps schematics with one of these keys.
        :param projects: Keeps the schematics owned by any of these projects.
        :param groups: Keeps schematics in any of these groups.
        :param grouped: Keeps only schematics in a group when True, and only schematics
            in no group when False.
        :param one: Returns one schematic for any filter.
        :returns: One schematic for a plain ``key`` or when ``one`` is set, otherwise a
            list of schematics. The list leaves out keys that do not exist.
        :raises NotFoundError: If one schematic is wanted and none matches.
        :raises MultipleFoundError: If one schematic is wanted and more than one
            matches.
        :raises ValidationError: If a filter field is None.
        """
        single = returns_one(fields, one)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields)
        )
        res = self._client.send("/schematic/retrieve", req, RetrieveResponse)
        return sole("schematic", req, res.schematics) if single else res.schematics

    def rename(self, key: Key, name: str) -> None:
        """Renames a schematic.

        :param key: The key of the schematic to rename.
        :param name: The new name for the schematic.
        :raises NotFoundError: If no schematic exists for key.
        """
        action = _SetAction(set=_SetPayload(path=["name"], value=name))
        self._client.send(
            "/schematic/dispatch",
            _DispatchRequest(key=key, actions=[action]),
            Empty,
        )

    def copy(self, key: Key, *, name: str) -> Schematic:
        """Copies a schematic into the project of the original.

        :param key: The key of the schematic to copy.
        :param name: The name of the copy.
        :returns: The copy.
        :raises NotFoundError: If no schematic exists for key.
        """
        return self._client.send(
            "/schematic/copy", _CopyRequest(key=key, name=name), _CopyResponse
        ).schematic

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes one or more schematics by key.

        :param keys: A single schematic key or a list of keys to delete.
        """
        self._client.send(
            "/schematic/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )
