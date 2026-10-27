#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax import group
from synnax.ontology.payload import ID, CrudeID
from synnax.schematic.symbol.types_gen import Key, Symbol
from x.lists import normalize


class _CreateRequest(BaseModel):
    parent: ID
    symbols: list[Symbol]


class _CreateResponse(BaseModel):
    symbols: list[Symbol]


class _RetrieveRequest(BaseModel):
    keys: list[Key] | None = None
    search_term: str | None = None


class _RetrieveResponse(BaseModel):
    symbols: list[Symbol] = Field(default_factory=list)


class _RenameRequest(BaseModel):
    key: Key
    name: str


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _RetrieveGroupResponse(BaseModel):
    group: group.Group


class _DeleteGroupRequest(BaseModel):
    key: group.Key


class Client:
    """Client for creating, retrieving, renaming, and deleting schematic symbols.

    A symbol is a reusable visual component that schematics place on their diagrams.
    """

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, symbols: Symbol, *, parent: CrudeID) -> Symbol: ...

    @overload
    def create(self, symbols: list[Symbol], *, parent: CrudeID) -> list[Symbol]: ...

    def create(
        self, symbols: Symbol | list[Symbol], *, parent: CrudeID
    ) -> Symbol | list[Symbol]:
        """Creates one or more symbols under a parent resource.

        :param symbols: A single symbol or a list of them.
        :param parent: The ontology ID of the resource to create the symbols under,
            such as the group from ``retrieve_group``.
        :returns: The created symbol, or a list of created symbols when ``symbols`` is
            a list.
        """
        is_single = not isinstance(symbols, list)
        res = self._client.send(
            "/schematic/symbol/create",
            _CreateRequest(parent=ID(parent), symbols=normalize(symbols)),
            _CreateResponse,
        )
        return res.symbols[0] if is_single else res.symbols

    @overload
    def retrieve(self, *, key: Key) -> Symbol: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        search_term: str | None = None,
    ) -> list[Symbol]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        search_term: str | None = None,
    ) -> Symbol | list[Symbol]:
        """Retrieves one or more symbols. With no arguments, it retrieves every symbol.

        :param key: The key of a single symbol to retrieve.
        :param keys: The keys of the symbols to retrieve.
        :param search_term: A fuzzy search term matched against symbol names.
        :returns: The matching symbol when ``key`` is provided, otherwise a list of
            matching symbols.
        :raises NotFoundError: If a requested key does not exist.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        res = self._client.send(
            "/schematic/symbol/retrieve",
            _RetrieveRequest(keys=keys, search_term=search_term),
            _RetrieveResponse,
        )
        return res.symbols[0] if is_single else res.symbols

    def rename(self, key: Key, name: str) -> None:
        """Renames a symbol.

        :param key: The key of the symbol to rename.
        :param name: The new name for the symbol.
        """
        self._client.send(
            "/schematic/symbol/rename", _RenameRequest(key=key, name=name), Empty
        )

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes one or more symbols by key.

        :param keys: A single symbol key or a list of keys to delete.
        """
        self._client.send(
            "/schematic/symbol/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    def retrieve_group(self) -> group.Group:
        """Retrieves the group that holds every symbol by default.

        :returns: The symbol group.
        """
        return self._client.send(
            "/schematic/symbol/retrieve-group", Empty(), _RetrieveGroupResponse
        ).group

    def delete_group(self, key: group.Key) -> None:
        """Deletes a group and every symbol in it in one transaction. A child that is
        not a symbol moves to the group from ``retrieve_group``.

        :param key: The key of the group to delete.
        """
        self._client.send(
            "/schematic/symbol/group/delete", _DeleteGroupRequest(key=key), Empty
        )
