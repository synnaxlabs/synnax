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
from synnax.ni.device.types_gen import (
    Cursor,
    Device,
    Key,
    Order,
    RetrieveRequest,
    RetrieveResponse,
    Update,
    Where,
)
from synnax.util import request_where, returns_one, sole
from x.lists import normalize

SET_CHANNEL = "sy_ni_device_set"
DELETE_CHANNEL = "sy_ni_device_delete"


class _CreateRequest(BaseModel):
    devices: list[Device]


class _CreateResponse(BaseModel):
    devices: list[Device] = Field(default_factory=list)


class _UpdateRequest(BaseModel):
    key: Key
    update: Update


class _UpdateResponse(BaseModel):
    device: Device


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    """Creates, retrieves, updates, and deletes NI devices in a Core."""

    _client: UnaryClient

    def __init__(self, *, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, devices: Device) -> Device: ...

    @overload
    def create(self, devices: list[Device]) -> list[Device]: ...

    def create(self, devices: Device | list[Device]) -> Device | list[Device]:
        """Creates NI devices.

        :param devices: The device to create, or a list of devices.
        :returns: The created device, or the list of created devices.
        """
        res = self._client.send(
            "/ni/device/create",
            _CreateRequest(devices=normalize(devices)),
            _CreateResponse,
        )
        return res.devices[0] if isinstance(devices, Device) else res.devices

    @overload
    def retrieve(self, key: Key, *, include_status: bool = ...) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        name: str,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        **fields: Any,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: Literal[True],
        **fields: Any,
    ) -> Device: ...

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
    ) -> list[Device]: ...

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
    ) -> Device | list[Device]:
        """Retrieves NI devices. With no arguments, retrieves every device.

        Each filter field (``key``, ``name``, ``serial``, ``resource_name``, ``model``,
        and ``rack``) takes a value, a list of values, or an operator, and a device must
        match every field.

        :param key: Retrieves the one device with this key.
        :param where: A filter tree that the devices must also match.
        :param search_term: A fuzzy search term matched against device names.
        :param limit: The maximum number of devices to retrieve.
        :param order: The order of the devices. Defaults to key order.
        :param after: Retrieves the devices past this position in the order.
        :param include_status: Whether to populate the status of each device.
        :param one: Returns one device for any filter.
        :returns: One device for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of devices.
        :raises NotFoundError: If one device is wanted and none matches.
        :raises MultipleFoundError: If one device is wanted and more than one matches.
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
        res = self._client.send("/ni/device/retrieve", req, RetrieveResponse)
        return sole("NI device", req, res.devices) if single else res.devices

    def update(self, key: Key, update: Update) -> Device:
        """Writes one field group of an NI device and keeps the other group as stored.
        The update replaces the whole group, so a field it leaves out is cleared, such
        as the chassis of an identity update.

        :param key: The key of the device.
        :param update: The identity or configuration group to write.
        :returns: The device as stored after the update.
        :raises NotFoundError: If no device has the key.
        """
        req = _UpdateRequest(key=key, update=update)
        return self._client.send("/ni/device/update", req, _UpdateResponse).device

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes NI devices.

        :param keys: The key of the device to delete, or a list of keys.
        """
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/ni/device/delete", req, Empty)
