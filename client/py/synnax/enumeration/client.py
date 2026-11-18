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
from synnax.enumeration import actions_gen
from synnax.enumeration.types_gen import (
    Cursor,
    Entry,
    Enumeration,
    Key,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.exceptions import ValidationError
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    enumerations: list[Enumeration]


_CreateResponse = _CreateRequest


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    """Client for creating, retrieving, changing, and deleting enumerations in a
    Core."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(
        self, *, name: str, entries: list[Entry] | None = None
    ) -> Enumeration: ...

    @overload
    def create(self, enumerations: Enumeration) -> Enumeration: ...

    @overload
    def create(self, enumerations: list[Enumeration]) -> list[Enumeration]: ...

    def create(
        self,
        enumerations: Enumeration | list[Enumeration] | None = None,
        *,
        name: str | None = None,
        entries: list[Entry] | None = None,
    ) -> Enumeration | list[Enumeration]:
        """Creates one or more enumerations.

        :param enumerations: A single enumeration or a list of them. When omitted, an
            enumeration is created from the ``name`` and ``entries`` keyword
            arguments.
        :param name: The name of the enumeration to create when ``enumerations`` is
            omitted.
        :param entries: The entries of the enumeration to create when
            ``enumerations`` is omitted.
        :returns: The created enumeration, or a list of them when ``enumerations`` is
            a list.
        :raises ValidationError: If ``enumerations`` and ``name`` are both omitted,
            if two entries share a key or a value, or if an enumeration with a given
            key already exists.
        """
        is_single = not isinstance(enumerations, list)
        if enumerations is None:
            if name is None:
                raise ValidationError("An enumeration needs a name.")
            enumerations = Enumeration(name=name, entries=entries or [])
        req = _CreateRequest(enumerations=normalize(enumerations))
        res = self._client.send("/enumeration/create", req, _CreateResponse)
        return res.enumerations[0] if is_single else res.enumerations

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Enumeration: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Enumeration: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Enumeration: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[Enumeration]: ...

    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Enumeration | list[Enumeration]:
        """Retrieves enumerations from the Core. With no arguments, retrieves every
        enumeration.

        Each filter field takes a value, a list of values, or an operator, and an
        enumeration must match every field. A plain ``key`` or ``name`` returns one
        enumeration.

        :param key: Keeps enumerations with this key.
        :param name: Keeps enumerations with this name.
        :param and_: Keeps enumerations that match every filter in the list.
        :param or_: Keeps enumerations that match any filter in the list.
        :param not_: Keeps enumerations that do not match the filter.
        :param where: A filter tree that the enumerations must also match.
        :param keys: Keeps enumerations with one of these keys.
        :param names: Keeps enumerations with one of these names.
        :param search_term: A fuzzy search term matched against the names.
        :param limit: The maximum number of enumerations to retrieve.
        :param order: The order of the enumerations. Defaults to key order.
        :param after: Retrieves the enumerations past this position in the order: the
            key of the last enumeration of the page before, and its value of the order
            field.
        :param one: Returns one enumeration for any filter.
        :returns: One enumeration for a plain ``key`` or ``name`` or when ``one`` is
            set, otherwise a list of them. The list leaves out keys that do not exist.
        :raises NotFoundError: If one enumeration is wanted and none matches.
        :raises MultipleFoundError: If one enumeration is wanted and more than one
            matches.
        :raises ValidationError: If a filter field is None, or if one enumeration is
            wanted beside a limit or a cursor.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
        )
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
        )
        res = self._client.send("/enumeration/retrieve", req, RetrieveResponse)
        return (
            sole("enumeration", req, res.enumerations) if single else res.enumerations
        )

    def dispatch(
        self, key: Key, actions: actions_gen.Action | list[actions_gen.Action]
    ) -> None:
        """Applies actions to an enumeration, in order, in one transaction.

        :param key: The key of the enumeration.
        :param actions: The actions to apply, such as a ``set`` of ``name`` or an
            ``insert`` into ``entries``.
        :raises NotFoundError: If no enumeration has the given key.
        :raises ValidationError: If an action is invalid, or if two entries would
            share a key or a value.
        """
        req = _DispatchRequest(key=key, actions=normalize(actions))
        self._client.send("/enumeration/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes one or more enumerations. Deleting an enumeration that does not
        exist is not an error.

        :param keys: A single key or a list of keys to delete.
        :raises ValidationError: If an enumeration is built in or a record uses it.
            The Core then deletes none.
        """
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/enumeration/delete", req, Empty)
