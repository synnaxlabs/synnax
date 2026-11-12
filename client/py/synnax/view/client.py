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

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.util import request_where, returns_one, sole
from synnax.view import actions_gen
from synnax.view.types_gen import (
    Cursor,
    Key,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Type,
    View,
    Where,
    WhereInput,
)
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    views: list[View]


_CreateResponse = _CreateRequest


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    _client: UnaryClient
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        *,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @overload
    def create(self, views: View) -> View: ...

    @overload
    def create(self, views: list[View]) -> list[View]: ...

    def create(self, views: View | list[View]) -> View | list[View]:
        is_single = not isinstance(views, list)
        req = _CreateRequest(views=normalize(views))
        res = self._client.send("/view/create", req, _CreateResponse)
        return res.views[0] if is_single else res.views

    def delete(self, keys: Key | list[Key]) -> None:
        if not isinstance(keys, list):
            keys = [keys]
        self._client.send("/view/delete", _DeleteRequest(keys=keys), Empty)

    def rename(self, key: Key, name: str) -> None:
        """Renames a view.

        :param key: The key of the view to rename.
        :param name: The new name for the view.
        :raises NotFoundError: If no view has the given key.
        """
        action = actions_gen.set(actions_gen.SetPayload(path=["name"], value=name))
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/view/dispatch", req, Empty)

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        type: filter.SetInput[Type] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> View: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        type: filter.SetInput[Type] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> View: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        type: filter.SetInput[Type] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        types: list[Type] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> View: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        type: filter.SetInput[Type] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        types: list[Type] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[View]: ...

    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        types: list[Type] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> View | list[View]:
        """Retrieves views. With no arguments, retrieves every view.

        Each filter field takes a value, a list of values, or an operator, and a view
        must match every field. A plain ``key`` or ``name`` returns one view.

        :param key: Keeps views with this key.
        :param name: Keeps views with this name.
        :param type: Keeps views of this type.
        :param and_: Keeps views that match every filter in the list.
        :param or_: Keeps views that match any filter in the list.
        :param not_: Keeps views that do not match the filter.
        :param where: A filter tree that the views must also match.
        :param keys: Keeps views with one of these keys.
        :param types: Keeps views of one of these types.
        :param search_term: A fuzzy search term matched against view names.
        :param limit: The maximum number of views to retrieve.
        :param order: The order of the views. Defaults to key order.
        :param after: Retrieves the views past this position in the order: the key of
            the last view of the page before, and its value of the order field.
        :param one: Returns one view for any filter.
        :returns: One view for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of views. The list leaves out keys that do not exist.
        :raises NotFoundError: If one view is wanted and none matches.
        :raises MultipleFoundError: If one view is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one view is wanted
            beside a limit or a cursor.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            type=None if types is None else filter.Set[Type](in_=types),
        )
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
        )
        res = self._client.send("/view/retrieve", req, RetrieveResponse)
        return sole("view", req, res.views) if single else res.views
