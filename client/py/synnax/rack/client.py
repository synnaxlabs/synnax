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

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.exceptions import NotFoundError
from synnax.rack.types_gen import Key, Rack, RetrieveRequest, Where
from synnax.util import one
from x import filter
from x.lists import all_none, override


class _CreateRequest(BaseModel):
    racks: list[Rack]


class _CreateResponse(BaseModel):
    racks: list[Rack]


class _RenameRequest(BaseModel):
    key: Key
    name: str


class _DeleteRequest(BaseModel):
    keys: list[int]


class _RetrieveResponse(BaseModel):
    racks: list[Rack] = Field(default_factory=list)


class Client:
    _client: UnaryClient
    _embedded_rack: Rack | None = None
    instrumentation: Instrumentation

    def __init__(
        self,
        *,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @overload
    def create(self, *, key: int = 0, name: str = "") -> Rack: ...

    @overload
    def create(self, racks: Rack) -> Rack: ...

    @overload
    def create(self, racks: list[Rack]) -> list[Rack]: ...

    def create(
        self,
        racks: Rack | list[Rack] | None = None,
        *,
        key: int = 0,
        name: str = "",
    ) -> Rack | list[Rack]:
        is_single = True
        if racks is None:
            racks = [Rack(key=key, name=name)]
        elif isinstance(racks, Rack):
            racks = [racks]
        else:
            is_single = False
        req = _CreateRequest(racks=racks)
        res = self._client.send("/rack/create", req, _CreateResponse)
        if is_single:
            return res.racks[0]
        return res.racks

    def delete(self, keys: list[int]) -> None:
        req = _DeleteRequest(keys=keys)
        self._client.send("/rack/delete", req, Empty)

    def rename(self, key: Key, name: str) -> None:
        """Renames a rack.

        :param key: The key of the rack to rename.
        :param name: The new name for the rack.
        :raises NotFoundError: If no rack has the given key.
        """
        self._client.send("/rack/rename", _RenameRequest(key=key, name=name), Empty)
        if self._embedded_rack is not None and self._embedded_rack.key == key:
            self._embedded_rack = None

    @overload
    def retrieve(self, key: int, *, include_status: bool = False) -> Rack: ...

    @overload
    def retrieve(self, *, name: str, include_status: bool = False) -> Rack: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[int] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        integration: str | None = None,
        embedded: bool | None = None,
        host_is_node: bool | None = None,
        limit: int | None = None,
        offset: int = 0,
        include_status: bool = False,
    ) -> list[Rack]: ...

    def retrieve(
        self,
        key: int | None = None,
        *,
        name: str | None = None,
        where: Where | None = None,
        keys: list[int] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        integration: str | None = None,
        embedded: bool | None = None,
        host_is_node: bool | None = None,
        limit: int | None = None,
        offset: int = 0,
        include_status: bool = False,
    ) -> Rack | list[Rack]:
        """Retrieves racks matching the given filters. Returns one rack when a key or
        name is given, and a list otherwise. A filter left as None does not filter.

        :param where: Keeps racks that match this filter tree.
        :param keys: Keeps racks with one of these keys.
        :param names: Keeps racks with one of these names.
        :param integration: Keeps racks that support this integration.
        :param embedded: Keeps racks that are, or are not, embedded in a Core.
        :param host_is_node: Keeps racks on the Core that serves the request, or racks
            on other Cores when False.
        :param search_term: A fuzzy search term matched against rack names.
        :param limit: The maximum number of racks to return.
        :param offset: The number of racks to skip.
        :param include_status: Whether to populate each rack's status.
        :raises ValueError: If two singular selectors are given, or one is
            combined with a list filter.
        :raises NotFoundError: If a single rack is requested and none matches.
        :raises MultipleFoundError: If more than one rack matches a single name.
        """
        is_single = not all_none(key, name)
        if sum(v is not None for v in (key, name)) > 1:
            raise ValueError("only one of key and name can be given")
        if is_single and (
            search_term
            or offset
            or not all_none(
                where, keys, names, integration, embedded, host_is_node, limit
            )
        ):
            raise ValueError(
                "key and name cannot be combined with list filters; use keys or "
                "names instead"
            )
        keys = override(key, keys)
        names = override(name, names)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
            integrations=(
                None if integration is None else filter.Contains[str](any=[integration])
            ),
            embedded=None if embedded is None else filter.Bool(eq=embedded),
            host_is_node=host_is_node,
        )
        req = RetrieveRequest(
            where=filter.merge(where, flat),
            search_term=search_term,
            limit=limit,
            offset=offset,
            include_status=include_status,
        )
        res = self._client.send("/rack/retrieve", req, _RetrieveResponse)
        if not is_single:
            return res.racks
        return one("Rack", key if key is not None else name, res.racks)

    def retrieve_embedded_rack(self) -> Rack:
        if self._embedded_rack is None:
            racks = self.retrieve(embedded=True, host_is_node=True, limit=1)
            if len(racks) == 0:
                raise NotFoundError("Embedded rack not found")
            self._embedded_rack = racks[0]
        return self._embedded_rack
