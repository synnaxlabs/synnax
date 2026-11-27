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
from synnax.labjack.device.actions_gen import Action
from synnax.labjack.device.types_gen import (
    Cursor,
    Device,
    Key,
    Model,
    RetrieveRequest,
    RetrieveResponse,
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


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[Action]


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

    def dispatch(self, key: Key, actions: Action | list[Action]) -> None:
        """Applies actions to a device, in order, in one transaction.

        :param key: The key of the device.
        :param actions: The actions to apply, such as ``labjack.device.set``. Paths
            take the snake_case form that the Core stores.
        :raises NotFoundError: If no device has the key.
        :raises ValidationError: If an action fails, or the result is invalid or
            changes the model.
        """
        req = _DispatchRequest(key=key, actions=normalize(actions))
        self._client.send("/labjack/device/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes devices with their statuses. Deleting a missing device succeeds.

        :param keys: The keys of the devices to delete.
        """
        self._client.send(
            "/labjack/device/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    @overload
    def retrieve(self, key: Key, /, *, include_status: bool = ...) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
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
        one: bool = ...,
    ) -> Device: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        serial: filter.OrderedInput[int] = ...,
        model: filter.SetInput[Model] = ...,
        rack: filter.SetInput[RackKey] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
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
        keys: list[Key] | None = ...,
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
        serial: filter.OrderedInput[int] = ...,
        model: filter.SetInput[Model] = ...,
        rack: filter.SetInput[RackKey] = ...,
        keys: list[Key] | None = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        after: Cursor | None = ...,
        include_status: bool = ...,
        one: Literal[False] = ...,
    ) -> list[Device]: ...

    def retrieve(
        self,
        *args: filter.SetInput[Key],
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
        device must match every field. A plain ``key``, by position or by keyword, or a
        plain ``name`` returns one device.

        :param key: Keeps devices with this key.
        :param name: Keeps devices with this name.
        :param serial: Keeps devices with this serial number.
        :param model: Keeps devices of this model.
        :param rack: Keeps devices on this rack.
        :param and_: Keeps devices that match every filter in the list.
        :param or_: Keeps devices that match any filter in the list.
        :param not_: Keeps devices that do not match the filter.
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
        :raises ValidationError: If a filter field is None, or if one device is wanted
            beside a limit or a cursor.
        """
        if args:
            (fields["key"],) = args
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
