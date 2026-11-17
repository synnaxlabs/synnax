#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload

from pydantic import BaseModel, Field

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax import group as group_
from synnax.exceptions import NotFoundError
from synnax.rack import actions_gen
from synnax.rack.types_gen import (
    Cursor,
    Key,
    Order,
    Rack,
    RetrieveRequest,
    Where,
    WhereInput,
)
from synnax.transport import send_batch
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    racks: list[Rack]


class _CreateResponse(BaseModel):
    racks: list[Rack]


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _DeleteRequest(BaseModel):
    keys: list[Key]


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
    def create(
        self, *, key: Key = 0, name: str = "", group: group_.Key | None = None
    ) -> Rack: ...

    @overload
    def create(self, racks: Rack) -> Rack: ...

    @overload
    def create(self, racks: list[Rack]) -> list[Rack]: ...

    def create(
        self,
        racks: Rack | list[Rack] | None = None,
        *,
        key: Key = 0,
        name: str = "",
        group: group_.Key | None = None,
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

    def delete(self, keys: list[Key]) -> None:
        req = _DeleteRequest(keys=keys)
        self._client.send("/rack/delete", req, Empty)

    def rename(self, key: Key, name: str) -> None:
        """Renames a rack.

        :param key: The key of the rack to rename.
        :param name: The new name for the rack.
        :raises NotFoundError: If no rack has the given key.
        """
        action = actions_gen.set(actions_gen.SetPayload(path=["name"], value=name))
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/rack/dispatch", req, Empty)
        if self._embedded_rack is not None and self._embedded_rack.key == key:
            self._embedded_rack = None

    def set_group(self, keys: Key | list[Key], group: group_.Key | None = None) -> None:
        """Moves racks into a group, or out of their group when group is None.

        :param keys: The key of the rack to move, or a list of keys.
        :param group: The key of the group to move the racks into.
        """
        value = None if group is None else str(group)
        action = actions_gen.set(actions_gen.SetPayload(path=["group"], value=value))
        send_batch(
            self._client,
            "/rack/dispatch",
            [_DispatchRequest(key=k, actions=[action]) for k in normalize(keys)],
        )

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
        groups: group_.Key | list[group_.Key] = ...,
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
        groups: group_.Key | list[group_.Key] = ...,
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
        groups: group_.Key | list[group_.Key] = ...,
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
        groups: group_.Key | list[group_.Key] = ...,
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
        order: Order | None = ...,
        after: Cursor | None = ...,
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
        order: Order | None = None,
        after: Cursor | None = None,
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
        :param order: The order of the racks. Defaults to key order.
        :param after: Retrieves the racks past this position in the order: the key of
            the last rack of the page before, and its value of the order field.
        :param include_status: Whether to populate each rack's status.
        :param one: Returns one rack for any filter.
        :returns: One rack for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of racks. The list leaves out keys that do not exist.
        :raises NotFoundError: If one rack is wanted and none matches.
        :raises MultipleFoundError: If one rack is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one rack is wanted
            beside a limit or a cursor.
        """
        if key is not None:
            fields["key"] = key
        single = returns_one(fields, one, limit=limit, after=after)
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
            order=order,
            after=after,
            include_status=include_status,
        )
        res = self._client.send("/rack/retrieve", req, _RetrieveResponse)
        return sole("rack", req, res.racks) if single else res.racks

    def retrieve_embedded_rack(self) -> Rack:
        if self._embedded_rack is None:
            racks = self.retrieve(embedded=True, limit=1)
            if len(racks) == 0:
                raise NotFoundError("Embedded rack not found")
            self._embedded_rack = racks[0]
        return self._embedded_rack
