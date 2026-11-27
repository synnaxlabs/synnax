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

from freighter import Empty, UnaryClient
from synnax.ni.chassis.types_gen import (
    Chassis,
    Cursor,
    Key,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Update,
    Where,
)
from synnax.util import request_where, returns_one, sole
from x.lists import normalize

SET_CHANNEL = "sy_ni_chassis_set"
DELETE_CHANNEL = "sy_ni_chassis_delete"


class _CreateRequest(BaseModel):
    chassis: list[Chassis]


class _CreateResponse(BaseModel):
    chassis: list[Chassis] = Field(default_factory=list)


class _UpdateRequest(BaseModel):
    key: Key
    update: Update


class _UpdateResponse(BaseModel):
    chassis: Chassis


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    """Creates, retrieves, updates, and deletes NI chassis in a Core."""

    _client: UnaryClient

    def __init__(self, *, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, chassis: Chassis) -> Chassis: ...

    @overload
    def create(self, chassis: list[Chassis]) -> list[Chassis]: ...

    def create(self, chassis: Chassis | list[Chassis]) -> Chassis | list[Chassis]:
        """Creates NI chassis.

        :param chassis: The chassis to create, or a list of chassis.
        :returns: The created chassis, or the list of created chassis.
        """
        res = self._client.send(
            "/ni/chassis/create",
            _CreateRequest(chassis=normalize(chassis)),
            _CreateResponse,
        )
        return res.chassis[0] if isinstance(chassis, Chassis) else res.chassis

    @overload
    def retrieve(self, key: Key, *, include_status: bool = ...) -> Chassis: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: Literal[True],
        **fields: Any,
    ) -> Chassis: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        include_status: bool = ...,
        one: Literal[False] = ...,
        **fields: Any,
    ) -> list[Chassis]: ...

    def retrieve(
        self,
        key: Key | None = None,
        *,
        where: Where | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        include_status: bool = False,
        one: bool = False,
        **fields: Any,
    ) -> Chassis | list[Chassis]:
        """Retrieves NI chassis. With no arguments, retrieves every chassis.

        Each filter field (``key``, ``name``, ``serial``, ``resource_name``, ``model``,
        and ``rack``) takes a value, a list of values, or an operator, and a chassis
        must match every field.

        :param key: Retrieves the one chassis with this key.
        :param where: A filter tree that the chassis must also match.
        :param search_term: A fuzzy search term matched against chassis names.
        :param limit: The maximum number of chassis to retrieve.
        :param order: The order of the chassis. Defaults to key order.
        :param after: Retrieves the chassis past this position in the order.
        :param include_status: Whether to populate the status of each chassis.
        :param one: Returns one chassis for any filter.
        :returns: One chassis for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of chassis.
        :raises NotFoundError: If one chassis is wanted and none matches.
        :raises MultipleFoundError: If one chassis is wanted and more than one matches.
        """
        if key is not None:
            fields["key"] = key
        single = returns_one(fields, one, limit=limit, after=after)
        req = RetrieveRequest(
            where=request_where(Where, where, fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
            include_status=include_status,
        )
        res = self._client.send("/ni/chassis/retrieve", req, RetrieveResponse)
        return sole("NI chassis", req, res.chassis) if single else res.chassis

    def update(self, key: Key, update: Update) -> Chassis:
        """Writes one field group of an NI chassis and keeps the other group as stored.
        A configuration update replaces the whole group, so a field it leaves out is
        cleared.

        :param key: The key of the chassis.
        :param update: The identity or configuration group to write.
        :returns: The chassis as stored after the update.
        :raises NotFoundError: If no chassis has the key.
        """
        req = _UpdateRequest(key=key, update=update)
        return self._client.send("/ni/chassis/update", req, _UpdateResponse).chassis

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes NI chassis.

        :param keys: The key of the chassis to delete, or a list of keys.
        """
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/ni/chassis/delete", req, Empty)
