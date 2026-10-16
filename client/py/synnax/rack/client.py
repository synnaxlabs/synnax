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
from synnax.rack.types_gen import Rack
from x.lists import check_for_none, override


class _CreateRequest(BaseModel):
    racks: list[Rack]


class _CreateResponse(BaseModel):
    racks: list[Rack]


class _DeleteRequest(BaseModel):
    keys: list[int]


class _RetrieveRequest(BaseModel):
    keys: list[int] | None = None
    names: list[str] | None = None
    integration: str | None = None
    search_term: str | None = None
    embedded: bool | None = None
    host_is_node: bool | None = None
    offset: int | None = None
    limit: int | None = None
    include_status: bool = False


class _RetrieveResponse(BaseModel):
    racks: list[Rack] = Field(default_factory=list)


class Client:
    _client: UnaryClient
    _embedded_rack: Rack | None = None
    instrumentation: Instrumentation

    def __init__(
        self,
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

    # Each list overload requires one filter that makes the result a list. The last
    # overload covers calls that pass those filters as possibly None.
    @overload
    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        *,
        integration: str | None = None,
        embedded: bool | None = None,
        host_is_node: bool | None = None,
        include_status: bool = False,
    ) -> Rack: ...

    @overload
    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        *,
        keys: list[int],
        names: list[str] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        integration: str | None = None,
        embedded: bool | None = None,
        host_is_node: bool | None = None,
        include_status: bool = False,
    ) -> list[Rack]: ...

    @overload
    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        *,
        keys: list[int] | None = None,
        names: list[str],
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        integration: str | None = None,
        embedded: bool | None = None,
        host_is_node: bool | None = None,
        include_status: bool = False,
    ) -> list[Rack]: ...

    @overload
    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        *,
        keys: list[int] | None = None,
        names: list[str] | None = None,
        search_term: str,
        limit: int | None = None,
        offset: int | None = None,
        integration: str | None = None,
        embedded: bool | None = None,
        host_is_node: bool | None = None,
        include_status: bool = False,
    ) -> list[Rack]: ...

    @overload
    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        *,
        keys: list[int] | None = None,
        names: list[str] | None = None,
        search_term: str | None = None,
        limit: int,
        offset: int | None = None,
        integration: str | None = None,
        embedded: bool | None = None,
        host_is_node: bool | None = None,
        include_status: bool = False,
    ) -> list[Rack]: ...

    @overload
    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        *,
        keys: list[int] | None = None,
        names: list[str] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int,
        integration: str | None = None,
        embedded: bool | None = None,
        host_is_node: bool | None = None,
        include_status: bool = False,
    ) -> list[Rack]: ...

    @overload
    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        *,
        keys: list[int] | None = None,
        names: list[str] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        integration: str | None = None,
        embedded: bool | None = None,
        host_is_node: bool | None = None,
        include_status: bool = False,
    ) -> Rack | list[Rack]: ...

    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        keys: list[int] | None = None,
        names: list[str] | None = None,
        *,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
        integration: str | None = None,
        host_is_node: bool | None = None,
        embedded: bool | None = None,
        include_status: bool = False,
    ) -> Rack | list[Rack]:
        """Retrieves racks matching the given filters. Returns one rack unless keys,
        names, a search term, a limit, or an offset is given.

        :param search_term: A fuzzy search term matched against rack names.
        :param limit: The maximum number of racks to return.
        :param offset: The number of racks to skip.
        :param include_status: Whether to populate each rack's status.
        :raises NotFoundError: If a single rack is requested and none matches.
        """
        is_single = check_for_none(keys, names, search_term, limit, offset)
        res = self._client.send(
            "/rack/retrieve",
            _RetrieveRequest(
                keys=override(key, keys),
                names=override(name, names),
                search_term=search_term,
                limit=limit,
                offset=offset,
                integration=integration,
                host_is_node=host_is_node,
                embedded=embedded,
                include_status=include_status,
            ),
            _RetrieveResponse,
        )
        if is_single:
            if len(res.racks) > 0:
                return res.racks[0]
            raise NotFoundError("Rack not found")
        return res.racks

    def retrieve_embedded_rack(self) -> Rack:
        if self._embedded_rack is None:
            self._embedded_rack = self.retrieve(embedded=True, host_is_node=True)
        return self._embedded_rack
