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
from synnax.opcua.server.types_gen import (
    Cursor,
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Server,
    Update,
    Where,
    WhereInput,
)
from synnax.rack import Key as RackKey
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    servers: list[Server]


class _CreateResponse(BaseModel):
    servers: list[Server]


class _UpdateRequest(BaseModel):
    key: Key
    update: Update


class _UpdateResponse(BaseModel):
    server: Server


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    """Creates, retrieves, updates, and deletes OPC UA servers."""

    _client: UnaryClient

    def __init__(self, *, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, servers: Server) -> Server: ...

    @overload
    def create(self, servers: list[Server]) -> list[Server]: ...

    def create(self, servers: Server | list[Server]) -> Server | list[Server]:
        """Creates servers, or replaces the servers whose keys exist. The Core gives
        each new server an unknown status.

        :param servers: The servers to create or replace.
        :returns: The stored servers, without their statuses.
        :raises ValidationError: If a server is invalid, or if its rack, password, or
            certificates do not exist.
        """
        is_single = not isinstance(servers, list)
        req = _CreateRequest(servers=normalize(servers))
        res = self._client.send("/opcua/server/create", req, _CreateResponse)
        return res.servers[0] if is_single else res.servers

    def update(self, key: Key, update: Update) -> Server:
        """Writes one field group to a server and keeps the other group as stored.
        The update replaces the whole group, so a field it leaves out is cleared.

        :param key: The key of the server.
        :param update: The field group to write.
        :returns: The server as stored, without its status.
        :raises NotFoundError: If no server has the key.
        :raises ValidationError: If the update is invalid.
        """
        req = _UpdateRequest(key=key, update=update)
        return self._client.send("/opcua/server/update", req, _UpdateResponse).server

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes servers with their statuses. Deleting a missing server succeeds.

        :param keys: The keys of the servers to delete.
        """
        self._client.send(
            "/opcua/server/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    @overload
    def retrieve(self, key: Key, *, include_status: bool = ...) -> Server: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        rack: filter.SetInput[RackKey] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Server: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        rack: filter.SetInput[RackKey] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Server: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        rack: filter.SetInput[RackKey] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: Literal[True],
    ) -> Server: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        rack: filter.SetInput[RackKey] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        after: Cursor | None = ...,
        include_status: bool = ...,
        one: Literal[False] = ...,
    ) -> list[Server]: ...

    def retrieve(
        self,
        key: filter.SetInput[Key] | None = None,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        after: Cursor | None = None,
        include_status: bool = False,
        one: bool = False,
        **fields: Any,
    ) -> Server | list[Server]:
        """Retrieves servers. With no arguments, retrieves every server.

        Each filter field takes a value, a list of values, or an operator, and a
        server must match every field. A plain ``key`` or ``name`` returns one server.

        :param key: Keeps servers with this key.
        :param name: Keeps servers with this name.
        :param rack: Keeps servers on this rack.
        :param where: A filter tree that the servers must also match.
        :param keys: Keeps servers with one of these keys.
        :param search_term: Keeps the servers whose names match, closest match first.
        :param limit: The most servers to return.
        :param after: Retrieves the servers past this key.
        :param include_status: Whether to fill the status of each server.
        :param one: Returns one server for any filter.
        :returns: One server for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of servers.
        :raises NotFoundError: If one server is wanted and none matches.
        :raises MultipleFoundError: If one server is wanted and more than one matches.
        :raises ValidationError: If one server is wanted beside a limit or a cursor.
        """
        if key is not None:
            fields["key"] = key
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            after=after,
            include_status=include_status,
        )
        res = self._client.send("/opcua/server/retrieve", req, RetrieveResponse)
        return sole("OPC UA server", req, res.servers) if single else res.servers
