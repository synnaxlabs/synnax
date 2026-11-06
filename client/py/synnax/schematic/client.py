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
from synnax import project as project_
from synnax import ranger
from synnax.schematic import symbol
from synnax.schematic.types_gen import Key, Schematic
from x.lists import normalize


class _CreateRequest(BaseModel):
    schematics: list[Schematic]


class _CreateResponse(BaseModel):
    schematics: list[Schematic]


class _RetrieveRequest(BaseModel):
    keys: list[Key] | None = None
    projects: list[project_.Key] | None = None
    ranges: list[ranger.Key] | None = None


class _RetrieveResponse(BaseModel):
    schematics: list[Schematic] = Field(default_factory=list)


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _CopyRequest(BaseModel):
    key: Key
    name: str
    snapshot: bool
    range: ranger.Key | None = None


class _CopyResponse(BaseModel):
    schematic: Schematic


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
    """Client for creating, retrieving, renaming, copying, and deleting schematics.

    A schematic is a diagram of symbols and the connections between them. Every
    schematic belongs to a project, or to a range when it is a snapshot.
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

        :param schematics: A single schematic or a list of them. Each one belongs to
            exactly one of a project and a range. When omitted, a schematic is created
            from the ``name`` and ``project`` keyword arguments.
        :param name: The name of the schematic to create when ``schematics`` is omitted.
        :param project: The key of the project that owns the schematic when
            ``schematics`` is omitted.
        :returns: The created schematic, or a list of created schematics when
            ``schematics`` is a list.
        :raises ValidationError: If a schematic does not belong to exactly one of a
            project and a range.
        """
        is_single = not isinstance(schematics, list)
        if schematics is None:
            schematics = Schematic(name=name, project=project)
        req = _CreateRequest(schematics=normalize(schematics))
        res = self._client.send("/schematic/create", req, _CreateResponse)
        return res.schematics[0] if is_single else res.schematics

    @overload
    def retrieve(self, *, key: Key) -> Schematic: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        projects: list[project_.Key] | None = None,
        ranges: list[ranger.Key] | None = None,
    ) -> list[Schematic]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        projects: list[project_.Key] | None = None,
        ranges: list[ranger.Key] | None = None,
    ) -> Schematic | list[Schematic]:
        """Retrieves one or more schematics.

        :param key: The key of a single schematic to retrieve.
        :param keys: The keys of the schematics to retrieve.
        :param projects: The keys of projects. Only the schematics these projects own
            are returned.
        :param ranges: The keys of ranges. Only the schematics these ranges own are
            returned.
        :returns: The matching schematic when ``key`` is provided, otherwise a list of
            matching schematics.
        :raises NotFoundError: If a requested key does not exist.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        res = self._client.send(
            "/schematic/retrieve",
            _RetrieveRequest(keys=keys, projects=projects, ranges=ranges),
            _RetrieveResponse,
        )
        return res.schematics[0] if is_single else res.schematics

    def rename(self, key: Key, name: str) -> None:
        """Renames a schematic.

        :param key: The key of the schematic to rename.
        :param name: The new name for the schematic.
        :raises NotFoundError: If no schematic exists for key.
        """
        action = _RenameAction(rename=_RenamePayload(name=name))
        self._client.send(
            "/schematic/dispatch",
            _DispatchRequest(key=key, actions=[action]),
            Empty,
        )

    def copy(
        self,
        key: Key,
        *,
        name: str,
        snapshot: bool = False,
        range: ranger.Key | None = None,
    ) -> Schematic:
        """Copies a schematic. The copy belongs to the range when one is given, and to
        the project of the original otherwise.

        :param key: The key of the schematic to copy.
        :param name: The name of the copy.
        :param snapshot: True to make the copy an immutable snapshot.
        :param range: The key of the range that owns the copy.
        :returns: The copy.
        :raises NotFoundError: If no schematic exists for key, or range is given and
            no range exists for it.
        """
        return self._client.send(
            "/schematic/copy",
            _CopyRequest(key=key, name=name, snapshot=snapshot, range=range),
            _CopyResponse,
        ).schematic

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes one or more schematics by key.

        :param keys: A single schematic key or a list of keys to delete.
        """
        self._client.send(
            "/schematic/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )
