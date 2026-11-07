#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload
from uuid import UUID

from pydantic import BaseModel, Field

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax import group
from synnax.exceptions import NotFoundError
from synnax.rack.types_gen import Key, Rack, RetrieveRequest, Where, WhereInput
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


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


class _SetGroupRequest(BaseModel):
    keys: list[Key]
    group: UUID | None = None


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
    def create(
        self, *, key: int = 0, name: str = "", group: group.Key | None = None
    ) -> Rack: ...

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
        group: group.Key | None = None,
    ) -> Rack | list[Rack]:
        is_single = True
        if racks is None:
            racks = [Rack(key=key, name=name, group=group)]
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

    def set_group(self, keys: Key | list[Key], group: group.Key | None = None) -> None:
        """Moves racks into a group, or out of their group when group is None.

        :param keys: The key of the rack to move, or a list of keys.
        :param group: The key of the group to move the racks into.
        """
        req = _SetGroupRequest(keys=normalize(keys), group=group)
        self._client.send("/rack/set-group", req, Empty)

    @overload
    def retrieve(self, key: Key, *, include_status: bool = ...) -> Rack: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        embedded: filter.BoolInput = ...,
        integrations: filter.ContainsInput[str] = ...,
        host_is_node: bool = ...,
        groups: group.Key | list[group.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Rack: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        embedded: filter.BoolInput = ...,
        integrations: filter.ContainsInput[str] = ...,
        host_is_node: bool = ...,
        groups: group.Key | list[group.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Rack: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        embedded: filter.BoolInput = ...,
        integrations: filter.ContainsInput[str] = ...,
        host_is_node: bool = ...,
        groups: group.Key | list[group.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        integration: str | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: Literal[True],
    ) -> Rack: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        embedded: filter.BoolInput = ...,
        integrations: filter.ContainsInput[str] = ...,
        host_is_node: bool = ...,
        groups: group.Key | list[group.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        integration: str | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        offset: int = ...,
        include_status: bool = ...,
        one: Literal[False] = ...,
    ) -> list[Rack]: ...

    def retrieve(
        self,
        key: filter.SetInput[Key] | None = None,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        integration: str | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        include_status: bool = False,
        one: bool = False,
        **fields: Any,
    ) -> Rack | list[Rack]:
        """Retrieves racks from the Core. With no arguments, retrieves every rack.

        Each filter field takes a value, a list of values, or an operator, and a rack
        must match every field. A plain ``key``, by position or by keyword, or a plain
        ``name`` returns one rack.

        :param key: Keeps racks with this key.
        :param name: Keeps racks with this name.
        :param embedded: Keeps racks that are, or are not, embedded in a Core.
        :param integrations: Keeps racks that support this integration, or any
            integration in the list.
        :param host_is_node: Keeps racks on the Core that serves the request, or racks
            on other Cores when False.
        :param groups: Keeps racks in any of these groups.
        :param grouped: Keeps only racks in a group when True, and only racks in no
            group when False.
        :param and_: Keeps racks that match every filter in the list.
        :param or_: Keeps racks that match any filter in the list.
        :param not_: Keeps racks that do not match the filter.
        :param where: A filter tree that the racks must also match.
        :param keys: Keeps racks with one of these keys.
        :param names: Keeps racks with one of these names.
        :param integration: Keeps racks that support this integration.
        :param search_term: A fuzzy search term matched against rack names.
        :param limit: The maximum number of racks to retrieve.
        :param offset: The number of racks to skip.
        :param include_status: Whether to populate each rack's status.
        :param one: Returns one rack for any filter.
        :returns: One rack for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of racks. The list leaves out keys that do not exist.
        :raises NotFoundError: If one rack is wanted and none matches.
        :raises MultipleFoundError: If one rack is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one rack is wanted
            beside a limit or an offset.
        """
        if key is not None:
            fields["key"] = key
        single = returns_one(fields, one, limit, offset)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
            integrations=(
                None if integration is None else filter.Contains[str](any=[integration])
            ),
        )
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            offset=offset,
            include_status=include_status,
        )
        res = self._client.send("/rack/retrieve", req, _RetrieveResponse)
        return sole("rack", req, res.racks) if single else res.racks

    def retrieve_embedded_rack(self) -> Rack:
        if self._embedded_rack is None:
            racks = self.retrieve(embedded=True, host_is_node=True, limit=1)
            if len(racks) == 0:
                raise NotFoundError("Embedded rack not found")
            self._embedded_rack = racks[0]
        return self._embedded_rack
