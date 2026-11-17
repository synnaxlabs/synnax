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
from synnax.schematic.symbol import actions_gen
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
    symbols: list[Symbol]


class _CreateResponse(BaseModel):
    symbols: list[Symbol]


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _DeleteRequest(BaseModel):
    keys: list[Key]


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
    def create(self, symbols: Symbol) -> Symbol: ...

    @overload
    def create(self, symbols: list[Symbol]) -> list[Symbol]: ...

    def create(self, symbols: Symbol | list[Symbol]) -> Symbol | list[Symbol]:
        """Creates one or more symbols. A symbol's ``group`` places it; a symbol with
        no group sits in none.

        :param symbols: A single symbol or a list of them.
        :returns: The created symbol, or a list of created symbols when ``symbols`` is
            a list.
        """
        is_single = not isinstance(symbols, list)
        res = self._client.send(
            "/schematic/symbol/create",
            _CreateRequest(symbols=normalize(symbols)),
            _CreateResponse,
        )
        return res.symbols[0] if is_single else res.symbols

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        groups: group.Key | list[group.Key] = ...,
        grouped: bool = ...,
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
        groups: group.Key | list[group.Key] = ...,
        grouped: bool = ...,
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
        groups: group.Key | list[group.Key] = ...,
        grouped: bool = ...,
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
        :param groups: Keeps symbols in any of these groups.
        :param grouped: Keeps only symbols in a group when True, and only symbols in no
            group when False.
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
        single = returns_one(fields, one)
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
        action = actions_gen.set(actions_gen.SetPayload(path=["name"], value=name))
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/schematic/symbol/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes one or more symbols by key.

        :param keys: A single symbol key or a list of keys to delete.
        """
        self._client.send(
            "/schematic/symbol/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    def delete_group(self, key: group.Key) -> None:
        """Deletes a group and every symbol in it in one transaction.

        :param key: The key of the group to delete.
        """
        self._client.send(
            "/schematic/symbol/group/delete", _DeleteGroupRequest(key=key), Empty
        )
