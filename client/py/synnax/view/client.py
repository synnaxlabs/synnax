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

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax import util
from synnax.view import builder_gen
from synnax.view.builder_gen import Condition
from synnax.view.types_gen import (
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Type,
    View,
    Where,
)
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    views: list[View]


_CreateResponse = _CreateRequest


class _RenameRequest(BaseModel):
    key: Key
    name: str


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
        self._client.send("/view/rename", _RenameRequest(key=key, name=name), Empty)

    @overload
    def retrieve(self, key: Key | str, /) -> View: ...

    @overload
    def retrieve(self, keys: list[Key] | list[str], /) -> list[View]: ...

    @overload
    def retrieve(self, *, key: Key) -> View: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[True],
        keys: list[Key] | None = None,
        types: list[Type] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> View: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[False] = False,
        keys: list[Key] | None = None,
        types: list[Type] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> list[View]: ...

    def retrieve(
        self,
        *args: Key | str | list[Key] | list[str] | Condition,
        key: Key | None = None,
        one: bool = False,
        keys: list[Key] | None = None,
        types: list[Type] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> View | list[View]:
        """Retrieves views that match every condition and filter. Returns one view for a
        key, a name, or one, and a list otherwise.

        :param args: Conditions from the view builder, such as
            ``view.type.eq("range")``, or one key, name, or list of either.
        :param key: The key of one view to retrieve.
        :param one: Returns the one view that matches, instead of a list.
        :param keys: Keeps views with one of these keys.
        :param types: Keeps views of one of these types.
        :param search_term: Fuzzy matches view names.
        :param limit: The maximum number of views to return.
        :param offset: The number of matching views to skip.
        :raises NotFoundError: If no view matches a key, a name, or one.
        :raises MultipleFoundError: If more than one view matches a name or one.
        :raises TypeError: If args are neither view conditions nor a shortcut.
        """
        conditions, query = util.shortcut(
            "View", args, Condition, Key, builder_gen.key, builder_gen.name
        )
        if key is not None:
            keys = [key]
            query = key
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            type=None if types is None else filter.Set[Type](in_=types),
        )
        where = filter.merge(filter.join(Where, conditions), flat)
        res = self._client.send(
            "/view/retrieve",
            RetrieveRequest(
                where=where,
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            RetrieveResponse,
        )
        views = res.views
        if query is None and not one:
            return views
        return util.one("view", where, views)
