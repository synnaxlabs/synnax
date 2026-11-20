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
from synnax import group as group_
from synnax.color import actions_gen
from synnax.color.types_gen import (
    Cursor,
    Key,
    Named,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.exceptions import ValidationError
from synnax.util import request_where, returns_one, sole
from x import color as color_, filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    colors: list[Named]


_CreateResponse = _CreateRequest


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    """Client for creating, retrieving, changing, and deleting named colors in a
    Core."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(
        self, *, name: str, light: color_.Crude, dark: color_.Crude | None = None
    ) -> Named: ...

    @overload
    def create(self, colors: Named) -> Named: ...

    @overload
    def create(self, colors: list[Named]) -> list[Named]: ...

    def create(
        self,
        colors: Named | list[Named] | None = None,
        *,
        name: str | None = None,
        light: color_.Crude | None = None,
        dark: color_.Crude | None = None,
    ) -> Named | list[Named]:
        """Creates one or more named colors.

        :param colors: A single named color or a list of them. When omitted, a named
            color is created from the ``name``, ``light``, and ``dark`` keyword
            arguments.
        :param name: The name of the color to create when ``colors`` is omitted.
        :param light: The color in the light theme when ``colors`` is omitted.
        :param dark: The color in the dark theme when ``colors`` is omitted. When
            None, the theme derives it.
        :returns: The created named color, or a list of them when ``colors`` is a
            list.
        :raises ValidationError: If ``colors`` is omitted and ``name`` or ``light`` is
            too, or if a named color with a given key already exists.
        """
        is_single = not isinstance(colors, list)
        if colors is None:
            if name is None or light is None:
                raise ValidationError("A named color needs a name and a light color.")
            colors = Named(
                name=name,
                light=color_.Color(light),
                dark=None if dark is None else color_.Color(dark),
            )
        req = _CreateRequest(colors=normalize(colors))
        res = self._client.send("/color/create", req, _CreateResponse)
        return res.colors[0] if is_single else res.colors

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Named: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Named: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Named: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
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
    ) -> list[Named]: ...

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
    ) -> Named | list[Named]:
        """Retrieves named colors from the Core. With no arguments, retrieves every
        named color.

        Each filter field takes a value, a list of values, or an operator, and a named
        color must match every field. A plain ``key`` or ``name`` returns one named
        color.

        :param key: Keeps named colors with this key.
        :param name: Keeps named colors with this name.
        :param groups: Keeps named colors in any of these palettes.
        :param grouped: Keeps only named colors in a palette when True, and only named
            colors in no palette when False.
        :param and_: Keeps named colors that match every filter in the list.
        :param or_: Keeps named colors that match any filter in the list.
        :param not_: Keeps named colors that do not match the filter.
        :param where: A filter tree that the named colors must also match.
        :param keys: Keeps named colors with one of these keys.
        :param names: Keeps named colors with one of these names.
        :param search_term: A fuzzy search term matched against the names.
        :param limit: The maximum number of named colors to retrieve.
        :param order: The order of the named colors. Defaults to key order.
        :param after: Retrieves the named colors past this position in the order: the
            key of the last named color of the page before, and its value of the order
            field.
        :param one: Returns one named color for any filter.
        :returns: One named color for a plain ``key`` or ``name`` or when ``one`` is
            set, otherwise a list of them. The list leaves out keys that do not exist.
        :raises NotFoundError: If one named color is wanted and none matches.
        :raises MultipleFoundError: If one named color is wanted and more than one
            matches.
        :raises ValidationError: If a filter field is None, or if one named color is
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
        res = self._client.send("/color/retrieve", req, RetrieveResponse)
        return sole("named color", req, res.colors) if single else res.colors

    def dispatch(
        self, key: Key, actions: actions_gen.Action | list[actions_gen.Action]
    ) -> None:
        """Applies actions to a named color, in order, in one transaction.

        :param key: The key of the named color.
        :param actions: The actions to apply, such as a ``set`` of ``name``,
            ``light``, or ``dark``.
        :raises NotFoundError: If no named color has the given key.
        :raises ValidationError: If an action is invalid.
        """
        req = _DispatchRequest(key=key, actions=normalize(actions))
        self._client.send("/color/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes one or more named colors. Deleting a named color that does not
        exist is not an error.

        :param keys: A single key or a list of keys to delete.
        """
        self._client.send("/color/delete", _DeleteRequest(keys=normalize(keys)), Empty)
