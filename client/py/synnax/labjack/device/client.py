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
from synnax.labjack.device.types_gen import (
    Cursor,
    Device,
    Key,
    Model,
    RetrieveRequest,
    RetrieveResponse,
    Update,
    Where,
    WhereInput,
)
from synnax.rack import Key as RackKey
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    devices: list[Device]


class _CreateResponse(BaseModel):
    devices: list[Device]


class _UpdateRequest(BaseModel):
    key: Key
    update: Update


class _UpdateResponse(BaseModel):
    device: Device


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Client:
    """Creates, retrieves, updates, and deletes LabJack devices."""

    _client: UnaryClient

    def __init__(self, *, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, devices: Device) -> Device: ...

    @overload
    def create(self, devices: list[Device]) -> list[Device]: ...

    def create(self, devices: Device | list[Device]) -> Device | list[Device]:
        """Creates devices, or replaces the devices whose keys exist. The Core gives
        each new device an unknown status.

        :param devices: The devices to create or replace.
        :returns: The stored devices, without their statuses.
        :raises ValidationError: If a device is invalid, its rack does not exist, or it
            replaces a device of another model.
        """
        is_single = not isinstance(devices, list)
        req = _CreateRequest(devices=normalize(devices))
        res = self._client.send("/labjack/device/create", req, _CreateResponse)
        return res.devices[0] if is_single else res.devices

    def update(self, key: Key, update: Update) -> Device:
        """Writes one field group to a device and keeps the other group as stored.
        The update replaces the whole group, so a field it leaves out is cleared.

        :param key: The key of the device.
        :param update: The field group to write.
        :returns: The device as stored, without its status.
        :raises NotFoundError: If no device has the key.
        :raises ValidationError: If the update is invalid or changes the model.
        """
        req = _UpdateRequest(key=key, update=update)
        return self._client.send("/labjack/device/update", req, _UpdateResponse).device

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes devices with their statuses. Deleting a missing device succeeds.

        :param keys: The keys of the devices to delete.
        :raises ValidationError: If a task or channel still uses a device.
        """
        self._client.send(
            "/labjack/device/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    @overload
    def retrieve(self, key: Key, *, include_status: bool = ...) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        name: str,
        rack: filter.SetInput[RackKey] = ...,
        include_status: bool = ...,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        serial: filter.OrderedInput[int] = ...,
        model: filter.SetInput[Model] = ...,
        rack: filter.SetInput[RackKey] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: Literal[True],
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        serial: list[int] | filter.OrderedDict[int] | filter.Ordered[int] = ...,
        model: filter.SetInput[Model] = ...,
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
    ) -> list[Device]: ...

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
    ) -> Device | list[Device]:
        """Retrieves devices. With no arguments, retrieves every device.

        Each filter field takes a value, a list of values, or an operator, and a
        device must match every field. A plain ``key`` or ``name`` returns one device.

        :param key: Keeps devices with this key.
        :param name: Keeps devices with this name.
        :param serial: Keeps devices with this serial number.
        :param model: Keeps devices of this model.
        :param rack: Keeps devices on this rack.
        :param where: A filter tree that the devices must also match.
        :param keys: Keeps devices with one of these keys.
        :param search_term: Keeps the devices whose names match, closest match first.
        :param limit: The most devices to return.
        :param after: Retrieves the devices past this key.
        :param include_status: Whether to fill the status of each device.
        :param one: Returns one device for any filter.
        :returns: One device for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of devices.
        :raises NotFoundError: If one device is wanted and none matches.
        :raises MultipleFoundError: If one device is wanted and more than one matches.
        :raises ValidationError: If one device is wanted beside a limit or a cursor.
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
        res = self._client.send("/labjack/device/retrieve", req, RetrieveResponse)
        return sole("LabJack device", req, res.devices) if single else res.devices
