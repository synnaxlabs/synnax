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
from synnax import group
from synnax.resource import ID, CrudeID
from synnax.schematic.symbol.types_gen import (
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Symbol,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    parent: ID
    symbols: list[Symbol]


class _CreateResponse(BaseModel):
    symbols: list[Symbol]


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
        :param parent: The resource ID of the resource to create the symbols under,
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
    ) -> Symbol: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Symbol: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[False] = ...,
    ) -> list[Symbol]: ...

    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        where: Where | None = None,
        search_term: str = "",
        one: bool = False,
        **fields: Any,
    ) -> Symbol | list[Symbol]:
        """Retrieves symbols. With no arguments, retrieves every symbol.

        Each filter field takes a value, a list of values, or an operator, and a symbol
        must match every field. A plain ``key`` returns one symbol.

        :param key: Keeps symbols with this key.
        :param and_: Keeps symbols that match every filter in the list.
        :param or_: Keeps symbols that match any filter in the list.
        :param not_: Keeps symbols that do not match the filter.
        :param where: A filter tree that the symbols must also match.
        :param keys: Keeps symbols with one of these keys.
        :param search_term: A fuzzy search term matched against symbol names.
        :param one: Returns one symbol for any filter.
        :returns: One symbol for a plain ``key`` or when ``one`` is set, otherwise a
            list of symbols. The list leaves out keys that do not exist.
        :raises NotFoundError: If one symbol is wanted and none matches.
        :raises MultipleFoundError: If one symbol is wanted and more than one matches.
        :raises ValidationError: If a filter field is None.
        """
        single = returns_one(fields, one, None, 0)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
        )
        res = self._client.send("/schematic/symbol/retrieve", req, RetrieveResponse)
        return sole("symbol", req, res.symbols) if single else res.symbols

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
