#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, overload
from uuid import UUID

from pydantic import BaseModel

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient, send_required
from synnax.exceptions import NotFoundError
from synnax.view.types_gen import Key, View
from x.normalize import normalize


class _CreateRequest(BaseModel):
    views: list[View]


_CreateResponse = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _RetrieveRequest(BaseModel):
    keys: list[Key] | None = None
    types: list[str] | None = None
    search_term: str | None = None
    limit: int | None = None
    offset: int | None = None


class _RetrieveResponse(BaseModel):
    views: list[View] | None = None


class Client:
    _client: UnaryClient
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        *,
        name: str,
        type: str,
        query: dict[str, Any] | None = None,
    ) -> View: ...

    @overload
    def create(
        self,
        views: View,
    ) -> View: ...

    @overload
    def create(
        self,
        views: list[View],
    ) -> list[View]: ...

    def create(
        self,
        views: list[View] | View | None = None,
        *,
        name: str = "",
        type: str = "",
        query: dict[str, Any] | None = None,
    ) -> View | list[View]:
        is_single = not isinstance(views, list)
        if views is None:
            views = [
                View(
                    key=UUID(int=0),
                    name=name,
                    type=type,
                    query=query if query is not None else {},
                )
            ]
        req = _CreateRequest(views=normalize(views))
        res = send_required(
            self._client,
            "/view/create",
            req,
            _CreateResponse,
        )
        return res.views[0] if is_single else res.views

    def delete(self, keys: Key | list[Key]) -> None:
        if not isinstance(keys, list):
            keys = [keys]
        send_required(self._client, "/view/delete", _DeleteRequest(keys=keys), Empty)

    @overload
    def retrieve(
        self,
        *,
        key: Key,
    ) -> View: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        types: list[str] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> list[View]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        types: list[str] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> View | list[View]:
        is_single = key is not None
        if key is not None:
            keys = [key]
        res = send_required(
            self._client,
            "/view/retrieve",
            _RetrieveRequest(
                keys=keys,
                types=types,
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            _RetrieveResponse,
        )
        views = res.views if res.views is not None else []
        if is_single:
            if len(views) == 0:
                raise NotFoundError("View not found")
            return views[0]
        return views
